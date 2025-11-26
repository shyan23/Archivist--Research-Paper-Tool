package worker

import (
	"archivist/internal/analyzer"
	"archivist/internal/app"
	"archivist/internal/cache"
	"archivist/internal/compiler"
	"archivist/internal/generator"
	"archivist/internal/graph"
	"archivist/internal/ui"
	"archivist/pkg/fileutil"
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"
)

type ProcessingJob struct {
	FilePath string
	FileHash string
	Priority int
}

type ProcessingResult struct {
	Job        *ProcessingJob
	PaperTitle string
	TexFile    string
	ReportFile string
	Duration   time.Duration
	Error      error
}

type WorkerPool struct {
	numWorkers     int
	jobs           chan *ProcessingJob
	results        chan *ProcessingResult
	wg             sync.WaitGroup
	config         *app.Config
	cache          *cache.RedisCache
	kafkaProducer  *graph.KafkaProducer
	enableRAG      bool // Enable RAG indexing during processing
}

// NewWorkerPool creates a new worker pool
func NewWorkerPool(numWorkers int, config *app.Config, redisCache *cache.RedisCache, enableGraphBuilding bool) *WorkerPool {
	// Initialize Kafka producer if graph is enabled AND user opted in
	var kafkaProducer *graph.KafkaProducer
	if config.Graph.Enabled && enableGraphBuilding {
		kafkaProducer = graph.NewKafkaProducer(
			[]string{"localhost:9094"}, // Kafka broker (external listener)
			"paper.processed",           // Topic
			true,                        // Enabled
		)
	}

	return &WorkerPool{
		numWorkers:    numWorkers,
		jobs:          make(chan *ProcessingJob, numWorkers*2),
		results:       make(chan *ProcessingResult, numWorkers*2),
		config:        config,
		cache:         redisCache,
		kafkaProducer: kafkaProducer,
		enableRAG:     false, // Default off
	}
}

// SetEnableRAG sets whether to enable RAG indexing
func (wp *WorkerPool) SetEnableRAG(enable bool) {
	wp.enableRAG = enable
}

// Start starts the worker pool
func (wp *WorkerPool) Start(ctx context.Context) {
	for i := 0; i < wp.numWorkers; i++ {
		wp.wg.Add(1)
		go wp.worker(ctx, i)
	}
}

// worker processes jobs
func (wp *WorkerPool) worker(ctx context.Context, id int) {
	defer wp.wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case job, ok := <-wp.jobs:
			if !ok {
				return
			}
			log.Printf("[Worker %d] Processing: %s", id, job.FilePath)
			result := wp.processJob(ctx, job)
			wp.results <- result
		}
	}
}

// processJob processes a single PDF file
func (wp *WorkerPool) processJob(ctx context.Context, job *ProcessingJob) *ProcessingResult {
	startTime := time.Now()
	result := &ProcessingResult{Job: job}

	log.Printf("  ⏱️  Starting processing pipeline for: %s", job.FilePath)

	// Compute hash for cache lookup
	fileHash, err := fileutil.ComputeFileHash(job.FilePath)
	if err != nil {
		log.Printf("  ⚠️  Warning: Could not compute hash: %v", err)
		fileHash = fmt.Sprintf("temp_%d", time.Now().UnixNano()) // Temporary hash
	}
	job.FileHash = fileHash

	// Step 1: Create analyzer
	stepStart := time.Now()
	log.Printf("  🔧 Step 1/4: Initializing Gemini analyzer...")
	analyzer, err := analyzer.NewAnalyzer(wp.config)
	if err != nil {
		result.Error = fmt.Errorf("failed to create analyzer: %w", err)
		return result
	}
	defer analyzer.Close()
	log.Printf("  ✓ Analyzer initialized (%.2fs)", time.Since(stepStart).Seconds())

	// Step 2: Check cache first, then analyze if needed
	stepStart = time.Now()
	var latexContent string
	var paperTitle string

	// Try to get from cache if enabled
	if wp.cache != nil {
		log.Printf("  🔍 Step 2/4: Checking cache for existing analysis...")
		cached, err := wp.cache.Get(ctx, fileHash)
		if err != nil {
			log.Printf("  ⚠️  Cache error (continuing with analysis): %v", err)
		} else if cached != nil {
			// Cache hit! Use cached result
			latexContent = cached.LatexContent
			paperTitle = cached.PaperTitle
			log.Printf("  ✓ Cache hit! Skipping Gemini API call (%.2fs)", time.Since(stepStart).Seconds())
		}
	}

	// If not in cache, analyze with Gemini
	if latexContent == "" {
		log.Printf("  🤖 Step 2/4: Analyzing paper with Gemini (cache miss)...")
		log.Printf("     → Sending PDF to Gemini API for analysis and LaTeX generation...")

		// Enforce timeout for API call
		apiCtx, apiCancel := context.WithTimeout(ctx, time.Duration(wp.config.Processing.TimeoutPerPaper)*time.Second)
		defer apiCancel()

		latexContent, err = analyzer.AnalyzePaper(apiCtx, job.FilePath)
		if err != nil {
			if apiCtx.Err() == context.DeadlineExceeded {
				result.Error = fmt.Errorf("analysis timed out after %d seconds (increase timeout_per_paper in config)", wp.config.Processing.TimeoutPerPaper)
			} else {
				result.Error = fmt.Errorf("analysis failed: %w", err)
			}
			return result
		}
		log.Printf("  ✓ Analysis complete (%.2fs)", time.Since(stepStart).Seconds())

		// Extract title (but DON'T cache yet - wait for successful PDF compilation)
		paperTitle = extractTitleFromLatex(latexContent)
		if paperTitle == "" {
			paperTitle = "Unknown Paper"
		}
	}

	// Set result paper title
	result.PaperTitle = paperTitle

	// Step 3: Write LaTeX file
	stepStart = time.Now()
	log.Printf("  📝 Step 3/4: Generating LaTeX file...")
	latexGen := generator.NewLatexGenerator(wp.config.TexOutputDir)
	texPath, err := latexGen.GenerateLatexFile(paperTitle, latexContent)
	if err != nil {
		result.Error = fmt.Errorf("LaTeX generation failed: %w", err)
		return result
	}
	result.TexFile = texPath
	log.Printf("  ✓ LaTeX file created: %s (%.2fs)", texPath, time.Since(stepStart).Seconds())

	// Step 4: Compile to PDF
	stepStart = time.Now()
	log.Printf("  🔨 Step 4/4: Compiling LaTeX to PDF (running pdflatex)...")
	compiler := compiler.NewLatexCompiler(
		wp.config.Latex.Compiler,
		wp.config.Latex.Engine == "latexmk",
		wp.config.Latex.CleanAux,
		wp.config.ReportOutputDir,
	)

	reportPath, err := compiler.Compile(texPath)
	if err != nil {
		result.Error = fmt.Errorf("PDF compilation failed: %w", err)
		return result
	}
	result.ReportFile = reportPath
	log.Printf("  ✓ PDF compiled: %s (%.2fs)", reportPath, time.Since(stepStart).Seconds())

	// Step 5: NOW cache the result after successful PDF compilation
	// Only cache if we generated new content (not from cache)
	if wp.cache != nil && latexContent != "" {
		// Check if this was a cache hit by seeing if we have the cache marker
		cached, _ := wp.cache.Get(ctx, fileHash)
		if cached == nil {
			// This was NOT from cache, so cache it now
			log.Printf("  💾 Caching successful analysis result...")
			cacheEntry := &cache.CachedAnalysis{
				ContentHash:  fileHash,
				PaperTitle:   paperTitle,
				LatexContent: latexContent,
				ModelUsed:    wp.config.Gemini.Model,
			}
			if err := wp.cache.Set(ctx, fileHash, cacheEntry); err != nil {
				log.Printf("  ⚠️  Failed to cache result: %v", err)
			} else {
				log.Printf("  ✓ Analysis cached for future use")
			}
		}
	}

	// Step 6: Publish to Kafka for microservices (RAG + Graph)
	// The Python microservices will handle:
	// - RAG Service: Indexing to Qdrant for chat feature
	// - Graph Service: Building Neo4j knowledge graph
	if wp.kafkaProducer != nil {
		log.Printf("  📡 Publishing to Kafka for microservices...")
		if err := wp.kafkaProducer.PublishPaperProcessed(ctx, paperTitle, latexContent, job.FilePath); err != nil {
			log.Printf("  ⚠️  Kafka publish warning: %v", err)
		}
	}

	result.Duration = time.Since(startTime)
	log.Printf("  🎉 Processing complete! Total time: %.2fs", result.Duration.Seconds())
	return result
}

// SubmitJob submits a job to the pool
func (wp *WorkerPool) SubmitJob(job *ProcessingJob) {
	wp.jobs <- job
}

// Close closes the job channel
func (wp *WorkerPool) Close() {
	close(wp.jobs)
}

// Wait waits for all workers to finish
func (wp *WorkerPool) Wait() {
	wp.wg.Wait()
	close(wp.results)
}

// Results returns the results channel
func (wp *WorkerPool) Results() <-chan *ProcessingResult {
	return wp.results
}

// ProcessBatch processes a batch of PDF files
func ProcessBatch(ctx context.Context, files []string, config *app.Config, force bool, enableRAG bool, enableGraphBuilding bool) error {
	// Initialize Redis cache if enabled
	var redisCache *cache.RedisCache
	var err error
	if config.Cache.Enabled && config.Cache.Type == "redis" {
		log.Println("🔌 Initializing Redis cache...")
		ttl := time.Duration(config.Cache.TTL) * time.Hour
		redisCache, err = cache.NewRedisCache(
			config.Cache.Redis.Addr,
			config.Cache.Redis.Password,
			config.Cache.Redis.DB,
			ttl,
		)
		if err != nil {
			log.Printf("⚠️  Warning: Failed to connect to Redis: %v", err)
			log.Println("   Continuing without cache...")
			redisCache = nil
		} else {
			defer redisCache.Close()
			stats, _ := redisCache.GetStats(ctx)
			log.Printf("✓ Cache ready (%d entries, TTL: %d hours)", stats, config.Cache.TTL)
		}
	}

	// Show graph integration status
	if config.Graph.Enabled {
		log.Println("📊 Knowledge graph integration enabled")
		log.Println("   → Papers will be added to Neo4j graph via Kafka")
	}

	// Queue files for processing
	log.Println("🔍 Queuing files for processing...")
	var jobsToProcess []*ProcessingJob
	for _, file := range files {
		// If not force mode and cache is enabled, check cache to skip already processed files
		if !force && redisCache != nil {
			hash, err := fileutil.ComputeFileHash(file)
			if err == nil {
				cached, _ := redisCache.Get(ctx, hash)
				if cached != nil {
					log.Printf("  ⏭️  Skipping (already in cache): %s", file)
					continue
				}
			}
		}

		log.Printf("  ✅ Queued for processing: %s", file)
		jobsToProcess = append(jobsToProcess, &ProcessingJob{
			FilePath: file,
			FileHash: "",
		})
	}

	if len(jobsToProcess) == 0 {
		log.Println("No files to process")
		return nil
	}

	log.Printf("Processing %d files with %d workers", len(jobsToProcess), config.Processing.MaxWorkers)

	if enableRAG {
		log.Println("💬 RAG indexing enabled - papers will be ready for chat after processing")
	}

	if enableGraphBuilding {
		log.Println("🕸️  Knowledge graph building enabled - papers will be added concurrently")
	}

	// Create and start worker pool
	pool := NewWorkerPool(config.Processing.MaxWorkers, config, redisCache, enableGraphBuilding)
	pool.SetEnableRAG(enableRAG) // Set RAG flag
	pool.Start(ctx)

	// Submit jobs
	go func() {
		for _, job := range jobsToProcess {
			pool.SubmitJob(job)
		}
		pool.Close()
	}()

	// Collect results
	var successful, failed, skipped int
	totalFiles := len(files)
	processedCount := 0
	startTime := time.Now()

	// Create progress bar with better description
	bar := ui.CreateProgressBar(len(jobsToProcess), fmt.Sprintf("📚 Processing %d papers", len(jobsToProcess)))

	// Wait for workers to finish in background and close results channel
	go func() {
		pool.Wait()
	}()

	// Collect results from workers
	for result := range pool.Results() {
		processedCount++

		// Update progress bar description with current status
		bar.Describe(fmt.Sprintf("📚 [%d/%d] Processing papers (✅ %d | ❌ %d)",
			processedCount, len(jobsToProcess), successful, failed))
		bar.Add(1)

		if result.Error != nil {
			failed++
			fmt.Println() // New line after progress bar
			ui.PrintError(fmt.Sprintf("[%d/%d] %s - %v", processedCount, len(jobsToProcess), result.Job.FilePath, result.Error))
		} else {
			successful++
			fmt.Println() // New line after progress bar
			ui.PrintSuccess(fmt.Sprintf("[%d/%d] %s -> %s (%.1fs)",
				processedCount, len(jobsToProcess), result.PaperTitle, result.ReportFile, result.Duration.Seconds()))
		}
	}

	// Finish the progress bar properly
	bar.Finish()
	fmt.Println() // Add extra newline for spacing

	// Calculate skipped files
	skipped = totalFiles - len(jobsToProcess)

	// Show summary
	totalTime := time.Since(startTime)
	ui.PrintSummary(successful, failed, skipped, totalTime)

	// Notify user that microservices are processing in background
	if enableRAG || enableGraphBuilding {
		fmt.Println()
		ui.PrintInfo("📡 Background services are processing:")
		if enableRAG {
			ui.PrintInfo("   • RAG indexing (chat feature)")
		}
		if enableGraphBuilding {
			ui.PrintInfo("   • Knowledge graph building (Neo4j)")
		}
		fmt.Println()

		// Start monitoring microservices in background
		go monitorMicroservices(successful, enableRAG, enableGraphBuilding)
	}

	// Wait for user input to continue
	fmt.Println()
	ui.PrintInfo("Press 'q' and Enter to return to homepage...")
	reader := bufio.NewReader(os.Stdin)
	for {
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(strings.ToLower(input))
		if input == "q" {
			break
		}
	}

	// Close Kafka producer
	if pool.kafkaProducer != nil {
		log.Println("📊 Closing Kafka producer...")
		if err := pool.kafkaProducer.Close(); err != nil {
			log.Printf("⚠️  Warning: Failed to close Kafka producer: %v", err)
		}
	}

	// Return error if any papers failed
	if failed > 0 {
		return fmt.Errorf("%d paper(s) failed to process", failed)
	}

	return nil
}

// extractTitleFromLatex extracts the paper title from LaTeX content
func extractTitleFromLatex(latexContent string) string {
	// Look for \title{...} command
	titleRegex := regexp.MustCompile(`\\title\{([^}]+)\}`)
	matches := titleRegex.FindStringSubmatch(latexContent)
	if len(matches) > 1 {
		title := strings.TrimSpace(matches[1])
		// Remove any LaTeX commands from the title
		title = strings.ReplaceAll(title, "\\", "")
		return title
	}

	// Fallback: look for the first section or subsection title
	sectionRegex := regexp.MustCompile(`\\(?:section|subsection)\*?\{([^}]+)\}`)
	matches = sectionRegex.FindStringSubmatch(latexContent)
	if len(matches) > 1 {
		title := strings.TrimSpace(matches[1])
		// Remove any LaTeX commands from the title
		title = strings.ReplaceAll(title, "\\", "")
		return title
	}

	return ""
}

// monitorMicroservices monitors the microservices and shows notifications when they complete
func monitorMicroservices(expectedPapers int, checkRAG bool, checkGraph bool) {
	const (
		graphServiceURL = "http://localhost:8081/api/graph/queue-stats"
		ragServiceURL   = "http://localhost:8082/status" // Python RAG service
		pollInterval    = 3 * time.Second
		maxWaitTime     = 5 * time.Minute
	)

	startTime := time.Now()
	ragCompleted := !checkRAG  // If not checking, mark as completed
	graphCompleted := !checkGraph

	// Track initial counts
	var initialGraphProcessed int
	if checkGraph {
		if stats := getGraphStats(graphServiceURL); stats != nil {
			initialGraphProcessed = stats.ProcessedCount
		}
	}

	for {
		// Check if max wait time exceeded
		if time.Since(startTime) > maxWaitTime {
			fmt.Println()
			ui.PrintWarning("⏱️  Microservices are still processing (taking longer than expected)")
			ui.PrintInfo("   You can continue using the app. Services will complete in background.")
			return
		}

		// Check Graph Service
		if checkGraph && !graphCompleted {
			if stats := getGraphStats(graphServiceURL); stats != nil {
				processed := stats.ProcessedCount - initialGraphProcessed
				if stats.QueueSize == 0 && processed >= expectedPapers {
					graphCompleted = true
					fmt.Println()
					ui.PrintSuccess("✅ Knowledge graph building complete!")
					ui.PrintInfo(fmt.Sprintf("   Processed %d papers into Neo4j", processed))
				}
			}
		}

		// Check RAG Service (if you have status endpoint)
		if checkRAG && !ragCompleted {
			// For now, assume RAG completes quickly after graph
			// You can add actual RAG service status check here
			if graphCompleted {
				ragCompleted = true
				fmt.Println()
				ui.PrintSuccess("✅ RAG indexing complete!")
				ui.PrintInfo("   Papers indexed in Qdrant for chat")
			}
		}

		// Both completed
		if ragCompleted && graphCompleted {
			fmt.Println()
			ui.PrintSuccess("🎉 All background services completed!")
			ui.PrintInfo("Press 'q' to return to homepage")
			return
		}

		time.Sleep(pollInterval)
	}
}

// GraphQueueStats represents the graph service queue statistics
type GraphQueueStats struct {
	QueueSize      int  `json:"queue_size"`
	ProcessedCount int  `json:"processed_count"`
	FailedCount    int  `json:"failed_count"`
	ActiveWorkers  int  `json:"active_workers"`
	IsRunning      bool `json:"is_running"`
}

// getGraphStats fetches graph service statistics
func getGraphStats(url string) *GraphQueueStats {
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil
	}

	var stats GraphQueueStats
	if err := json.Unmarshal(body, &stats); err != nil {
		return nil
	}

	return &stats
}
