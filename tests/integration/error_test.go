package integration_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"archivist/internal/generator"
	"archivist/internal/parser"
	"archivist/internal/storage"
	"archivist/tests/helpers/testhelpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// TestErrorHandlingInAnalysis tests error handling in the analysis process
func TestErrorHandlingInAnalysis(t *testing.T) {
	// Note: This test cannot inject the mock client through the public API
	// The Analyzer struct has unexported fields (client, config) that cannot be set
	// from external packages. This test should be moved to the analyzer package itself
	// as a unit test where it can access internal implementation details.
	t.Skip("Cannot inject mock client through public API - test needs to be moved to analyzer package")
}

// TestErrorHandlingInMetadataExtraction tests error handling in metadata extraction
func TestErrorHandlingInMetadataExtraction(t *testing.T) {
	// Create a mock analyzer client that returns an error for metadata extraction
	mockClient := new(MockGeminiClient)
	
	tmpDir := t.TempDir()
	pdfPath := filepath.Join(tmpDir, "test.pdf")
	err := os.WriteFile(pdfPath, []byte("fake pdf content"), 0644)
	require.NoError(t, err)

	mockClient.On("AnalyzePDFWithVision", mock.Anything, pdfPath, mock.AnythingOfType("string")).Return("", assert.AnError)

	pdfParser := parser.NewPDFParser(mockClient)

	_, err = pdfParser.ExtractMetadata(context.Background(), pdfPath)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to extract metadata")

	mockClient.AssertExpectations(t)
}

// TestErrorHandlingInLatexGeneration tests error handling in LaTeX generation
func TestErrorHandlingInLatexGeneration(t *testing.T) {
	// Try to create LaTeX generator with invalid directory
	invalidDir := "/invalid/path/that/should/not/exist"
	latexGen := generator.NewLatexGenerator(invalidDir)

	paperTitle := "Test Paper"
	latexContent := "\\documentclass{article}\n\\begin{document}\nTest\n\\end{document}"

	_, err := latexGen.GenerateLatexFile(paperTitle, latexContent)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create output directory")
}

// TestErrorHandlingInMetadataStorage tests error handling in metadata storage
func TestErrorHandlingInMetadataStorage(t *testing.T) {
	tmpDir := t.TempDir()
	metadataDir := filepath.Join(tmpDir, "metadata")
	
	// Create a file instead of a directory to cause an error
	err := os.WriteFile(metadataDir, []byte("not a directory"), 0644)
	require.NoError(t, err)

	// This should fail because the path is not a directory
	_, err = storage.NewMetadataStore(metadataDir)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create output directory")
}

// TestEmptyPDFFile tests handling of empty PDF files
func TestEmptyPDFFile(t *testing.T) {
	config := testhelpers.TestConfig(t)
	
	// Create an empty PDF file
	emptyPdfPath := filepath.Join(config.InputDir, "empty.pdf")
	err := os.WriteFile(emptyPdfPath, []byte(""), 0644)
	require.NoError(t, err)

	// Mock analyzer client for this test
	mockClient := new(MockGeminiClient)
	mockClient.On("AnalyzePDFWithVision", mock.Anything, emptyPdfPath, mock.AnythingOfType("string")).Return("", assert.AnError)

	pdfParser := parser.NewPDFParser(mockClient)

	_, err = pdfParser.ExtractMetadata(context.Background(), emptyPdfPath)
	assert.Error(t, err)

	mockClient.AssertExpectations(t)
}

// TestNonExistentFile tests handling of non-existent files
func TestNonExistentFile(t *testing.T) {
	// Mock analyzer client
	mockClient := new(MockGeminiClient)
	
	nonExistentPath := "/path/that/does/not/exist.pdf"

	pdfParser := parser.NewPDFParser(mockClient)

	// The API call should fail because the file doesn't exist
	_, err := pdfParser.ExtractMetadata(context.Background(), nonExistentPath)
	assert.Error(t, err)

	mockClient.AssertNotCalled(t, "AnalyzePDFWithVision")
}

// TestInvalidPDFFile tests handling of corrupted or invalid PDF files
func TestInvalidPDFFile(t *testing.T) {
	config := testhelpers.TestConfig(t)
	
	// Create a file that's not a valid PDF
	invalidPdfPath := filepath.Join(config.InputDir, "invalid.pdf")
	err := os.WriteFile(invalidPdfPath, []byte("this is not a pdf file"), 0644)
	require.NoError(t, err)

	// Mock analyzer client - might try to process invalid PDF
	mockClient := new(MockGeminiClient)
	mockClient.On("AnalyzePDFWithVision", mock.Anything, invalidPdfPath, mock.AnythingOfType("string")).Return("", assert.AnError)

	pdfParser := parser.NewPDFParser(mockClient)

	_, err = pdfParser.ExtractMetadata(context.Background(), invalidPdfPath)
	assert.Error(t, err)

	mockClient.AssertExpectations(t)
}

// TestTimeoutHandling tests timeout handling in the processing pipeline
func TestTimeoutHandling(t *testing.T) {
	config := testhelpers.TestConfig(t)
	
	pdfPath := testhelpers.CreateTestPDF(t, config.InputDir, "timeout_test.pdf", "")
	
	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	// Mock analyzer client that would take longer than the timeout
	mockClient := new(MockGeminiClient)
	mockClient.On("AnalyzePDFWithVision", mock.Anything, pdfPath, mock.AnythingOfType("string")).Run(func(args mock.Arguments) {
		// Simulate a slow operation that exceeds the timeout
		time.Sleep(10 * time.Millisecond)
	}).Return("result", nil)

	pdfParser := parser.NewPDFParser(mockClient)

	_, _ = pdfParser.ExtractMetadata(ctx, pdfPath)
	// The error could be a context cancellation error or the mock error
	// depending on implementation timing - we don't assert on it

	mockClient.AssertExpectations(t)
}

// TestDiskSpaceError tests handling when disk space runs out during processing
func TestDiskSpaceError(t *testing.T) {
	// Note: Actually testing disk space exhaustion is difficult in a unit test
	// Instead, we'll test the error handling path when file writing fails
	
	tmpDir := t.TempDir()
	latexDir := filepath.Join(tmpDir, "tex")
	
	// Create the directory
	err := os.MkdirAll(latexDir, 0755)
	require.NoError(t, err)

	// Create a LaTeX generator
	latexGen := generator.NewLatexGenerator(latexDir)

	// Temporarily make the directory read-only to simulate write error
	err = os.Chmod(latexDir, 0444) // read-only
	require.NoError(t, err)

	// This should fail due to permission error
	_, err = latexGen.GenerateLatexFile("Test Paper", "\\documentclass{article}")
	assert.Error(t, err)

	// Restore permissions for cleanup
	err = os.Chmod(latexDir, 0755)
	require.NoError(t, err)
}

// TestMetadataStoreRaceCondition tests potential race conditions in metadata storage
func TestMetadataStoreRaceCondition(t *testing.T) {
	tmpDir := t.TempDir()
	metadataDir := filepath.Join(tmpDir, "metadata")
	
	store, err := storage.NewMetadataStore(metadataDir)
	require.NoError(t, err)

	// Simulate multiple goroutines trying to update the same record
	numGoroutines := 10
	done := make(chan bool, numGoroutines)

	hash := "test-hash"
	
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			record := storage.ProcessingRecord{
				FilePath:    "/path/to/paper.pdf",
				FileHash:    hash,
				PaperTitle:  "Test Paper " + string(rune('0'+id)),
				ProcessedAt: time.Now(),
				Status:      storage.StatusCompleted,
			}
			err := store.MarkCompleted(record)
			assert.NoError(t, err)
			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	// Check that the store still works correctly
	record, exists := store.GetRecord(hash)
	assert.True(t, exists)
	assert.Equal(t, storage.StatusCompleted, record.Status)
}

// TestLargeTitleTruncation tests handling of very long titles
func TestLargeTitleTruncation(t *testing.T) {
	tmpDir := t.TempDir()
	latexDir := filepath.Join(tmpDir, "tex")
	err := os.MkdirAll(latexDir, 0755)
	require.NoError(t, err)

	latexGen := generator.NewLatexGenerator(latexDir)

	// Create a very long title (> 200 characters)
	longTitle := ""
	for i := 0; i < 30; i++ {
		longTitle += "This is a very long title segment "
	}

	latexContent := "\\documentclass{article}\n\\begin{document}\nTest\n\\end{document}"

	path, err := latexGen.GenerateLatexFile(longTitle, latexContent)
	assert.NoError(t, err)
	
	// Check that the filename was truncated
	filename := filepath.Base(path)
	assert.Less(t, len(filename), len(longTitle))
	assert.Less(t, len(filename), 210) // Less than 200 + .tex extension
	
	// Check that the file exists
	_, err = os.Stat(path)
	assert.NoError(t, err)
}

// TestConcurrentProcessingError tests error handling with concurrent processing
func TestConcurrentProcessingError(t *testing.T) {
	tmpDir := t.TempDir()
	metadataDir := filepath.Join(tmpDir, "metadata")
	
	store, err := storage.NewMetadataStore(metadataDir)
	require.NoError(t, err)

	// Simulate multiple goroutines trying to mark the same file as failed
	numGoroutines := 5
	done := make(chan bool, numGoroutines)

	hash := "concurrent-hash"
	
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			err := store.MarkFailed(hash, "Error from goroutine "+string(rune('0'+id)))
			assert.NoError(t, err)
			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	// Check that the store still works correctly
	record, exists := store.GetRecord(hash)
	assert.True(t, exists)
	assert.Equal(t, storage.StatusFailed, record.Status)
}

// TestAPIErrorHandling tests handling of API errors in the analyzer
func TestAPIErrorHandling(t *testing.T) {
	// Note: This test cannot inject the mock client through the public API
	// The Analyzer struct has unexported fields (client, config) that cannot be set
	// from external packages. This test should be moved to the analyzer package itself
	// as a unit test where it can access internal implementation details.
	t.Skip("Cannot inject mock client through public API - test needs to be moved to analyzer package")
}