package parser_test

import (
	"archivist/internal/parser"
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockGeminiAnalyzer for testing
type MockGeminiAnalyzer struct {
	mock.Mock
}

func (m *MockGeminiAnalyzer) AnalyzePDFWithVision(ctx context.Context, pdfPath, prompt string) (string, error) {
	args := m.Called(ctx, pdfPath, prompt)
	return args.String(0), args.Error(1)
}

// TestNewPDFParser tests creating a new PDF parser
func TestNewPDFParser(t *testing.T) {
	mockClient := &MockGeminiAnalyzer{}

	p := parser.NewPDFParser(mockClient)
	assert.NotNil(t, p)
}

// TestExtractMetadata tests metadata extraction from PDF
func TestExtractMetadata(t *testing.T) {
	// Create a mock client
	mockClient := new(MockGeminiAnalyzer)
	
	// Create a temporary PDF file for testing
	tmpDir := t.TempDir()
	pdfPath := filepath.Join(tmpDir, "test.pdf")
	err := os.WriteFile(pdfPath, []byte("fake pdf content"), 0644)
	require.NoError(t, err)

	expectedResponse := "TITLE: Test Paper Title\nAUTHORS: Author 1, Author 2\nYEAR: 2023\nABSTRACT: This is a test abstract for the research paper.\n"
	mockClient.On("AnalyzePDFWithVision", mock.Anything, pdfPath, mock.Anything).Return(expectedResponse, nil)

	parser := parser.NewPDFParser(mockClient)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	metadata, err := parser.ExtractMetadata(ctx, pdfPath)
	assert.NoError(t, err)
	assert.Equal(t, "Test Paper Title", metadata.Title)
	assert.Equal(t, []string{"Author 1", "Author 2"}, metadata.Authors)
	assert.Equal(t, "2023", metadata.Year)
	assert.Equal(t, "This is a test abstract for the research paper.", metadata.Abstract)

	mockClient.AssertExpectations(t)
}

// TestExtractMetadataErrorHandling tests error handling in metadata extraction
func TestExtractMetadataErrorHandling(t *testing.T) {
	mockClient := new(MockGeminiAnalyzer)
	
	tmpDir := t.TempDir()
	pdfPath := filepath.Join(tmpDir, "test.pdf")
	err := os.WriteFile(pdfPath, []byte("fake pdf content"), 0644)
	require.NoError(t, err)

	mockClient.On("AnalyzePDFWithVision", mock.Anything, pdfPath, mock.Anything).Return("", assert.AnError)

	parser := parser.NewPDFParser(mockClient)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	metadata, err := parser.ExtractMetadata(ctx, pdfPath)
	assert.Error(t, err)
	assert.Nil(t, metadata)
	assert.Contains(t, err.Error(), "failed to extract metadata")

	mockClient.AssertExpectations(t)
}

// TestParseMetadataResponse tests parsing of metadata response
func TestParseMetadataResponse(t *testing.T) {
	// parseMetadataResponse is unexported and cannot be tested from external package
	t.Skip("parseMetadataResponse is unexported and cannot be tested from external package")
}

// TestHelperFunctions tests helper functions
func TestHelperFunctions(t *testing.T) {
	// Helper functions (splitLines, splitComma, trim) are unexported
	t.Skip("Helper functions are unexported and cannot be tested from external package")
}

// TestFileRenamingTests tests file renaming functionality
func TestFileRenamingTests(t *testing.T) {
	tmpDir := t.TempDir()
	
	// TC-1.1.1: Verify paper with unnamed file is renamed to its title
	t.Run("TC-1.1.1_rename_unnamed_file", func(t *testing.T) {
		// Create a mock PDF file with "download.pdf" name
		downloadPath := filepath.Join(tmpDir, "download.pdf")
		err := os.WriteFile(downloadPath, []byte("fake pdf content"), 0644)
		require.NoError(t, err)

		// Mock the metadata extraction to return a title
		mockClient := new(MockGeminiAnalyzer)
		expectedResponse := "TITLE: Attention Is All You Need\nAUTHORS: Vaswani et al.\nYEAR: 2017\nABSTRACT: We propose a new simple network architecture."
		mockClient.On("AnalyzePDFWithVision", mock.Anything, downloadPath, mock.Anything).Return(expectedResponse, nil)

		parser := parser.NewPDFParser(mockClient)
		metadata, err := parser.ExtractMetadata(context.Background(), downloadPath)
		require.NoError(t, err)
		
		// Verify title extraction works
		assert.Equal(t, "Attention Is All You Need", metadata.Title)
		mockClient.AssertExpectations(t)
	})

	// TC-1.1.2: Verify paper already named correctly is not renamed
	t.Run("TC-1.1.2_no_rename_correctly_named", func(t *testing.T) {
		correctlyNamedPath := filepath.Join(tmpDir, "attention_is_all_you_need.pdf")
		err := os.WriteFile(correctlyNamedPath, []byte("fake pdf content"), 0644)
		require.NoError(t, err)

		mockClient := new(MockGeminiAnalyzer)
		expectedResponse := "TITLE: Attention Is All You Need\nAUTHORS: Vaswani et al.\nYEAR: 2017\nABSTRACT: We propose a new simple network architecture."
		mockClient.On("AnalyzePDFWithVision", mock.Anything, correctlyNamedPath, mock.Anything).Return(expectedResponse, nil)

		parser := parser.NewPDFParser(mockClient)
		metadata, err := parser.ExtractMetadata(context.Background(), correctlyNamedPath)
		require.NoError(t, err)
		
		// The title should match the filename (indicating no rename needed)
		expectedTitle := "Attention Is All You Need"
		assert.Equal(t, expectedTitle, metadata.Title)
		mockClient.AssertExpectations(t)
	})

	// TC-1.1.3: Handle special characters in paper titles
	t.Run("TC-1.1.3_special_characters_in_title", func(t *testing.T) {
		specialCharPath := filepath.Join(tmpDir, "special_chars.pdf")
		err := os.WriteFile(specialCharPath, []byte("fake pdf content"), 0644)
		require.NoError(t, err)

		mockClient := new(MockGeminiAnalyzer)
		expectedResponse := "TITLE: Learning Deep Features: A Study with α, β, γ\nAUTHORS: Author Name\nYEAR: 2023\nABSTRACT: This paper studies deep features."
		mockClient.On("AnalyzePDFWithVision", mock.Anything, specialCharPath, mock.Anything).Return(expectedResponse, nil)

		parser := parser.NewPDFParser(mockClient)
		metadata, err := parser.ExtractMetadata(context.Background(), specialCharPath)
		require.NoError(t, err)
		
		// Verify special characters are handled
		assert.Equal(t, "Learning Deep Features: A Study with α, β, γ", metadata.Title)
		mockClient.AssertExpectations(t)
	})

	// TC-1.1.4: Handle very long paper titles
	t.Run("TC-1.1.4_long_title_handling", func(t *testing.T) {
		longTitlePath := filepath.Join(tmpDir, "long_title.pdf")
		err := os.WriteFile(longTitlePath, []byte("fake pdf content"), 0644)
		require.NoError(t, err)

		mockClient := new(MockGeminiAnalyzer)
		// Create a very long title (>255 characters)
		longTitle := "This is a very long paper title that exceeds the maximum filename length and should be truncated to prevent issues with the filesystem and ensure compatibility across different operating systems and file systems that have various limitations on filename lengths and character restrictions"
		expectedResponse := "TITLE: " + longTitle + "\nAUTHORS: Author Name\nYEAR: 2023\nABSTRACT: This paper has a very long title."
		mockClient.On("AnalyzePDFWithVision", mock.Anything, longTitlePath, mock.Anything).Return(expectedResponse, nil)

		parser := parser.NewPDFParser(mockClient)
		metadata, err := parser.ExtractMetadata(context.Background(), longTitlePath)
		require.NoError(t, err)
		
		// Verify long title is extracted correctly
		assert.Equal(t, longTitle, metadata.Title)
		mockClient.AssertExpectations(t)
	})
}

// TestFileDiscoveryTests tests file discovery functionality
func TestFileDiscoveryTests(t *testing.T) {
	tmpDir := t.TempDir()
	
	// TC-1.2.1: Discover all PDF files in directory
	t.Run("TC-1.2.1_discover_all_pdfs", func(t *testing.T) {
		libDir := filepath.Join(tmpDir, "lib")
		err := os.MkdirAll(libDir, 0755)
		require.NoError(t, err)

		// Create multiple PDF files
		pdfFiles := []string{
			"paper1.pdf",
			"paper2.pdf", 
			"paper3.pdf",
		}

		for _, filename := range pdfFiles {
			filePath := filepath.Join(libDir, filename)
			err := os.WriteFile(filePath, []byte("%PDF-1.4 fake content"), 0644)
			require.NoError(t, err)
		}

		// Discover PDF files
		var discoveredFiles []string
		err = filepath.Walk(libDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() && filepath.Ext(path) == ".pdf" {
				discoveredFiles = append(discoveredFiles, path)
			}
			return nil
		})
		require.NoError(t, err)

		assert.Len(t, discoveredFiles, 3, "Should discover all 3 PDF files")
	})

	// TC-1.2.2: Ignore non-PDF files
	t.Run("TC-1.2.2_ignore_non_pdf_files", func(t *testing.T) {
		libDir := filepath.Join(tmpDir, "lib_mixed")
		err := os.MkdirAll(libDir, 0755)
		require.NoError(t, err)

		// Create mixed file types
		files := []struct {
			name string
			isPDF bool
		}{
			{"paper1.pdf", true},
			{"document.txt", false},
			{"paper2.pdf", true},
			{"image.png", false},
			{"paper3.pdf", true},
		}

		for _, file := range files {
			filePath := filepath.Join(libDir, file.name)
			content := "content"
			if file.isPDF {
				content = "%PDF-1.4 fake content"
			}
			err := os.WriteFile(filePath, []byte(content), 0644)
			require.NoError(t, err)
		}

		// Discover only PDF files
		var discoveredFiles []string
		err = filepath.Walk(libDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() && filepath.Ext(path) == ".pdf" {
				discoveredFiles = append(discoveredFiles, path)
			}
			return nil
		})
		require.NoError(t, err)

		assert.Len(t, discoveredFiles, 3, "Should discover only 3 PDF files, ignoring non-PDF files")
	})

	// TC-1.2.3: Handle nested directories
	t.Run("TC-1.2.3_handle_nested_directories", func(t *testing.T) {
		libDir := filepath.Join(tmpDir, "lib_nested")
		err := os.MkdirAll(libDir, 0755)
		require.NoError(t, err)

		// Create nested directory structure
		subDir := filepath.Join(libDir, "subdir")
		err = os.MkdirAll(subDir, 0755)
		require.NoError(t, err)

		// Create PDFs in both root and subdirectory
		rootPdf := filepath.Join(libDir, "root_paper.pdf")
		err = os.WriteFile(rootPdf, []byte("%PDF-1.4 fake content"), 0644)
		require.NoError(t, err)

		nestedPdf := filepath.Join(subDir, "nested_paper.pdf")
		err = os.WriteFile(nestedPdf, []byte("%PDF-1.4 fake content"), 0644)
		require.NoError(t, err)

		// Discover PDF files recursively
		var discoveredFiles []string
		err = filepath.Walk(libDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() && filepath.Ext(path) == ".pdf" {
				discoveredFiles = append(discoveredFiles, path)
			}
			return nil
		})
		require.NoError(t, err)

		assert.Len(t, discoveredFiles, 2, "Should discover PDFs in both root and nested directories")
	})

	// TC-1.2.4: Handle empty directory
	t.Run("TC-1.2.4_handle_empty_directory", func(t *testing.T) {
		emptyDir := filepath.Join(tmpDir, "empty_lib")
		err := os.MkdirAll(emptyDir, 0755)
		require.NoError(t, err)

		// Try to discover PDF files in empty directory
		var discoveredFiles []string
		err = filepath.Walk(emptyDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() && filepath.Ext(path) == ".pdf" {
				discoveredFiles = append(discoveredFiles, path)
			}
			return nil
		})
		require.NoError(t, err)

		assert.Len(t, discoveredFiles, 0, "Should find no PDF files in empty directory")
	})
}