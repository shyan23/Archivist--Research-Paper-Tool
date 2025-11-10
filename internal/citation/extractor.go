package citation

import (
	"archivist/internal/analyzer"
	"archivist/internal/graph"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
)

// CitationExtractor handles citation extraction from papers
type CitationExtractor struct {
	geminiClient *analyzer.GeminiClient
}

// NewCitationExtractor creates a new citation extractor
func NewCitationExtractor(geminiClient *analyzer.GeminiClient) *CitationExtractor {
	return &CitationExtractor{
		geminiClient: geminiClient,
	}
}

// ExtractCitations analyzes paper content to find citations
func (ce *CitationExtractor) ExtractCitations(ctx context.Context, latexContent string) (*graph.CitationData, error) {
	log.Println("  📚 Extracting citations from paper...")

	// Step 1: Parse references section
	references, err := ce.ParseReferencesSection(latexContent)
	if err != nil {
		log.Printf("  ⚠️  Reference parsing warning: %v", err)
		references = []graph.Reference{} // Continue with empty references
	}
	log.Printf("  ✓ Found %d references", len(references))

	// Step 2: Extract in-text citations using Gemini
	inTextCitations, err := ce.ExtractInTextCitations(ctx, latexContent, references)
	if err != nil {
		log.Printf("  ⚠️  In-text citation extraction warning: %v", err)
		inTextCitations = []graph.InTextCitation{} // Continue with empty
	}

	// Filter by importance (only high and medium)
	filtered := make([]graph.InTextCitation, 0)
	for _, citation := range inTextCitations {
		if citation.Importance == "high" || citation.Importance == "medium" {
			filtered = append(filtered, citation)
		}
	}
	log.Printf("  ✓ Found %d in-text citations (%d high/medium importance)", len(inTextCitations), len(filtered))

	return &graph.CitationData{
		References:      references,
		InTextCitations: filtered,
		ManualOverrides: make(map[string]string),
	}, nil
}

// ParseReferencesSection extracts formal reference list
func (ce *CitationExtractor) ParseReferencesSection(latexContent string) ([]graph.Reference, error) {
	references := make([]graph.Reference, 0)

	// Look for common reference section patterns
	patterns := []string{
		`\\section\*?\{References\}(.*?)(?:\\section|\\end\{document\}|$)`,
		`\\section\*?\{Bibliography\}(.*?)(?:\\section|\\end\{document\}|$)`,
		`\\begin\{thebibliography\}(.*?)\\end\{thebibliography\}`,
		`\\bibliographystyle.*?\\bibliography`,
	}

	var refSection string
	for _, pattern := range patterns {
		re := regexp.MustCompile(`(?s)` + pattern)
		matches := re.FindStringSubmatch(latexContent)
		if len(matches) > 1 {
			refSection = matches[1]
			break
		}
	}

	if refSection == "" {
		return references, fmt.Errorf("no reference section found")
	}

	// Parse individual references
	// Pattern for numbered references: [1], [2], etc.
	refPattern := regexp.MustCompile(`\[(\d+)\]\s*([^\[]+)`)
	matches := refPattern.FindAllStringSubmatch(refSection, -1)

	for _, match := range matches {
		if len(match) >= 3 {
			index, _ := strconv.Atoi(match[1])
			rawText := strings.TrimSpace(match[2])

			// Try to extract structured info
			ref := graph.Reference{
				Index:   index,
				RawText: rawText,
			}

			// Extract title (usually in quotes or after certain patterns)
			titlePattern := regexp.MustCompile(`"([^"]+)"`)
			if titleMatch := titlePattern.FindStringSubmatch(rawText); len(titleMatch) > 1 {
				ref.Title = titleMatch[1]
			}

			// Extract year (4-digit number)
			yearPattern := regexp.MustCompile(`\b(19|20)\d{2}\b`)
			if yearMatch := yearPattern.FindString(rawText); yearMatch != "" {
				year, _ := strconv.Atoi(yearMatch)
				ref.Year = year
			}

			// Extract authors (simplified: text before title or year)
			if ref.Title != "" {
				authorText := strings.Split(rawText, ref.Title)[0]
				ref.Authors = parseAuthors(authorText)
			}

			references = append(references, ref)
		}
	}

	return references, nil
}

// ExtractInTextCitations finds cited papers in main text using Gemini
func (ce *CitationExtractor) ExtractInTextCitations(ctx context.Context, latexContent string, references []graph.Reference) ([]graph.InTextCitation, error) {
	// Build a simplified version of the paper for analysis (remove refs section)
	mainContent := removeReferencesSection(latexContent)

	// Limit content size for API call (keep first 15000 chars)
	if len(mainContent) > 15000 {
		mainContent = mainContent[:15000]
	}

	// Create reference list summary for the prompt
	refSummary := ""
	for _, ref := range references {
		refSummary += fmt.Sprintf("[%d] %s\n", ref.Index, ref.Title)
		if refSummary == "" {
			refSummary += fmt.Sprintf("[%d] %s\n", ref.Index, truncateString(ref.RawText, 100))
		}
	}

	prompt := fmt.Sprintf(`You are analyzing a research paper to extract citation information.

Below is the main content of the paper (excluding the references section):

%s

Reference List:
%s

Task: Find all citations in the text and rate their importance based on how extensively they are discussed:
- "high": Core methodology, major influence, or extensively discussed
- "medium": Supporting work, comparison, or moderately discussed
- "low": Brief mention, passing reference

For each citation found, extract:
1. The reference number (e.g., [1], [2])
2. A snippet of surrounding context (2-3 sentences)
3. Importance rating

Return a JSON array with this structure:
[
  {
    "reference_index": 1,
    "context": "The attention mechanism introduced in [1] forms the basis of our approach...",
    "importance": "high"
  }
]

Return ONLY the JSON array, no additional text.
`, mainContent, refSummary)

	response, err := ce.geminiClient.GenerateText(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("Gemini API call failed: %w", err)
	}

	// Parse JSON response
	response = extractJSONFromResponse(response)

	var citations []struct {
		ReferenceIndex int    `json:"reference_index"`
		Context        string `json:"context"`
		Importance     string `json:"importance"`
	}

	if err := json.Unmarshal([]byte(response), &citations); err != nil {
		log.Printf("  ⚠️  Failed to parse Gemini response as JSON: %v", err)
		// Try to extract citations from text manually as fallback
		return extractCitationsManually(mainContent), nil
	}

	// Convert to InTextCitation structs
	result := make([]graph.InTextCitation, 0)
	for _, c := range citations {
		result = append(result, graph.InTextCitation{
			ReferenceIndex: c.ReferenceIndex,
			Context:        c.Context,
			Importance:     strings.ToLower(c.Importance),
		})
	}

	return result, nil
}

// Helper functions

func parseAuthors(authorText string) []string {
	// Simple author parsing (split by "and", ",", etc.)
	authorText = strings.TrimSpace(authorText)
	separators := regexp.MustCompile(`\s+and\s+|,\s+|;\s+`)
	authors := separators.Split(authorText, -1)

	result := make([]string, 0)
	for _, author := range authors {
		author = strings.TrimSpace(author)
		if author != "" && len(author) > 2 {
			result = append(result, author)
		}
	}

	if len(result) > 10 {
		return result[:10] // Limit to 10 authors
	}
	return result
}

func removeReferencesSection(content string) string {
	patterns := []string{
		`(?s)\\section\*?\{References\}.*`,
		`(?s)\\section\*?\{Bibliography\}.*`,
		`(?s)\\begin\{thebibliography\}.*?\\end\{thebibliography\}`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		content = re.ReplaceAllString(content, "")
	}

	return content
}

func extractJSONFromResponse(response string) string {
	// Try to find JSON array in the response
	start := strings.Index(response, "[")
	end := strings.LastIndex(response, "]")

	if start != -1 && end != -1 && end > start {
		return response[start : end+1]
	}

	return response
}

func extractCitationsManually(content string) []graph.InTextCitation {
	// Fallback: Find citation patterns manually
	citations := make([]graph.InTextCitation, 0)

	// Pattern: [number] or \cite{...}
	pattern := regexp.MustCompile(`\[(\d+)\]`)
	matches := pattern.FindAllStringSubmatch(content, -1)

	seen := make(map[int]bool)
	for _, match := range matches {
		if len(match) >= 2 {
			index, _ := strconv.Atoi(match[1])
			if !seen[index] {
				seen[index] = true
				citations = append(citations, graph.InTextCitation{
					ReferenceIndex: index,
					Context:        "",
					Importance:     "medium", // Default
				})
			}
		}
	}

	return citations
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
