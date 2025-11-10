package rag

import (
	"context"
	"fmt"
	"log"
	"sort"
)

// RetrievalConfig holds configuration for retrieval
type RetrievalConfig struct {
	TopK              int     // Number of chunks to retrieve
	MinScore          float32 // Minimum similarity score threshold
	IncludeSections   []string // Specific sections to prioritize
	MaxContextLength  int     // Maximum total context length in characters
}

// DefaultRetrievalConfig returns default retrieval settings
func DefaultRetrievalConfig() RetrievalConfig {
	return RetrievalConfig{
		TopK:             5,
		MinScore:         0.3,
		MaxContextLength: 8000,
	}
}

// RetrievedContext represents retrieved context with metadata
type RetrievedContext struct {
	Chunks      []SearchResult `json:"chunks"`
	TotalChunks int            `json:"total_chunks"`
	Sources     []string       `json:"sources"`      // Unique paper sources
	Sections    []string       `json:"sections"`     // Unique sections
	Context     string         `json:"context"`      // Combined text context
}

// Retriever handles RAG retrieval operations
type Retriever struct {
	vectorStore VectorStoreInterface
	embedClient *EmbeddingClient
	config      RetrievalConfig
}

// NewRetriever creates a new retriever
func NewRetriever(vectorStore VectorStoreInterface, embedClient *EmbeddingClient, config RetrievalConfig) *Retriever {
	return &Retriever{
		vectorStore: vectorStore,
		embedClient: embedClient,
		config:      config,
	}
}

// Retrieve retrieves relevant context for a query
func (r *Retriever) Retrieve(ctx context.Context, query string, filter map[string]string) (*RetrievedContext, error) {
	if query == "" {
		return nil, fmt.Errorf("empty query")
	}

	// Generate embedding for query
	log.Printf("  🔍 Generating embedding for query: %s", truncateString(query, 50))
	queryEmbedding, err := r.embedClient.GenerateEmbedding(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to generate query embedding: %w", err)
	}

	// Perform vector search
	log.Printf("  📚 Searching vector store (top %d results)...", r.config.TopK)
	results, err := r.vectorStore.Search(ctx, queryEmbedding, r.config.TopK, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to search vector store: %w", err)
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("no relevant chunks found")
	}

	// Filter by minimum score if needed
	filteredResults := r.filterByScore(results)

	// Deduplicate and rank
	rankedResults := r.rankAndDeduplicate(filteredResults)

	// Build context
	context := r.buildContext(rankedResults)

	log.Printf("  ✓ Retrieved %d relevant chunks from %d sources",
		len(rankedResults), len(context.Sources))

	return context, nil
}

// RetrieveFromPaper retrieves context from a specific paper
func (r *Retriever) RetrieveFromPaper(ctx context.Context, query, paperTitle string) (*RetrievedContext, error) {
	filter := map[string]string{"source": paperTitle}
	return r.Retrieve(ctx, query, filter)
}

// RetrieveMultiPaper retrieves context from multiple papers
func (r *Retriever) RetrieveMultiPaper(ctx context.Context, query string, paperTitles []string) (*RetrievedContext, error) {
	if len(paperTitles) == 0 {
		return r.Retrieve(ctx, query, nil)
	}

	// Retrieve from each paper separately
	allResults := []SearchResult{}

	for _, paperTitle := range paperTitles {
		results, err := r.RetrieveFromPaper(ctx, query, paperTitle)
		if err != nil {
			log.Printf("  ⚠️  Warning: Failed to retrieve from %s: %v", paperTitle, err)
			continue
		}
		allResults = append(allResults, results.Chunks...)
	}

	if len(allResults) == 0 {
		return nil, fmt.Errorf("no chunks retrieved from %d papers", len(paperTitles))
	}

	// Sort by score and take top K
	sort.Slice(allResults, func(i, j int) bool {
		return allResults[i].Score > allResults[j].Score
	})

	if len(allResults) > r.config.TopK {
		allResults = allResults[:r.config.TopK]
	}

	// Build context
	context := r.buildContext(allResults)

	return context, nil
}

// RetrieveWithCitations retrieves context and adds citation metadata
func (r *Retriever) RetrieveWithCitations(ctx context.Context, query string, filter map[string]string) (*RetrievedContext, error) {
	context, err := r.Retrieve(ctx, query, filter)
	if err != nil {
		return nil, err
	}

	// Add citation information to each chunk
	for i := range context.Chunks {
		chunk := &context.Chunks[i]
		citation := r.generateCitation(chunk.Document)
		if chunk.Document.Metadata == nil {
			chunk.Document.Metadata = make(map[string]string)
		}
		chunk.Document.Metadata["citation"] = citation
	}

	return context, nil
}

// filterByScore filters results by minimum score threshold
func (r *Retriever) filterByScore(results []SearchResult) []SearchResult {
	if r.config.MinScore <= 0 {
		return results
	}

	filtered := []SearchResult{}
	for _, result := range results {
		if result.Score >= r.config.MinScore {
			filtered = append(filtered, result)
		}
	}

	return filtered
}

// rankAndDeduplicate ranks results and removes duplicates
func (r *Retriever) rankAndDeduplicate(results []SearchResult) []SearchResult {
	if len(results) == 0 {
		return results
	}

	// Sort by score (descending)
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	// Deduplicate by chunk text
	seen := make(map[string]bool)
	unique := []SearchResult{}

	for _, result := range results {
		key := result.Document.Source + ":" + result.Document.ChunkText[:min(50, len(result.Document.ChunkText))]
		if !seen[key] {
			seen[key] = true
			unique = append(unique, result)
		}
	}

	return unique
}

// buildContext builds the final context from search results
func (r *Retriever) buildContext(results []SearchResult) *RetrievedContext {
	context := &RetrievedContext{
		Chunks:      results,
		TotalChunks: len(results),
		Sources:     []string{},
		Sections:    []string{},
	}

	// Track unique sources and sections
	sourceSet := make(map[string]bool)
	sectionSet := make(map[string]bool)

	// Build combined context text
	var contextText string
	currentLength := 0

	for i, result := range results {
		doc := result.Document

		// Track unique values
		if !sourceSet[doc.Source] {
			sourceSet[doc.Source] = true
			context.Sources = append(context.Sources, doc.Source)
		}

		if doc.Section != "" && !sectionSet[doc.Section] {
			sectionSet[doc.Section] = true
			context.Sections = append(context.Sections, doc.Section)
		}

		// Add to context text with citation
		chunkHeader := fmt.Sprintf("\n[Source: %s", doc.Source)
		if doc.Section != "" {
			chunkHeader += fmt.Sprintf(", Section: %s", doc.Section)
		}
		chunkHeader += fmt.Sprintf(", Chunk %d]\n", i+1)

		chunkText := chunkHeader + doc.ChunkText + "\n"

		// Check context length limit
		if r.config.MaxContextLength > 0 && currentLength+len(chunkText) > r.config.MaxContextLength {
			log.Printf("  ⚠️  Reached max context length (%d chars), truncating at %d chunks",
				r.config.MaxContextLength, i)
			break
		}

		contextText += chunkText
		currentLength += len(chunkText)
	}

	context.Context = contextText

	return context
}

// generateCitation generates a citation string for a document
func (r *Retriever) generateCitation(doc VectorDocument) string {
	citation := fmt.Sprintf("Source: %s", doc.Source)

	if doc.Section != "" {
		citation += fmt.Sprintf(", Section: %s", doc.Section)
	}

	citation += fmt.Sprintf(", Chunk %d", doc.ChunkIndex)

	return citation
}

// Helper functions

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
