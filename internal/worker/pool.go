package worker

import (
	"archivist/internal/analyzer"
	"archivist/internal/app"
	"archivist/internal/compiler"
	"archivist/internal/generator"
	"archivist/internal/parser"
	"archivist/internal/storage"
	"archivist/pkg/fileutil"
	"context"
	"fmt"
	"log"
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

	// Mark as processing
	wp.metadata.MarkProcessing(job.FileHash, job.FilePath)

	// Step 1: Create analyzer
	analyzer, err := analyzer.NewAnalyzer(wp.config)
	if err != nil {
		result.Error = fmt.Errorf("failed to create analyzer: %w", err)
		wp.metadata.MarkFailed(job.FileHash, result.Error.Error())
		return result
	}
	defer analyzer.Close()

	// Step 2: Extract metadata for title
	pdfParser := parser.NewPDFParser(analyzer.GetClient())
	metadata, err := pdfParser.ExtractMetadata(ctx, job.FilePath)
	if err != nil {
		result.Error = fmt.Errorf("failed to extract metadata: %w", err)
		wp.metadata.MarkFailed(job.FileHash, result.Error.Error())
		return result
	}
	result.PaperTitle = metadata.Title

	// Step 3: Analyze paper and generate LaTeX
	latexContent, err := analyzer.AnalyzePaper(ctx, job.FilePath)
	if err != nil {
		result.Error = fmt.Errorf("analysis failed: %w", err)
		wp.metadata.MarkFailed(job.FileHash, result.Error.Error())
		return result
	}

	// Step 4: Write LaTeX file
	latexGen := generator.NewLatexGenerator(wp.config.TexOutputDir)
	texPath, err := latexGen.GenerateLatexFile(metadata.Title, latexContent)
	if err != nil {
		result.Error = fmt.Errorf("LaTeX generation failed: %w", err)
		wp.metadata.MarkFailed(job.FileHash, result.Error.Error())
		return result
	}
	result.TexFile = texPath

	// Step 5: Compile to PDF
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

	// Step 6: Mark as completed
	wp.metadata.MarkCompleted(storage.ProcessingRecord{
		FilePath:    job.FilePath,
		FileHash:    job.FileHash,
		PaperTitle:  metadata.Title,
		TexFilePath: texPath,
		ReportPath:  reportPath,
	})

	result.Duration = time.Since(startTime)
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

	// Filter already processed files
	var jobsToProcess []*ProcessingJob
	for _, file := range files {
		hash, err := fileutil.ComputeFileHash(file)
		if err != nil {
			log.Printf("Error hashing %s: %v", file, err)
			continue
		}

		if !force && metadataStore.IsProcessed(hash) {
			log.Printf("Skipping already processed: %s", file)
			continue
		}

		jobsToProcess = append(jobsToProcess, &ProcessingJob{
			FilePath: file,
			FileHash: hash,
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
	var successful, failed int
	for result := range pool.Results() {
		if result.Error != nil {
			failed++
			log.Printf("❌ Failed: %s - %v", result.Job.FilePath, result.Error)
		} else {
			successful++
			log.Printf("✅ Success: %s -> %s (%.2fs)", result.PaperTitle, result.ReportFile, result.Duration.Seconds())
		}
	}

	pool.Wait()

	log.Printf("\n📊 Batch complete: %d successful, %d failed", successful, failed)
	return nil
}
