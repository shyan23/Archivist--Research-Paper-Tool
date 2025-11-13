package graph

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"strings"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

// CitationExtractor handles extraction of citations from papers using LLM
type CitationExtractor struct {
	client *genai.Client
	model  string
}

// NewCitationExtractor creates a new citation extractor
func NewCitationExtractor(apiKey string, model string) (*CitationExtractor, error) {
	ctx := context.Background()

	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return nil, fmt.Errorf("failed to create Gemini client: %w", err)
	}

	return &CitationExtractor{
		client: client,
		model:  model,
	}, nil
}

// Close closes the citation extractor
func (ce *CitationExtractor) Close() error {
	return ce.client.Close()
}

// ExtractCitations extracts citations from paper content
func (ce *CitationExtractor) ExtractCitations(ctx context.Context, paperContent string, paperTitle string) (*CitationData, error) {
	// Step 1: Extract references from bibliography section
	references, err := ce.extractReferences(ctx, paperContent)
	if err != nil {
		log.Printf("Warning: Failed to extract references: %v", err)
		references = []Reference{}
	}

	// Step 2: Extract in-text citations with context and importance
	inTextCitations, err := ce.extractInTextCitations(ctx, paperContent, references)
	if err != nil {
		log.Printf("Warning: Failed to extract in-text citations: %v", err)
		inTextCitations = []InTextCitation{}
	}

	return &CitationData{
		References:      references,
		InTextCitations: inTextCitations,
		ManualOverrides: make(map[string]string),
	}, nil
}

// extractReferences extracts formal references from the bibliography section
func (ce *CitationExtractor) extractReferences(ctx context.Context, paperContent string) ([]Reference, error) {
	prompt := `You are a citation extraction expert. Extract ALL references from the paper's bibliography/references section.

PAPER CONTENT:
` + paperContent + `

TASK:
Extract every reference and return them in JSON format. Each reference should include:
- index: the citation number (e.g., 1, 2, 3)
- authors: array of author names
- title: paper title
- year: publication year
- venue: conference/journal name
- raw_text: the full reference text as it appears

Return ONLY a JSON array of references, no other text:
[
  {
    "index": 1,
    "authors": ["Author A", "Author B"],
    "title": "Paper Title",
    "year": 2023,
    "venue": "Conference Name",
    "raw_text": "Full reference text..."
  },
  ...
]

IMPORTANT:
- Extract ALL references, not just a subset
- If a field is missing, use empty string/array or 0
- Be precise with titles and authors`

	model := ce.client.GenerativeModel(ce.model)
	resp, err := model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return nil, fmt.Errorf("LLM request failed: %w", err)
	}

	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("empty response from LLM")
	}

	// Extract JSON from response
	responseText := fmt.Sprintf("%v", resp.Candidates[0].Content.Parts[0])
	jsonStr := extractJSON(responseText)

	var references []Reference
	if err := json.Unmarshal([]byte(jsonStr), &references); err != nil {
		return nil, fmt.Errorf("failed to parse references JSON: %w", err)
	}

	log.Printf("✓ Extracted %d references from bibliography", len(references))
	return references, nil
}

// extractInTextCitations extracts citations from the main text with context
func (ce *CitationExtractor) extractInTextCitations(ctx context.Context, paperContent string, references []Reference) ([]InTextCitation, error) {
	// Build reference map for lookup
	refMap := make(map[int]Reference)
	for _, ref := range references {
		refMap[ref.Index] = ref
	}

	prompt := `You are a citation analysis expert. Identify ALL in-text citations in the paper and assess their importance.

PAPER CONTENT (excluding references section):
` + paperContent + `

TASK:
For each citation (e.g., [1], [2, 3], [Smith et al., 2023]), extract:
- reference_index: the citation number(s)
- context: 1-2 sentences surrounding the citation
- importance: "high" (foundational work, main comparison), "medium" (related work), or "low" (brief mention)

Return ONLY a JSON array, no other text:
[
  {
    "reference_index": 1,
    "context": "The surrounding text mentioning this work...",
    "importance": "high"
  },
  ...
]

IMPORTANCE CRITERIA:
- HIGH: Paper is central to the methodology, used as baseline, or foundational
- MEDIUM: Related work, alternative approach, or supporting evidence
- LOW: Brief mention, tangential reference

Extract ALL citations from the main text (introduction, methodology, results).`

	model := ce.client.GenerativeModel(ce.model)
	resp, err := model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return nil, fmt.Errorf("LLM request failed: %w", err)
	}

	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("empty response from LLM")
	}

	// Extract JSON from response
	responseText := fmt.Sprintf("%v", resp.Candidates[0].Content.Parts[0])
	jsonStr := extractJSON(responseText)

	var citations []InTextCitation
	if err := json.Unmarshal([]byte(jsonStr), &citations); err != nil {
		return nil, fmt.Errorf("failed to parse in-text citations JSON: %w", err)
	}

	log.Printf("✓ Extracted %d in-text citations", len(citations))
	return citations, nil
}

// ExtractCitationsFromLatex extracts citations from generated LaTeX content
// This is faster and cheaper than analyzing the full PDF
func (ce *CitationExtractor) ExtractCitationsFromLatex(ctx context.Context, latexContent string, paperTitle string) (*CitationData, error) {
	// Use regex to find citation patterns in LaTeX
	citationPattern := regexp.MustCompile(`\\cite\{([^}]+)\}`)
	refPattern := regexp.MustCompile(`\\bibitem\{([^}]+)\}`)

	matches := citationPattern.FindAllStringSubmatch(latexContent, -1)
	refMatches := refPattern.FindAllStringSubmatch(latexContent, -1)

	log.Printf("Found %d citation references in LaTeX for '%s'", len(matches), paperTitle)

	// Build reference list from bibitem entries
	references := make([]Reference, 0)
	for i, match := range refMatches {
		if len(match) > 1 {
			// Extract reference details from LaTeX
			references = append(references, Reference{
				Index:   i + 1,
				Title:   match[1], // Citation key as title placeholder
				RawText: match[0],
			})
		}
	}

	// Build in-text citations
	inTextCitations := make([]InTextCitation, 0)
	citationCounts := make(map[string]int)

	for _, match := range matches {
		if len(match) > 1 {
			citations := strings.Split(match[1], ",")
			for _, cite := range citations {
				cite = strings.TrimSpace(cite)
				citationCounts[cite]++
			}
		}
	}

	// Assign importance based on frequency
	for _, count := range citationCounts {
		importance := "low"
		if count >= 5 {
			importance = "high"
		} else if count >= 2 {
			importance = "medium"
		}

		inTextCitations = append(inTextCitations, InTextCitation{
			ReferenceIndex: 0, // Would need to map citation key to index
			Context:        fmt.Sprintf("Cited %d times in the paper", count),
			Importance:     importance,
		})
	}

	return &CitationData{
		References:      references,
		InTextCitations: inTextCitations,
		ManualOverrides: make(map[string]string),
	}, nil
}

// extractJSON extracts JSON array or object from text that may contain markdown formatting
func extractJSON(text string) string {
	// Remove markdown code blocks
	text = strings.TrimPrefix(text, "```json")
	text = strings.TrimPrefix(text, "```")
	text = strings.TrimSuffix(text, "```")
	text = strings.TrimSpace(text)

	// Find first [ or { and last ] or }
	startIdx := strings.IndexAny(text, "[{")
	if startIdx == -1 {
		return text
	}

	endIdx := strings.LastIndexAny(text, "]}")
	if endIdx == -1 {
		return text[startIdx:]
	}

	return text[startIdx : endIdx+1]
}

// MatchCitationsToGraph attempts to match citation titles to papers in the graph
func (ce *CitationExtractor) MatchCitationsToGraph(ctx context.Context, citations *CitationData, graphBuilder *GraphBuilder) []CitationRelationship {
	relationships := make([]CitationRelationship, 0)

	// For each reference, try to find matching paper in graph
	for _, ref := range citations.References {
		// Try exact title match
		exists, err := graphBuilder.PaperExists(ctx, ref.Title)
		if err != nil || !exists {
			// Try fuzzy matching with LLM (optional, for better matching)
			continue
		}

		// Find corresponding in-text citation
		var importance string = "medium"
		var context string = ""
		for _, inText := range citations.InTextCitations {
			if inText.ReferenceIndex == ref.Index {
				importance = inText.Importance
				context = inText.Context
				break
			}
		}

		relationships = append(relationships, CitationRelationship{
			SourcePaper: "", // To be set by caller
			TargetPaper: ref.Title,
			Importance:  importance,
			Context:     context,
		})
	}

	return relationships
}
