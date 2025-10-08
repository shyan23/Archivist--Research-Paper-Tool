package generator

import (
	"fmt"
	"os"
	"path/filepath"
)

type LatexGenerator struct {
	outputDir string
}

// NewLatexGenerator creates a new LaTeX generator
func NewLatexGenerator(outputDir string) *LatexGenerator {
	return &LatexGenerator{
		outputDir: outputDir,
	}
}

// GenerateLatexFile writes LaTeX content to a file
func (lg *LatexGenerator) GenerateLatexFile(paperTitle, latexContent string) (string, error) {
	// Sanitize filename
	filename := sanitizeFilename(paperTitle)
	if filename == "" {
		filename = "paper_analysis"
	}

	// Create output path
	outputPath := filepath.Join(lg.outputDir, filename+".tex")

	// Ensure output directory exists
	if err := os.MkdirAll(lg.outputDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create output directory: %w", err)
	}

	// Write LaTeX content
	if err := os.WriteFile(outputPath, []byte(latexContent), 0644); err != nil {
		return "", fmt.Errorf("failed to write LaTeX file: %w", err)
	}

	return outputPath, nil
}

// sanitizeFilename removes invalid characters
func sanitizeFilename(name string) string {
	// Simple sanitization
	var result []rune
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
			result = append(result, r)
		} else if r == ' ' {
			result = append(result, '_')
		}
	}

	s := string(result)
	if len(s) > 200 {
		s = s[:200]
	}

	return s
}
