package analyzer

import (
	"archivist/internal/search"
	"context"
	"fmt"
	"log"
	"strings"
	"time"
)

// PaperEssence represents the key characteristics of a paper
type PaperEssence struct {
	MainMethodology    string   // Primary approach/technique
	KeyConcepts        []string // Essential concepts and terms
	ProblemDomain      string   // What problem it solves
	Techniques         []string // Specific techniques used
	RelatedFields      []string // Related research areas
	NovelContributions []string // What makes it unique
}

// SimilarPaperFinder finds papers similar to a given paper
type SimilarPaperFinder struct {
	analyzer      *Analyzer
	searchClient  *search.Client
	serviceURL    string
}

// NewSimilarPaperFinder creates a new similar paper finder
func NewSimilarPaperFinder(analyzer *Analyzer, searchServiceURL string) *SimilarPaperFinder {
	if searchServiceURL == "" {
		searchServiceURL = "http://localhost:8000"
	}

	return &SimilarPaperFinder{
		analyzer:     analyzer,
		searchClient: search.NewClient(searchServiceURL),
		serviceURL:   searchServiceURL,
	}
}

// ExtractEssence analyzes a paper and extracts its essence
func (spf *SimilarPaperFinder) ExtractEssence(ctx context.Context, pdfPath string) (*PaperEssence, error) {
	log.Println("📊 Extracting paper essence...")

	prompt := `Analyze this research paper and extract its ESSENCE in a structured format.

Output EXACTLY in this format (no additional text):

MAIN_METHODOLOGY: [One sentence describing the primary approach/technique]
KEY_CONCEPTS: [Comma-separated list of 5-10 essential technical concepts/terms]
PROBLEM_DOMAIN: [One sentence describing what problem this solves]
TECHNIQUES: [Comma-separated list of specific techniques used]
RELATED_FIELDS: [Comma-separated list of related research areas]
NOVEL_CONTRIBUTIONS: [Comma-separated list of what makes this paper unique]

Be specific and technical. Use actual terminology from the paper.`

	startTime := time.Now()
	// Use retry logic with exponential backoff (max 5 attempts)
	result, err := spf.analyzer.client.AnalyzePDFWithVisionRetry(ctx, pdfPath, prompt, 5)
	if err != nil {
		return nil, fmt.Errorf("failed to extract essence: %w", err)
	}

	log.Printf("✓ Essence extracted (%.2fs)", time.Since(startTime).Seconds())

	// Parse the result
	essence := &PaperEssence{}
	lines := strings.Split(result, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		switch key {
		case "MAIN_METHODOLOGY":
			essence.MainMethodology = value
		case "KEY_CONCEPTS":
			essence.KeyConcepts = splitAndTrim(value)
		case "PROBLEM_DOMAIN":
			essence.ProblemDomain = value
		case "TECHNIQUES":
			essence.Techniques = splitAndTrim(value)
		case "RELATED_FIELDS":
			essence.RelatedFields = splitAndTrim(value)
		case "NOVEL_CONTRIBUTIONS":
			essence.NovelContributions = splitAndTrim(value)
		}
	}

	return essence, nil
}

// FindSimilarPapers searches for papers similar to the given essence
func (spf *SimilarPaperFinder) FindSimilarPapers(ctx context.Context, essence *PaperEssence, maxResults int) (*search.SearchResponse, error) {
	if !spf.searchClient.IsServiceRunning() {
		return nil, fmt.Errorf("search service is not running at %s", spf.serviceURL)
	}

	log.Println("🔍 Searching for similar papers...")

	// Build search query from essence
	query := spf.buildSearchQuery(essence)
	log.Printf("   Query: %s", query)

	searchQuery := &search.SearchQuery{
		Query:      query,
		MaxResults: maxResults,
		Sources:    []string{}, // Search all sources
	}

	results, err := spf.searchClient.Search(searchQuery)
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}

	log.Printf("✓ Found %d similar papers", results.Total)

	return results, nil
}

// FindSimilarPapersFromPDF is a convenience method that extracts essence and searches in one call
func (spf *SimilarPaperFinder) FindSimilarPapersFromPDF(ctx context.Context, pdfPath string, maxResults int) (*PaperEssence, *search.SearchResponse, error) {
	essence, err := spf.ExtractEssence(ctx, pdfPath)
	if err != nil {
		return nil, nil, err
	}

	results, err := spf.FindSimilarPapers(ctx, essence, maxResults)
	if err != nil {
		return essence, nil, err
	}

	return essence, results, nil
}

// buildSearchQuery constructs a search query from paper essence
func (spf *SimilarPaperFinder) buildSearchQuery(essence *PaperEssence) string {
	var queryParts []string

	// Add main methodology (highest weight)
	if essence.MainMethodology != "" {
		queryParts = append(queryParts, essence.MainMethodology)
	}

	// Add top key concepts (3-5)
	conceptsToAdd := min(5, len(essence.KeyConcepts))
	for i := 0; i < conceptsToAdd; i++ {
		queryParts = append(queryParts, essence.KeyConcepts[i])
	}

	// Add top techniques (2-3)
	techniquesToAdd := min(3, len(essence.Techniques))
	for i := 0; i < techniquesToAdd; i++ {
		queryParts = append(queryParts, essence.Techniques[i])
	}

	// Join all parts
	query := strings.Join(queryParts, " ")

	// Clean up query
	query = strings.ReplaceAll(query, "[", "")
	query = strings.ReplaceAll(query, "]", "")
	query = strings.TrimSpace(query)

	return query
}

// splitAndTrim splits a comma-separated string and trims each element
func splitAndTrim(s string) []string {
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))

	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		// Remove brackets if present
		trimmed = strings.Trim(trimmed, "[]")
		trimmed = strings.TrimSpace(trimmed)

		if trimmed != "" {
			result = append(result, trimmed)
		}
	}

	return result
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
