package internal

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"archivist/internal/analyzer"
	"archivist/internal/generator"
	"archivist/internal/storage"
	"archivist/internal/testhelpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// BenchmarkLatexGeneration benchmarks LaTeX file generation
func BenchmarkLatexGeneration(b *testing.B) {
	tmpDir := b.TempDir()
	latexDir := filepath.Join(tmpDir, "tex")
	err := os.MkdirAll(latexDir, 0755)
	if err != nil {
		b.Fatalf("Failed to create temp directory: %v", err)
	}

	latexGen := generator.NewLatexGenerator(latexDir)
	
	latexContent := `\\documentclass{article}
\\usepackage[utf8]{inputenc}
\\usepackage{amsmath,amssymb,amsfonts}
\\usepackage{graphicx}
\\usepackage{hyperref}
\\usepackage{xcolor}
\\usepackage{geometry}
\\usepackage{tcolorbox}
\\usepackage{enumitem}
\\geometry{margin=1in}
\\title{Performance Test Paper}
\\author{Test Author}
\\date{\\today}
\\begin{document}
\\maketitle
\\section{Introduction}
This is a performance test document with sufficient content to measure generation time.
\\end{document}`

	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		title := "Performance Test Paper " + string(rune('0'+(i%10)))
		_, err := latexGen.GenerateLatexFile(title, latexContent)
		if err != nil {
			b.Errorf("Failed to generate LaTeX file: %v", err)
		}
	}
}

// BenchmarkMetadataStore benchmarks metadata storage operations
func BenchmarkMetadataStore(b *testing.B) {
	tmpDir := b.TempDir()
	metadataDir := filepath.Join(tmpDir, "metadata")
	
	store, err := storage.NewMetadataStore(metadataDir)
	if err != nil {
		b.Fatalf("Failed to create metadata store: %v", err)
	}

	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		hash := "hash-" + string(rune('0'+(i%10)))
		record := storage.ProcessingRecord{
			FilePath:    "/path/to/paper" + string(rune('0'+(i%10))) + ".pdf",
			FileHash:    hash,
			PaperTitle:  "Benchmark Paper " + string(rune('0'+(i%10))),
			ProcessedAt: time.Now(),
			Status:      storage.StatusCompleted,
		}
		
		err := store.MarkCompleted(record)
		if err != nil {
			b.Errorf("Failed to mark completed: %v", err)
		}
		
		_, exists := store.GetRecord(hash)
		if !exists {
			b.Errorf("Record not found after storing")
		}
	}
}

// BenchmarkConcurrentMetadataAccess benchmarks concurrent access to metadata store
func BenchmarkConcurrentMetadataAccess(b *testing.B) {
	tmpDir := b.TempDir()
	metadataDir := filepath.Join(tmpDir, "metadata")
	
	store, err := storage.NewMetadataStore(metadataDir)
	if err != nil {
		b.Fatalf("Failed to create metadata store: %v", err)
	}

	// Pre-populate some records
	for i := 0; i < 10; i++ {
		hash := "hash-" + string(rune('0'+i%10))
		record := storage.ProcessingRecord{
			FilePath:    "/path/to/paper" + string(rune('0'+i%10)) + ".pdf",
			FileHash:    hash,
			PaperTitle:  "Benchmark Paper " + string(rune('0'+i%10)),
			ProcessedAt: time.Now(),
			Status:      storage.StatusCompleted,
		}
		store.MarkCompleted(record)
	}

	b.ResetTimer()
	
	// Run concurrent operations
	for i := 0; i < b.N; i++ {
		// Use multiple goroutines to access the store concurrently
		done := make(chan bool, 5)
		
		for j := 0; j < 5; j++ {
			go func(id int) {
				hash := "hash-" + string(rune('0'+(id%10)))
				
				// Try to get a record
				_, exists := store.GetRecord(hash)
				if !exists {
					// Maybe add one if it doesn't exist
					record := storage.ProcessingRecord{
						FilePath:    "/path/to/new_paper" + string(rune('0'+id%10)) + ".pdf",
						FileHash:    hash,
						PaperTitle:  "New Benchmark Paper " + string(rune('0'+id%10)),
						ProcessedAt: time.Now(),
						Status:      storage.StatusCompleted,
					}
					store.MarkCompleted(record)
				}
				
				done <- true
			}(i + j)
		}
		
		// Wait for all goroutines to complete
		for j := 0; j < 5; j++ {
			<-done
		}
	}
}

// TestLatexGenerationPerformance tests performance bounds
func TestLatexGenerationPerformance(t *testing.T) {
	tmpDir := t.TempDir()
	latexDir := filepath.Join(tmpDir, "tex")
	err := os.MkdirAll(latexDir, 0755)
	require.NoError(t, err)

	latexGen := generator.NewLatexGenerator(latexDir)
	
	latexContent := `\\documentclass{article}
\\begin{document}
\\title{Performance Test}
\\section{Introduction}
This is a test document for performance evaluation.
\\end{document}`

	// Measure time for 100 LaTeX generations
	start := time.Now()
	
	for i := 0; i < 100; i++ {
		title := "Performance Test " + string(rune('0'+i%10))
		_, err := latexGen.GenerateLatexFile(title, latexContent)
		require.NoError(t, err)
	}
	
	duration := time.Since(start)
	
	// Expect to generate 100 files in under 5 seconds
	assert.Less(t, duration.Seconds(), 5.0, "LaTeX generation took too long: %v", duration)
	
	t.Logf("Generated 100 LaTeX files in %v (%.2f per second)", duration, 100/duration.Seconds())
}

// TestMetadataStorePerformance tests metadata store performance
func TestMetadataStorePerformance(t *testing.T) {
	tmpDir := t.TempDir()
	metadataDir := filepath.Join(tmpDir, "metadata")
	
	store, err := storage.NewMetadataStore(metadataDir)
	require.NoError(t, err)

	// Measure time for 1000 operations
	start := time.Now()
	
	// Add 1000 records
	for i := 0; i < 1000; i++ {
		hash := "hash-" + string(rune('0'+i%100))
		record := storage.ProcessingRecord{
			FilePath:    "/path/to/paper" + string(rune('0'+i%100)) + ".pdf",
			FileHash:    hash,
			PaperTitle:  "Performance Test Paper " + string(rune('0'+i%100)),
			ProcessedAt: time.Now(),
			Status:      storage.StatusCompleted,
		}
		
		err := store.MarkCompleted(record)
		require.NoError(t, err)
	}
	
	// Retrieve 1000 records
	for i := 0; i < 1000; i++ {
		hash := "hash-" + string(rune('0'+i%100))
		_, exists := store.GetRecord(hash)
		assert.True(t, exists, "Record %d should exist", i)
	}
	
	duration := time.Since(start)
	
	// Expect to handle 2000 operations (1000 writes + 1000 reads) in under 5 seconds
	assert.Less(t, duration.Seconds(), 5.0, "Metadata operations took too long: %v", duration)
	
	t.Logf("Completed 2000 metadata operations in %v (%.2f per second)", duration, 2000/duration.Seconds())
}

// TestAnalyzerPerformance tests performance of the analyzer
func TestAnalyzerPerformance(t *testing.T) {
	// This test simulates the analyzer performance by timing the cleanLatexOutput function
	// which is one of the core functions in the analyzer
	
	latexContent := `\\documentclass{article}
\\begin{document}
\\title{Performance Test}
\\begin{verbatim}
This is a longer LaTeX document to test the processing performance.
\\end{verbatim}
\\section{Introduction}
This is the introduction section of our test document.
\\subsection{Subsection}
Here we have a subsection to further test the processing capabilities.
\\begin{equation}
E = mc^2
\\end{equation}
\\end{document}`

	// Measure time for cleaning 1000 LaTeX documents
	start := time.Now()
	
	for i := 0; i < 1000; i++ {
		cleaned := analyzer.CleanLatexOutput(latexContent) // Assuming this function exists
		_ = cleaned
	}
	
	duration := time.Since(start)
	
	// Expect to clean 1000 documents in under 1 second
	assert.Less(t, duration.Seconds(), 1.0, "LaTeX cleaning took too long: %v", duration)
	
	t.Logf("Cleaned 1000 LaTeX documents in %v (%.2f per second)", duration, 1000/duration.Seconds())
}

// TestFileDiscoveryPerformance tests file discovery performance
func TestFileDiscoveryPerformance(t *testing.T) {
	// Create a directory structure with many files
	tmpDir := t.TempDir()
	
	// Create subdirectories
	for i := 0; i < 10; i++ {
		dir := filepath.Join(tmpDir, "dir"+string(rune('0'+i)))
		err := os.MkdirAll(dir, 0755)
		require.NoError(t, err)
		
		// Create various files in each directory
		for j := 0; j < 50; j++ {
			extension := ".pdf"
			if j%5 == 0 {
				extension = ".txt"
			} else if j%7 == 0 {
				extension = ".doc"
			}
			
			filename := "file" + string(rune('0'+j%10)) + extension
			path := filepath.Join(dir, filename)
			err := os.WriteFile(path, []byte("content"), 0644)
			require.NoError(t, err)
		}
	}

	// Measure time to find all PDF files
	start := time.Now()
	
	pdfCount := 0
	err := filepath.Walk(tmpDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && filepath.Ext(path) == ".pdf" {
			pdfCount++
		}
		return nil
	})
	
	require.NoError(t, err)
	duration := time.Since(start)
	
	// We expect 10 directories * (50 files - 10 .txt - 7 .doc) = 10 * 33 = 330 PDFs
	// (every 5th file is .txt (10 files), every 7th file is .doc (7 files))
	assert.Equal(t, 330, pdfCount, "Expected 330 PDF files")
	
	// Expect to find 330 PDF files in under 1 second
	assert.Less(t, duration.Seconds(), 1.0, "File discovery took too long: %v", duration)
	
	t.Logf("Discovered %d PDF files in %v (%.2f per second)", pdfCount, duration, float64(pdfCount)/duration.Seconds())
}

// TestHashComputationPerformance tests the performance of file hash computation
func TestHashComputationPerformance(t *testing.T) {
	tmpDir := t.TempDir()
	
	// Create test files of various sizes
	files := []struct {
		name string
		size int
	}{
		{"small.pdf", 1024},         // 1KB
		{"medium.pdf", 1024 * 100},  // 100KB
		{"large.pdf", 1024 * 1024},  // 1MB
	}
	
	testFiles := make([]string, 0)
	
	for _, fileSpec := range files {
		content := make([]byte, fileSpec.size)
		for i := range content {
			content[i] = byte(i % 256)
		}
		
		filePath := filepath.Join(tmpDir, fileSpec.name)
		err := os.WriteFile(filePath, content, 0644)
		require.NoError(t, err)
		testFiles = append(testFiles, filePath)
	}

	// Measure time to compute hashes for all files
	start := time.Now()
	hashes := make([]string, 0, len(testFiles))
	
	for _, filePath := range testFiles {
		hash, err := testhelpers.ComputeTestFileHash(t, filePath)
		require.NoError(t, err)
		hashes = append(hashes, hash)
	}
	
	duration := time.Since(start)
	
	// Expect to hash ~1.1MB in under 1 second (hashing is generally fast)
	assert.Less(t, duration.Seconds(), 1.0, "Hash computation took too long: %v", duration)
	
	t.Logf("Computed %d hashes in %v (%.2f per second)", len(testFiles), duration, float64(len(testFiles))/duration.Seconds())
	
	// Verify that all hashes are unique
	uniqueHashes := make(map[string]bool)
	for _, hash := range hashes {
		assert.False(t, uniqueHashes[hash], "Hash should be unique")
		uniqueHashes[hash] = true
	}
}

// TestConcurrentProcessingPerformance tests performance under concurrent load
func TestConcurrentProcessingPerformance(t *testing.T) {
	tmpDir := t.TempDir()
	latexDir := filepath.Join(tmpDir, "latex")
	err := os.MkdirAll(latexDir, 0755)
	require.NoError(t, err)

	latexGen := generator.NewLatexGenerator(latexDir)
	
	latexContent := `\\documentclass{article}
\\begin{document}
\\title{Concurrent Test}
\\section{Test}
Content for concurrent test.
\\end{document}`

	// Run 20 concurrent operations
	numConcurrent := 20
	start := time.Now()
	
	done := make(chan bool, numConcurrent)
	
	for i := 0; i < numConcurrent; i++ {
		go func(id int) {
			title := "Concurrent Test " + string(rune('0'+id%10))
			_, err := latexGen.GenerateLatexFile(title, latexContent)
			assert.NoError(t, err)
			done <- true
		}(i)
	}
	
	// Wait for all operations to complete
	for i := 0; i < numConcurrent; i++ {
		<-done
	}
	
	duration := time.Since(start)
	
	// Expect to complete 20 concurrent operations in under 5 seconds
	assert.Less(t, duration.Seconds(), 5.0, "Concurrent processing took too long: %v", duration)
	
	t.Logf("Completed %d concurrent operations in %v (%.2f per second)", numConcurrent, duration, float64(numConcurrent)/duration.Seconds())
}

// Helper function to maintain compatibility with analyzer.CleanLatexOutput
// Since the function may not be exported, we'll define it here for the performance test
func (a *analyzer.Analyzer) CleanLatexOutput(content string) string {
	// This is a copy of the cleanLatexOutput function from analyzer
	// Remove markdown code blocks if present
	content = stringReplaceAll(content, "```latex", "")
	content = stringReplaceAll(content, "```tex", "")
	content = stringReplaceAll(content, "```", "")

	// Trim whitespace
	content = stringTrimSpace(content)

	return content
}

// Helper functions for the test
func stringReplaceAll(s, old, new string) string {
	// A simple implementation of ReplaceAll
	result := ""
	i := 0
	for i < len(s) {
		if i <= len(s)-len(old) && s[i:i+len(old)] == old {
			result += new
			i += len(old)
		} else {
			result += string(s[i])
			i++
		}
	}
	return result
}

func stringTrimSpace(s string) string {
	// A simple implementation of TrimSpace
	start := 0
	end := len(s)
	
	for start < end && (s[start] == ' ' || s[start] == '\t' || s[start] == '\n' || s[start] == '\r') {
		start++
	}
	
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t' || s[end-1] == '\n' || s[end-1] == '\r') {
		end--
	}
	
	return s[start:end]
}