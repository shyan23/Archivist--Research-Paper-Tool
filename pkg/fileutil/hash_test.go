package fileutil

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TC-2.1.1: Generate consistent hash for same file
func TestComputeFileHash_Consistency(t *testing.T) {
	// Create temporary test file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.pdf")
	testContent := []byte("This is a test PDF content")

	err := os.WriteFile(testFile, testContent, 0644)
	require.NoError(t, err)

	// Compute hash twice
	hash1, err := ComputeFileHash(testFile)
	require.NoError(t, err)

	hash2, err := ComputeFileHash(testFile)
	require.NoError(t, err)

	// Should be identical
	assert.Equal(t, hash1, hash2, "Hash should be consistent for the same file")
}

// TC-2.1.2: Generate different hashes for different files
func TestComputeFileHash_Uniqueness(t *testing.T) {
	tmpDir := t.TempDir()

	file1 := filepath.Join(tmpDir, "file1.pdf")
	file2 := filepath.Join(tmpDir, "file2.pdf")

	err := os.WriteFile(file1, []byte("Content A"), 0644)
	require.NoError(t, err)

	err = os.WriteFile(file2, []byte("Content B"), 0644)
	require.NoError(t, err)

	hash1, err := ComputeFileHash(file1)
	require.NoError(t, err)

	hash2, err := ComputeFileHash(file2)
	require.NoError(t, err)

	assert.NotEqual(t, hash1, hash2, "Different files should have different hashes")
}

// TC-1.2.1: Discover all PDF files in directory
func TestGetPDFFiles_Discovery(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test files
	testFiles := []string{"paper1.pdf", "paper2.pdf", "paper3.pdf"}
	for _, name := range testFiles {
		path := filepath.Join(tmpDir, name)
		err := os.WriteFile(path, []byte("PDF content"), 0644)
		require.NoError(t, err)
	}

	files, err := GetPDFFiles(tmpDir)
	require.NoError(t, err)

	assert.Len(t, files, 3, "Should discover all 3 PDF files")
}

// TC-1.2.2: Ignore non-PDF files
func TestGetPDFFiles_IgnoreNonPDF(t *testing.T) {
	tmpDir := t.TempDir()

	// Create mixed file types
	files := map[string]string{
		"paper.pdf":     "PDF",
		"notes.txt":     "Text",
		"document.docx": "Word",
		"image.png":     "Image",
	}

	for name, content := range files {
		path := filepath.Join(tmpDir, name)
		err := os.WriteFile(path, []byte(content), 0644)
		require.NoError(t, err)
	}

	pdfFiles, err := GetPDFFiles(tmpDir)
	require.NoError(t, err)

	assert.Len(t, pdfFiles, 1, "Should only find 1 PDF file")
	assert.Contains(t, pdfFiles[0], "paper.pdf", "Should find the PDF file")
}

// TC-1.2.4: Handle empty directory
func TestGetPDFFiles_EmptyDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	files, err := GetPDFFiles(tmpDir)
	require.NoError(t, err)

	assert.Len(t, files, 0, "Should return empty list for empty directory")
}

// TC-1.1.3: Handle special characters in sanitization
func TestSanitizeFilename_SpecialCharacters(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Basic sanitization",
			input:    "Learning Deep Features: A Study",
			expected: "Learning_Deep_Features__A_Study",
		},
		{
			name:     "Special symbols",
			input:    "Paper: Title/Subtitle (2024) - Part 1",
			expected: "Paper__Title_Subtitle_(2024)_-_Part_1",
		},
		{
			name:     "Multiple spaces",
			input:    "Too    Many     Spaces",
			expected: "Too____Many_____Spaces",
		},
		{
			name:     "Leading/trailing spaces",
			input:    "  Trimmed Title  ",
			expected: "__Trimmed_Title__",
		},
		{
			name:     "Invalid filename chars",
			input:    "file/name\\with:invalid*chars",
			expected: "file_name_with_invalid_chars",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeFilename(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TC-1.1.4: Handle very long filenames
func TestSanitizeFilename_LongTitle(t *testing.T) {
	// Create a very long title (>255 characters)
	longTitle := "This is an extremely long paper title that goes on and on and on without stopping because some researchers like to put their entire abstract in the title which is really not a good practice but we need to handle it anyway so here we go with more and more text until we exceed two hundred and fifty five characters"

	result := SanitizeFilename(longTitle)

	// Should be truncated to reasonable length
	assert.LessOrEqual(t, len(result), 200, "Filename should be truncated")
	assert.NotEmpty(t, result, "Filename should not be empty")
}

// TC-9.1.1: Handle non-existent file path
func TestComputeFileHash_NonExistent(t *testing.T) {
	_, err := ComputeFileHash("/nonexistent/file.pdf")
	assert.Error(t, err, "Should return error for non-existent file")
}

// Test GetPDFFiles with non-existent directory
func TestGetPDFFiles_NonExistentDirectory(t *testing.T) {
	_, err := GetPDFFiles("/nonexistent/directory")
	assert.Error(t, err, "Should return error for non-existent directory")
}

// Benchmark hash computation
func BenchmarkComputeFileHash(b *testing.B) {
	tmpDir := b.TempDir()
	testFile := filepath.Join(tmpDir, "bench.pdf")

	// Create a larger test file
	content := make([]byte, 1024*1024) // 1MB
	err := os.WriteFile(testFile, content, 0644)
	require.NoError(b, err)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = ComputeFileHash(testFile)
	}
}
