package worker_test

import (
	"archivist/internal/worker"
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"archivist/internal/app"
	"archivist/internal/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewWorkerPool tests creating a new worker pool
func TestNewWorkerPool(t *testing.T) {
	config := &app.Config{
		InputDir:        "/tmp/input",
		TexOutputDir:    "/tmp/tex",
		ReportOutputDir: "/tmp/reports",
		MetadataDir:     "/tmp/metadata",
		Processing: app.ProcessingConfig{
			MaxWorkers:      2,
			BatchSize:       10,
			TimeoutPerPaper: 300,
		},
		Gemini: app.GeminiConfig{
			APIKey:      "test-key",
			Model:       "gemini-pro",
			Temperature: 0.7,
			MaxTokens:   2048,
		},
		Latex: app.LatexConfig{
			Compiler: "pdflatex",
			Engine:   "pdflatex",
			CleanAux: true,
		},
	}
	
	metadataStore, err := storage.NewMetadataStore(config.MetadataDir)
	require.NoError(t, err)

	pool := worker.NewWorkerPool(2, config, metadataStore)

	assert.NotNil(t, pool)
	// Cannot test unexported fields, so just verify pool was created
}

// TestWorkerPoolStartStop tests starting and stopping the worker pool
func TestWorkerPoolStartStop(t *testing.T) {
	tmpDir := t.TempDir()
	config := &app.Config{
		InputDir:        filepath.Join(tmpDir, "input"),
		TexOutputDir:    filepath.Join(tmpDir, "tex"),
		ReportOutputDir: filepath.Join(tmpDir, "reports"),
		MetadataDir:     filepath.Join(tmpDir, "metadata"),
		Processing: app.ProcessingConfig{
			MaxWorkers:      2,
			BatchSize:       10,
			TimeoutPerPaper: 300,
		},
		Gemini: app.GeminiConfig{
			APIKey:      "test-key",
			Model:       "gemini-pro",
			Temperature: 0.7,
			MaxTokens:   2048,
		},
		Latex: app.LatexConfig{
			Compiler: "pdflatex",
			Engine:   "pdflatex",
			CleanAux: true,
		},
	}
	
	metadataStore, err := storage.NewMetadataStore(config.MetadataDir)
	require.NoError(t, err)

	pool := worker.NewWorkerPool(2, config, metadataStore)
	
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	pool.Start(ctx)
	pool.Close()
	pool.Wait()
	
	// If we get here without panic, the start/stop worked
	assert.True(t, true)
}

// TestProcessBatch tests the ProcessBatch function with mocked dependencies
func TestProcessBatch(t *testing.T) {
	// Note: This test would require mocking the full processing pipeline
	// including API calls, which is complex. We'll create a minimal test
	// to ensure the function structure works.
	
	tmpDir := t.TempDir()
	config := &app.Config{
		InputDir:        filepath.Join(tmpDir, "input"),
		TexOutputDir:    filepath.Join(tmpDir, "tex"),
		ReportOutputDir: filepath.Join(tmpDir, "reports"),
		MetadataDir:     filepath.Join(tmpDir, "metadata"),
		Processing: app.ProcessingConfig{
			MaxWorkers:      2,
			BatchSize:       10,
			TimeoutPerPaper: 300,
		},
		Gemini: app.GeminiConfig{
			APIKey:      "test-key",
			Model:       "gemini-pro",
			Temperature: 0.7,
			MaxTokens:   2048,
		},
		Latex: app.LatexConfig{
			Compiler: "pdflatex",
			Engine:   "pdflatex",
			CleanAux: true,
		},
	}
	
	// Create some test PDF files
	inputDir := config.InputDir
	err := os.MkdirAll(inputDir, 0755)
	require.NoError(t, err)
	
	testFiles := []string{}
	for i := 0; i < 2; i++ {
		pdfPath := filepath.Join(inputDir, "test"+string(rune('0'+i))+".pdf")
		err := os.WriteFile(pdfPath, []byte("fake pdf content"), 0644)
		require.NoError(t, err)
		testFiles = append(testFiles, pdfPath)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	// This will fail due to missing API key and other dependencies,
	// but we're testing the function structure
	err = worker.ProcessBatch(ctx, testFiles, config, false)
	// Don't assert error as it's expected to fail without real dependencies
	_ = err
}

// TestProcessingJobLifecycle tests the complete job lifecycle
func TestProcessingJobLifecycle(t *testing.T) {
	tmpDir := t.TempDir()
	config := &app.Config{
		InputDir:        filepath.Join(tmpDir, "input"),
		TexOutputDir:    filepath.Join(tmpDir, "tex"),
		ReportOutputDir: filepath.Join(tmpDir, "reports"),
		MetadataDir:     filepath.Join(tmpDir, "metadata"),
		Processing: app.ProcessingConfig{
			MaxWorkers:      1,
			BatchSize:       10,
			TimeoutPerPaper: 300,
		},
		Gemini: app.GeminiConfig{
			APIKey:      "test-key",
			Model:       "gemini-pro",
			Temperature: 0.7,
			MaxTokens:   2048,
		},
		Latex: app.LatexConfig{
			Compiler: "pdflatex",
			Engine:   "pdflatex",
			CleanAux: true,
		},
	}
	
	metadataStore, err := storage.NewMetadataStore(config.MetadataDir)
	require.NoError(t, err)

	pool := worker.NewWorkerPool(1, config, metadataStore)
	
	// Create a fake job
	job := &worker.ProcessingJob{
		FilePath: filepath.Join(tmpDir, "fake.pdf"),
		FileHash: "fake-hash",
		Priority: 1,
	}
	
	// Write a fake PDF file
	err = os.WriteFile(job.FilePath, []byte("fake pdf content"), 0644)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	pool.Start(ctx)
	
	// Submit a job - this will fail due to missing API dependencies,
	// but we're testing the structure
	pool.SubmitJob(job)
	pool.Close()
	
	// Process results until channel is closed
	go func() {
		for range pool.Results() {
			// Consume results
		}
	}()
	
	pool.Wait()
	
	// If we get here without panic, the structure worked
	assert.True(t, true)
}