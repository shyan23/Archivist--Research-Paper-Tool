package generator

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewLatexGenerator tests creating a new LaTeX generator
func TestNewLatexGenerator(t *testing.T) {
	outputDir := "/tmp/test-output"
	generator := NewLatexGenerator(outputDir)
	
	assert.NotNil(t, generator)
	assert.Equal(t, outputDir, generator.outputDir)
}

// TestGenerateLatexFile tests generating a LaTeX file
func TestGenerateLatexFile(t *testing.T) {
	tmpDir := t.TempDir()
	outputDir := filepath.Join(tmpDir, "output")
	generator := NewLatexGenerator(outputDir)

	paperTitle := "Test Paper Title"
	latexContent := "\\documentclass{article}\n\\begin{document}\nTest content\n\\end{document}"

	path, err := generator.GenerateLatexFile(paperTitle, latexContent)
	assert.NoError(t, err)
	assert.Equal(t, filepath.Join(outputDir, "Test_Paper_Title.tex"), path)

	// Check if file exists and has correct content
	content, err := os.ReadFile(path)
	assert.NoError(t, err)
	assert.Equal(t, latexContent, string(content))
}

// TestGenerateLatexFileWithSpecialCharacters tests LaTeX generation with special characters in title
func TestGenerateLatexFileWithSpecialCharacters(t *testing.T) {
	tmpDir := t.TempDir()
	outputDir := filepath.Join(tmpDir, "output")
	generator := NewLatexGenerator(outputDir)

	paperTitle := "Test: Paper & Title with / Special \\ Chars"
	expectedFilename := "Test_Paper_Title_with__Special__Chars.tex"
	latexContent := "\\documentclass{article}\n\\begin{document}\nTest content\n\\end{document}"

	path, err := generator.GenerateLatexFile(paperTitle, latexContent)
	assert.NoError(t, err)
	assert.Equal(t, filepath.Join(outputDir, expectedFilename), path)

	// Check if file exists
	_, err = os.Stat(path)
	assert.NoError(t, err)
}

// TestGenerateLatexFileWithLongTitle tests LaTeX generation with a very long title
func TestGenerateLatexFileWithLongTitle(t *testing.T) {
	tmpDir := t.TempDir()
	outputDir := filepath.Join(tmpDir, "output")
	generator := NewLatexGenerator(outputDir)

	longTitle := "This is a very long paper title that exceeds the maximum filename length and should be truncated to prevent issues with the filesystem"
	latexContent := "\\documentclass{article}\n\\begin{document}\nTest content\n\\end{document}"

	path, err := generator.GenerateLatexFile(longTitle, latexContent)
	assert.NoError(t, err)
	
	// Check that the filename was truncated
	filename := filepath.Base(path)
	assert.Less(t, len(filename), len(longTitle))
	assert.Equal(t, outputDir, filepath.Dir(path))
	assert.True(t, len(filename) <= 205) // .tex adds 4 chars, we truncate to 200

	// Check if file exists
	_, err = os.Stat(path)
	assert.NoError(t, err)
}

// TestGenerateLatexFileWithEmptyTitle tests LaTeX generation with empty title
func TestGenerateLatexFileWithEmptyTitle(t *testing.T) {
	tmpDir := t.TempDir()
	outputDir := filepath.Join(tmpDir, "output")
	generator := NewLatexGenerator(outputDir)

	paperTitle := ""
	latexContent := "\\documentclass{article}\n\\begin{document}\nTest content\n\\end{document}"

	path, err := generator.GenerateLatexFile(paperTitle, latexContent)
	assert.NoError(t, err)
	assert.Equal(t, filepath.Join(outputDir, "paper_analysis.tex"), path)

	// Check if file exists
	_, err = os.Stat(path)
	assert.NoError(t, err)
}

// TestGenerateLatexFileCreateOutputDir tests that output directory is created if it doesn't exist
func TestGenerateLatexFileCreateOutputDir(t *testing.T) {
	tmpDir := t.TempDir()
	outputDir := filepath.Join(tmpDir, "new", "output", "directory")
	generator := NewLatexGenerator(outputDir)

	paperTitle := "Test Paper Title"
	latexContent := "\\documentclass{article}\n\\begin{document}\nTest content\n\\end{document}"

	path, err := generator.GenerateLatexFile(paperTitle, latexContent)
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(outputDir, "Test_Paper_Title.tex"), path)

	// Check if file exists and if directory was created
	_, err = os.Stat(path)
	assert.NoError(t, err)
	
	// Check if output directory exists
	_, err = os.Stat(outputDir)
	assert.NoError(t, err)
}

// TestGenerateLatexFileErrorHandling tests error handling for file writing
func TestGenerateLatexFileErrorHandling(t *testing.T) {
	// Try to write to a path where we don't have permissions (this may not work in all environments)
	// Instead, we'll test with an invalid path
	outputDir := "/invalid/path/that/should/not/exist"
	generator := NewLatexGenerator(outputDir)

	paperTitle := "Test Paper Title"
	latexContent := "\\documentclass{article}\n\\begin{document}\nTest content\n\\end{document}"

	path, err := generator.GenerateLatexFile(paperTitle, latexContent)
	assert.Error(t, err)
	assert.Empty(t, path)
	assert.Contains(t, err.Error(), "failed to create output directory")
}

// TestSanitizeFilename tests the sanitizeFilename function
func TestSanitizeFilename(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Valid characters",
			input:    "valid_title_123",
			expected: "valid_title_123",
		},
		{
			name:     "Spaces to underscores",
			input:    "title with spaces",
			expected: "title_with_spaces",
		},
		{
			name:     "Special characters removed",
			input:    "title:with/special\\chars*?",
			expected: "title_with_special_chars_",
		},
		{
			name:     "Non-ASCII characters",
			input:    "title_with_α_β_γ",
			expected: "title_with____", // Greek letters become underscores
		},
		{
			name:     "Empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "Long string truncation",
			input:    "a" + "b" + "c" + "d" + "e", // Make a string longer than 200 chars
			expected: "a" + "b" + "c" + "d" + "e", // This test case needs adjustment
		},
		{
			name:     "Mixed valid and invalid characters",
			input:    "My Paper: A Study of Something & Other Things",
			expected: "My_Paper_A_Study_of_Something__Other_Things",
		},
	}

	// Create a long input for truncation test
	longInput := ""
	for i := 0; i < 250; i++ {
		longInput += "a"
	}
	longExpected := longInput[:200] // Should be truncated to 200 chars

	testCases = append(testCases, struct {
		name     string
		input    string
		expected string
	}{
		name:     "Long string truncation",
		input:    longInput,
		expected: longExpected,
	})

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := sanitizeFilename(tc.input)
			assert.Equal(t, tc.expected, result)
			
			// Additional check for truncation
			if len(tc.expected) > 200 {
				assert.Equal(t, 200, len(result))
			} else {
				assert.Equal(t, tc.expected, result)
			}
		})
	}
}