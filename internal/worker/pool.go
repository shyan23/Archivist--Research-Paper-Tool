package worker

import (
	"archivist/internal/analyzer"
	"archivist/internal/app"
	"archivist/internal/compiler"
	"archivist/internal/generator"
	"archivist/internal/storage"
	"archivist/internal/ui"
	"archivist/pkg/fileutil"
	"context"
	"fmt"
	"log"
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
	numWorkers int
	jobs       chan *ProcessingJob
	results    chan *ProcessingResult
	wg         sync.WaitGroup
	config     *app.Config
	metadata   *storage.MetadataStore
}

// NewWorkerPool creates a new worker pool
func NewWorkerPool(numWorkers int, config *app.Config, metadata *storage.MetadataStore) *WorkerPool {
	return &WorkerPool{
		numWorkers: numWorkers,
		jobs:       make(chan *ProcessingJob, numWorkers*2),
		results:    make(chan *ProcessingResult, numWorkers*2),
		config:     config,
		metadata:   metadata,
	}
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

	// Compute hash first for tracking (but don't let failures block processing)
	fileHash, err := fileutil.ComputeFileHash(job.FilePath)
	if err != nil {
		log.Printf("  ⚠️  Warning: Could not compute hash initially: %v", err)
		fileHash = fmt.Sprintf("temp_%d", time.Now().UnixNano()) // Temporary hash
	}
	job.FileHash = fileHash

	// Mark as processing
	wp.metadata.MarkProcessing(fileHash, job.FilePath)

	// Step 1: Create analyzer
	stepStart := time.Now()
	log.Printf("  🔧 Step 1/4: Initializing Gemini analyzer...")
	analyzer, err := analyzer.NewAnalyzer(wp.config)
	if err != nil {
		result.Error = fmt.Errorf("failed to create analyzer: %w", err)
		wp.metadata.MarkFailed(job.FileHash, result.Error.Error())
		return result
	}
	defer analyzer.Close()
	log.Printf("  ✓ Analyzer initialized (%.2fs)", time.Since(stepStart).Seconds())

	// Step 2: Analyze paper and generate LaTeX (single Gemini API call)
	stepStart = time.Now()
	log.Printf("  🤖 Step 2/4: Analyzing paper with Gemini (single API call)...")
	log.Printf("     → Sending PDF to Gemini API for analysis and LaTeX generation...")
	latexContent, err := analyzer.AnalyzePaper(ctx, job.FilePath)
	if err != nil {
		result.Error = fmt.Errorf("analysis failed: %w", err)
		wp.metadata.MarkFailed(job.FileHash, result.Error.Error())
		return result
	}

	// Extract title from LaTeX content using string parsing
	paperTitle := extractTitleFromLatex(latexContent)
	if paperTitle == "" {
		paperTitle = "Unknown Paper" // Fallback title
	}
	result.PaperTitle = paperTitle
	log.Printf("  ✓ Analysis complete, title extracted: \"%s\" (%.2fs)", paperTitle, time.Since(stepStart).Seconds())

	// Step 3: Write LaTeX file
	stepStart = time.Now()
	log.Printf("  📝 Step 3/4: Generating LaTeX file...")
	latexGen := generator.NewLatexGenerator(wp.config.TexOutputDir)
	texPath, err := latexGen.GenerateLatexFile(paperTitle, latexContent)
	if err != nil {
		result.Error = fmt.Errorf("LaTeX generation failed: %w", err)
		wp.metadata.MarkFailed(job.FileHash, result.Error.Error())
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
		wp.metadata.MarkFailed(job.FileHash, result.Error.Error())
		return result
	}
	result.ReportFile = reportPath
	log.Printf("  ✓ PDF compiled: %s (%.2fs)", reportPath, time.Since(stepStart).Seconds())

	// Step 6: Compute final hash after successful processing
	log.Printf("  🔐 Computing final file hash...")
	finalHash, err := fileutil.ComputeFileHash(job.FilePath)
	if err != nil {
		log.Printf("  ⚠️  Warning: Could not compute final hash, using initial: %v", err)
		finalHash = job.FileHash // Use initial hash if final computation fails
	} else {
		job.FileHash = finalHash // Update with final hash
	}

	// Step 5: Mark as completed
	log.Printf("  💾 Saving metadata and marking as complete...")
	wp.metadata.MarkCompleted(storage.ProcessingRecord{
		FilePath:    job.FilePath,
		FileHash:    job.FileHash,
		PaperTitle:  paperTitle,
		TexFilePath: texPath,
		ReportPath:  reportPath,
	})

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
func ProcessBatch(ctx context.Context, files []string, config *app.Config, force bool) error {
	// Initialize metadata store
	metadataStore, err := storage.NewMetadataStore(config.MetadataDir)
	if err != nil {
		return fmt.Errorf("failed to initialize metadata store: %w", err)
	}

	// Queue files for processing (hash will be computed after successful processing)
	log.Println("🔍 Queuing files for processing...")
	var jobsToProcess []*ProcessingJob
	for _, file := range files {
		// Skip hash check if force flag is set
		if !force {
			// Quick check: try to compute hash to see if already processed
			// But don't fail if hash computation fails - just process anyway
			hash, err := fileutil.ComputeFileHash(file)
			if err == nil && metadataStore.IsProcessed(hash) {
				log.Printf("  ⏭️  Skipping (already processed): %s", file)
				continue
			}
		}

		log.Printf("  ✅ Queued for processing: %s", file)
		jobsToProcess = append(jobsToProcess, &ProcessingJob{
			FilePath: file,
			FileHash: "", // Will be computed after successful processing
		})
	}

	if len(jobsToProcess) == 0 {
		log.Println("No files to process")
		return nil
	}

	log.Printf("Processing %d files with %d workers", len(jobsToProcess), config.Processing.MaxWorkers)

	// Create and start worker pool
	pool := NewWorkerPool(config.Processing.MaxWorkers, config, metadataStore)
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

	// Create progress bar
	bar := ui.CreateProgressBar(len(jobsToProcess), "Processing papers")

	for result := range pool.Results() {
		processedCount++
		bar.Add(1)

		if result.Error != nil {
			failed++
			fmt.Println() // New line after progress bar
			ui.PrintError(fmt.Sprintf("%s - %v", result.Job.FilePath, result.Error))
		} else {
			successful++
			fmt.Println() // New line after progress bar
			ui.PrintSuccess(fmt.Sprintf("%s -> %s (%.1fs)", result.PaperTitle, result.ReportFile, result.Duration.Seconds()))
		}
	}

	pool.Wait()

	// Calculate skipped files
	skipped = totalFiles - len(jobsToProcess)

	// Show summary
	totalTime := time.Since(startTime)
	ui.PrintSummary(successful, failed, skipped, totalTime)

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
