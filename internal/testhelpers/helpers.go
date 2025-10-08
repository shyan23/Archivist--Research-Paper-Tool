package testhelpers

import (
	"os"
	"path/filepath"
	"testing"

	"archivist/internal/app"
	"archivist/pkg/fileutil"
)

// TestConfig creates a test configuration with temporary directories
func TestConfig(t *testing.T) *app.Config {
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
			APIKey:      "test-key", // This will be overridden in tests that need it
			Model:       "gemini-pro",
			Temperature: 0.7,
			MaxTokens:   2048,
			Agentic: app.AgenticConfig{
				Enabled: false,
			},
		},
		Latex: app.LatexConfig{
			Compiler: "pdflatex",
			Engine:   "pdflatex",
			CleanAux: true,
		},
		HashAlgorithm: "md5",
		Logging: app.LoggingConfig{
			Level:   "info",
			File:    "",
			Console: true,
		},
	}
	
	// Create all required directories
	dirs := []string{
		config.InputDir,
		config.TexOutputDir,
		config.ReportOutputDir,
		config.MetadataDir,
	}
	
	for _, dir := range dirs {
		err := os.MkdirAll(dir, 0755)
		if err != nil {
			t.Fatalf("Failed to create directory %s: %v", dir, err)
		}
	}
	
	return config
}

// CreateTestPDF creates a test PDF file in the specified directory
func CreateTestPDF(t *testing.T, dir, filename, content string) string {
	pdfPath := filepath.Join(dir, filename)
	
	// Create a minimal PDF content
	if content == "" {
		content = "%PDF-1.4\n1 0 obj\n<<\n/Type /Catalog\n/Pages 2 0 R\n>>\nendobj\n"
	}
	
	err := os.WriteFile(pdfPath, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create test PDF %s: %v", pdfPath, err)
	}
	
	return pdfPath
}

// CreateTestLaTeX creates a test LaTeX file in the specified directory
func CreateTestLaTeX(t *testing.T, dir, filename, content string) string {
	texPath := filepath.Join(dir, filename)
	
	if content == "" {
		content = "\\documentclass{article}\n\\begin{document}\nTest content\n\\end{document}\n"
	}
	
	err := os.WriteFile(texPath, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create test LaTeX %s: %v", texPath, err)
	}
	
	return texPath
}

// ComputeTestFileHash computes the hash of a test file
func ComputeTestFileHash(t *testing.T, filePath string) string {
	hash, err := fileutil.ComputeFileHash(filePath)
	if err != nil {
		t.Fatalf("Failed to compute hash for %s: %v", filePath, err)
	}
	return hash
}

// CleanupTempDir removes the temporary directory and all its contents
func CleanupTempDir(t *testing.T, dir string) {
	err := os.RemoveAll(dir)
	if err != nil {
		t.Logf("Warning: failed to clean up temp directory %s: %v", dir, err)
	}
}