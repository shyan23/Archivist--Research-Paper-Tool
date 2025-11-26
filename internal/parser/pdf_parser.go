package parser

import (
	"context"
	"fmt"
)

type PaperMetadata struct {
	Title    string
	Authors  []string
	Abstract string
	Year     string
}

type PDFParser struct {
	geminiClient GeminiAnalyzer
}

// GeminiAnalyzer interface for Gemini client
type GeminiAnalyzer interface {
	AnalyzePDFWithVision(ctx context.Context, pdfPath, prompt string) (string, error)
	AnalyzePDFWithVisionRetry(ctx context.Context, pdfPath, prompt string, maxAttempts int) (string, error)
}

// NewPDFParser creates a new PDF parser
func NewPDFParser(geminiClient GeminiAnalyzer) *PDFParser {
	return &PDFParser{
		geminiClient: geminiClient,
	}
}

// ExtractMetadata extracts basic metadata from PDF
func (p *PDFParser) ExtractMetadata(ctx context.Context, pdfPath string) (*PaperMetadata, error) {
	prompt := `Extract the following metadata from this research paper:
- Title
- Authors (comma-separated)
- Abstract
- Publication year

Return ONLY in this exact format:
TITLE: [paper title]
AUTHORS: [author1, author2, ...]
YEAR: [year]
ABSTRACT: [abstract text]

Be concise and accurate.`

	// Use retry logic with up to 3 attempts
	response, err := p.geminiClient.AnalyzePDFWithVisionRetry(ctx, pdfPath, prompt, 3)
	if err != nil {
		return nil, fmt.Errorf("failed to extract metadata: %w", err)
	}

	metadata := parseMetadataResponse(response)
	return metadata, nil
}

// parseMetadataResponse parses the structured response
func parseMetadataResponse(response string) *PaperMetadata {
	// Simple parsing - in production, you'd want more robust parsing
	metadata := &PaperMetadata{}

	lines := splitLines(response)
	for _, line := range lines {
		if len(line) > 7 && line[:6] == "TITLE:" {
			metadata.Title = trim(line[6:])
		} else if len(line) > 9 && line[:8] == "AUTHORS:" {
			authors := trim(line[8:])
			metadata.Authors = splitComma(authors)
		} else if len(line) > 6 && line[:5] == "YEAR:" {
			metadata.Year = trim(line[5:])
		} else if len(line) > 10 && line[:9] == "ABSTRACT:" {
			metadata.Abstract = trim(line[9:])
		}
	}

	return metadata
}

// Helper functions
func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}

func splitComma(s string) []string {
	var parts []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == ',' {
			parts = append(parts, trim(s[start:i]))
			start = i + 1
		}
	}
	if start < len(s) {
		parts = append(parts, trim(s[start:]))
	}
	return parts
}

func trim(s string) string {
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
