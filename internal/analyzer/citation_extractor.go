package analyzer

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"
)

// CitedPaper represents a paper referenced in the text
type CitedPaper struct {
	Title          string   // Paper title
	Authors        []string // Authors
	Year           string   // Publication year
	Venue          string   // Conference/Journal
	Context        string   // Where/why it was cited
	CitationCount  int      // Number of times cited in paper
	IsFoundational bool     // Whether it's a foundational paper for this work
}

// CitationExtractor extracts all papers referenced throughout a document
type CitationExtractor struct {
	analyzer *Analyzer
}

// NewCitationExtractor creates a new citation extractor
func NewCitationExtractor(analyzer *Analyzer) *CitationExtractor {
	return &CitationExtractor{
		analyzer: analyzer,
	}
}

// ExtractAllCitations extracts all papers mentioned throughout the document
func (ce *CitationExtractor) ExtractAllCitations(ctx context.Context, pdfPath string) ([]CitedPaper, error) {
	log.Println("📚 Extracting all citations from paper...")

	prompt := `Analyze this research paper and extract ALL papers that are cited/referenced throughout the document.

IMPORTANT:
- Look for citations in the ENTIRE document, not just the references section
- Include papers mentioned in the introduction, methodology, related work, and throughout the text
- Identify which papers are FOUNDATIONAL (directly helped or influenced this work)
- Note the context of why each paper was cited

Output EXACTLY in this format (one paper per block, separated by "---"):

TITLE: [Paper title]
AUTHORS: [Comma-separated list of authors]
YEAR: [Publication year]
VENUE: [Conference/Journal if mentioned]
CONTEXT: [Brief description of why/where this paper was cited]
IS_FOUNDATIONAL: [YES if this paper directly helped/influenced the current work, NO otherwise]
---

Example:
TITLE: Attention Is All You Need
AUTHORS: Vaswani, Shazeer, Parmar, Uszkoreit, Jones, Gomez, Kaiser, Polosukhin
YEAR: 2017
VENUE: NeurIPS
CONTEXT: Introduced the Transformer architecture that this paper builds upon
IS_FOUNDATIONAL: YES
---

Be thorough. Extract ALL cited papers, even if only mentioned once.`

	startTime := time.Now()
	result, err := ce.analyzer.client.AnalyzePDFWithVision(ctx, pdfPath, prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to extract citations: %w", err)
	}

	log.Printf("✓ Citations extracted (%.2fs)", time.Since(startTime).Seconds())

	// Parse the result
	citations := ce.parseCitations(result)

	// Count citation occurrences in the text
	citations = ce.enrichWithCounts(ctx, pdfPath, citations)

	log.Printf("✓ Found %d cited papers", len(citations))

	return citations, nil
}

// ExtractFoundationalPapers extracts only the papers that directly influenced the current work
func (ce *CitationExtractor) ExtractFoundationalPapers(ctx context.Context, pdfPath string) ([]CitedPaper, error) {
	allCitations, err := ce.ExtractAllCitations(ctx, pdfPath)
	if err != nil {
		return nil, err
	}

	foundational := make([]CitedPaper, 0)
	for _, citation := range allCitations {
		if citation.IsFoundational {
			foundational = append(foundational, citation)
		}
	}

	log.Printf("✓ Found %d foundational papers", len(foundational))

	return foundational, nil
}

// ExtractMostCitedPapers returns the most frequently cited papers
func (ce *CitationExtractor) ExtractMostCitedPapers(ctx context.Context, pdfPath string, topN int) ([]CitedPaper, error) {
	allCitations, err := ce.ExtractAllCitations(ctx, pdfPath)
	if err != nil {
		return nil, err
	}

	// Sort by citation count (simple bubble sort since list is small)
	for i := 0; i < len(allCitations); i++ {
		for j := i + 1; j < len(allCitations); j++ {
			if allCitations[j].CitationCount > allCitations[i].CitationCount {
				allCitations[i], allCitations[j] = allCitations[j], allCitations[i]
			}
		}
	}

	// Return top N
	if topN > len(allCitations) {
		topN = len(allCitations)
	}

	return allCitations[:topN], nil
}

// parseCitations parses the LLM output into structured citations
func (ce *CitationExtractor) parseCitations(result string) []CitedPaper {
	citations := make([]CitedPaper, 0)

	// Split by "---" delimiter
	blocks := strings.Split(result, "---")

	for _, block := range blocks {
		block = strings.TrimSpace(block)
		if block == "" {
			continue
		}

		citation := CitedPaper{}
		lines := strings.Split(block, "\n")

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
			case "TITLE":
				citation.Title = value
			case "AUTHORS":
				citation.Authors = splitAndTrimAuthors(value)
			case "YEAR":
				citation.Year = value
			case "VENUE":
				citation.Venue = value
			case "CONTEXT":
				citation.Context = value
			case "IS_FOUNDATIONAL":
				citation.IsFoundational = strings.ToUpper(value) == "YES"
			}
		}

		// Only add if we have at least a title
		if citation.Title != "" {
			citations = append(citations, citation)
		}
	}

	return citations
}

// enrichWithCounts counts how many times each paper is cited in the document
func (ce *CitationExtractor) enrichWithCounts(ctx context.Context, pdfPath string, citations []CitedPaper) []CitedPaper {
	// This is a simplified version - ideally we'd extract the full text and count
	// For now, we'll use a heuristic based on title/author mentions

	// Create a prompt to get citation counts
	titles := make([]string, 0)
	for _, cit := range citations {
		titles = append(titles, cit.Title)
	}

	if len(titles) == 0 {
		return citations
	}

	// Build prompt to count citations
	countPrompt := fmt.Sprintf(`Count how many times each of these papers is cited in this document:

Papers:
%s

Output format:
[Title]: [Count]

Be accurate - count all citations, including in-text citations and reference mentions.`, strings.Join(titles, "\n"))

	result, err := ce.analyzer.client.AnalyzePDFWithVision(ctx, pdfPath, countPrompt)
	if err != nil {
		log.Printf("Warning: Could not enrich with citation counts: %v", err)
		return citations
	}

	// Parse counts
	countMap := parseCounts(result)

	// Update citations with counts
	for i := range citations {
		if count, found := countMap[citations[i].Title]; found {
			citations[i].CitationCount = count
		} else {
			citations[i].CitationCount = 1 // Default to 1 if not found
		}
	}

	return citations
}

// parseCounts parses the count output
func parseCounts(result string) map[string]int {
	countMap := make(map[string]int)

	lines := strings.Split(result, "\n")
	countRegex := regexp.MustCompile(`^(.+?):\s*(\d+)`)

	for _, line := range lines {
		line = strings.TrimSpace(line)
		matches := countRegex.FindStringSubmatch(line)

		if len(matches) == 3 {
			title := strings.TrimSpace(matches[1])
			count := 0
			fmt.Sscanf(matches[2], "%d", &count)
			countMap[title] = count
		}
	}

	return countMap
}

// splitAndTrimAuthors splits author names
func splitAndTrimAuthors(s string) []string {
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))

	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}

	return result
}
