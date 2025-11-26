package graph

import (
	"context"
	"fmt"
	"log"
	"math"
	"sort"
	"strings"

	"archivist/internal/rag"
	"archivist/internal/vectorstore"

	qdrant "github.com/qdrant/go-client/qdrant"
)

// HybridSearchEngine combines vector, graph, and keyword search
type HybridSearchEngine struct {
	enhancedBuilder *EnhancedGraphBuilder
	embeddingClient *rag.EmbeddingClient
}

// NewHybridSearchEngine creates a new hybrid search engine
func NewHybridSearchEngine(enhancedBuilder *EnhancedGraphBuilder, embeddingClient *rag.EmbeddingClient) *HybridSearchEngine {
	return &HybridSearchEngine{
		enhancedBuilder: enhancedBuilder,
		embeddingClient: embeddingClient,
	}
}

// Search performs hybrid search combining multiple strategies
func (hse *HybridSearchEngine) Search(ctx context.Context, query *vectorstore.HybridSearchQuery) ([]*vectorstore.HybridSearchResult, error) {
	log.Printf("Starting hybrid search for query: '%s'", query.Query)

	// Step 1: Vector Search
	vectorResults, err := hse.vectorSearch(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("vector search failed: %w", err)
	}
	log.Printf("✓ Vector search returned %d results", len(vectorResults))

	// Step 2: Graph Traversal Search
	graphResults, err := hse.graphSearch(ctx, query)
	if err != nil {
		log.Printf("Warning: Graph search failed: %v", err)
		graphResults = make(map[string]float64)
	}
	log.Printf("✓ Graph search returned %d results", len(graphResults))

	// Step 3: Keyword Search
	keywordResults := hse.keywordSearch(query.Query, vectorResults)
	log.Printf("✓ Keyword search scored %d results", len(keywordResults))

	// Step 4: Combine scores using weighted fusion
	hybridResults := hse.combineResults(vectorResults, graphResults, keywordResults, query)

	// Step 5: Sort and return top-K
	sort.Slice(hybridResults, func(i, j int) bool {
		return hybridResults[i].HybridScore > hybridResults[j].HybridScore
	})

	if len(hybridResults) > query.TopK {
		hybridResults = hybridResults[:query.TopK]
	}

	// Assign ranks
	for i := range hybridResults {
		hybridResults[i].Rank = i + 1
	}

	log.Printf("✓ Hybrid search complete: returning %d results", len(hybridResults))
	return hybridResults, nil
}

// vectorSearch performs semantic vector search using Qdrant
func (hse *HybridSearchEngine) vectorSearch(ctx context.Context, query *vectorstore.HybridSearchQuery) ([]*vectorstore.SearchResult, error) {
	// Generate embedding for query
	var queryVector []float32
	var err error

	if query.QueryVector != nil && len(query.QueryVector) > 0 {
		queryVector = query.QueryVector
	} else {
		queryVector, err = hse.embeddingClient.GenerateEmbedding(ctx, query.Query)
		if err != nil {
			return nil, fmt.Errorf("failed to generate query embedding: %w", err)
		}
	}

	// Build filter from query filters
	var filter *qdrant.Filter
	if len(query.Filters) > 0 {
		filter = buildQdrantFilter(query.Filters)
	}

	// Perform vector search
	searchQuery := &vectorstore.SearchQuery{
		Vector:         queryVector,
		Limit:          uint64(query.TopK * 3), // Get more for reranking
		ScoreThreshold: 0.5,                    // Minimum similarity
		Filter:         filter,
	}

	results, err := hse.enhancedBuilder.vectorStore.Search(ctx, searchQuery)
	if err != nil {
		return nil, fmt.Errorf("Qdrant search failed: %w", err)
	}

	return results, nil
}

// graphSearch performs graph-based search using Neo4j
func (hse *HybridSearchEngine) graphSearch(ctx context.Context, query *vectorstore.HybridSearchQuery) (map[string]float64, error) {
	// First, find seed papers using keyword matching
	seedPapers := hse.findSeedPapers(ctx, query.Query)
	if len(seedPapers) == 0 {
		return make(map[string]float64), nil
	}

	log.Printf("Found %d seed papers for graph traversal", len(seedPapers))

	// Traverse the graph from seed papers
	paperScores := make(map[string]float64)

	for _, seedPaper := range seedPapers {
		// Start with seed paper
		paperScores[seedPaper] = 1.0

		// Traverse citations (papers this paper cites)
		cited := hse.traverseCitations(ctx, seedPaper, query.TraversalDepth)
		for paper, score := range cited {
			if existing, ok := paperScores[paper]; ok {
				paperScores[paper] = math.Max(existing, score)
			} else {
				paperScores[paper] = score
			}
		}

		// Traverse similar papers
		similar := hse.traverseSimilar(ctx, seedPaper, query.TraversalDepth)
		for paper, score := range similar {
			if existing, ok := paperScores[paper]; ok {
				paperScores[paper] = math.Max(existing, score)
			} else {
				paperScores[paper] = score
			}
		}
	}

	return paperScores, nil
}

// keywordSearch performs simple keyword matching
func (hse *HybridSearchEngine) keywordSearch(queryText string, vectorResults []*vectorstore.SearchResult) map[string]float64 {
	scores := make(map[string]float64)
	queryTokens := tokenize(strings.ToLower(queryText))

	for _, result := range vectorResults {
		content := result.Payload["content"].GetStringValue()
		paperTitle := result.Payload["paper_title"].GetStringValue()

		contentLower := strings.ToLower(content)
		matchCount := 0

		for _, token := range queryTokens {
			if strings.Contains(contentLower, token) {
				matchCount++
			}
		}

		if matchCount > 0 {
			score := float64(matchCount) / float64(len(queryTokens))
			if existing, ok := scores[paperTitle]; ok {
				scores[paperTitle] = math.Max(existing, score)
			} else {
				scores[paperTitle] = score
			}
		}
	}

	return scores
}

// combineResults fuses results from multiple search strategies
func (hse *HybridSearchEngine) combineResults(
	vectorResults []*vectorstore.SearchResult,
	graphResults map[string]float64,
	keywordResults map[string]float64,
	query *vectorstore.HybridSearchQuery,
) []*vectorstore.HybridSearchResult {
	resultsMap := make(map[string]*vectorstore.HybridSearchResult)

	// Add vector results
	for _, vr := range vectorResults {
		paperTitle := vr.Payload["paper_title"].GetStringValue()
		content := vr.Payload["content"].GetStringValue()

		if _, exists := resultsMap[paperTitle]; !exists {
			resultsMap[paperTitle] = &vectorstore.HybridSearchResult{
				PaperTitle:   paperTitle,
				ChunkContent: content,
				Metadata:     extractMetadata(vr.Payload),
			}
		}

		resultsMap[paperTitle].VectorScore = math.Max(resultsMap[paperTitle].VectorScore, vr.Score)
	}

	// Add graph scores
	for paper, score := range graphResults {
		if _, exists := resultsMap[paper]; !exists {
			resultsMap[paper] = &vectorstore.HybridSearchResult{
				PaperTitle: paper,
			}
		}
		resultsMap[paper].GraphScore = score
	}

	// Add keyword scores
	for paper, score := range keywordResults {
		if result, exists := resultsMap[paper]; exists {
			result.KeywordScore = score
		}
	}

	// Calculate hybrid score
	results := make([]*vectorstore.HybridSearchResult, 0, len(resultsMap))
	for _, result := range resultsMap {
		result.HybridScore = (result.VectorScore * query.VectorWeight) +
			(result.GraphScore * query.GraphWeight) +
			(result.KeywordScore * query.KeywordWeight)
		results = append(results, result)
	}

	return results
}

// findSeedPapers finds initial papers to start graph traversal
func (hse *HybridSearchEngine) findSeedPapers(ctx context.Context, query string) []string {
	// Use simple keyword matching on paper titles
	// In production, this could use full-text search or vector similarity
	tokens := tokenize(strings.ToLower(query))

	// Get all papers from graph (for small scale)
	// For larger scale, use a proper search index
	limit := uint32(100)
	points, err := hse.enhancedBuilder.vectorStore.ScrollPoints(ctx, limit, nil)
	if err != nil {
		log.Printf("Warning: Failed to scroll points: %v", err)
		return []string{}
	}

	seedPapers := make(map[string]int)
	for _, point := range points {
		if point.Payload == nil {
			continue
		}

		paperTitle := point.Payload["paper_title"].GetStringValue()
		content := point.Payload["content"].GetStringValue()

		titleLower := strings.ToLower(paperTitle)
		contentLower := strings.ToLower(content)

		matchCount := 0
		for _, token := range tokens {
			if strings.Contains(titleLower, token) {
				matchCount += 2 // Title matches count more
			} else if strings.Contains(contentLower, token) {
				matchCount++
			}
		}

		if matchCount > 0 {
			if existing, ok := seedPapers[paperTitle]; ok {
				seedPapers[paperTitle] = existing + matchCount
			} else {
				seedPapers[paperTitle] = matchCount
			}
		}
	}

	// Return top seed papers
	type paperScore struct {
		title string
		score int
	}
	papers := make([]paperScore, 0, len(seedPapers))
	for title, score := range seedPapers {
		papers = append(papers, paperScore{title, score})
	}
	sort.Slice(papers, func(i, j int) bool {
		return papers[i].score > papers[j].score
	})

	maxSeeds := 5
	if len(papers) > maxSeeds {
		papers = papers[:maxSeeds]
	}

	seeds := make([]string, len(papers))
	for i, p := range papers {
		seeds[i] = p.title
	}

	return seeds
}

// traverseCitations traverses citation relationships in the graph
func (hse *HybridSearchEngine) traverseCitations(ctx context.Context, startPaper string, maxDepth int) map[string]float64 {
	scores := make(map[string]float64)
	// This is a simplified version. In production, use Cypher queries
	// to traverse the graph efficiently

	// TODO: Implement actual graph traversal using Neo4j Cypher
	// For now, return empty map
	return scores
}

// traverseSimilar traverses similarity relationships
func (hse *HybridSearchEngine) traverseSimilar(ctx context.Context, startPaper string, maxDepth int) map[string]float64 {
	scores := make(map[string]float64)
	// This is a simplified version. In production, use Cypher queries

	// TODO: Implement actual similarity traversal using Neo4j Cypher
	return scores
}

// buildQdrantFilter converts generic filters to Qdrant filter format
func buildQdrantFilter(filters map[string]interface{}) *qdrant.Filter {
	conditions := make([]*qdrant.Condition, 0)

	for key, value := range filters {
		switch v := value.(type) {
		case string:
			conditions = append(conditions, &qdrant.Condition{
				ConditionOneOf: &qdrant.Condition_Field{
					Field: &qdrant.FieldCondition{
						Key: key,
						Match: &qdrant.Match{
							MatchValue: &qdrant.Match_Keyword{Keyword: v},
						},
					},
				},
			})
		case int, int64:
			intVal := v.(int64)
			conditions = append(conditions, &qdrant.Condition{
				ConditionOneOf: &qdrant.Condition_Field{
					Field: &qdrant.FieldCondition{
						Key: key,
						Match: &qdrant.Match{
							MatchValue: &qdrant.Match_Integer{Integer: intVal},
						},
					},
				},
			})
		}
	}

	if len(conditions) == 0 {
		return nil
	}

	return &qdrant.Filter{
		Must: conditions,
	}
}

// extractMetadata extracts metadata from Qdrant payload
func extractMetadata(payload map[string]*qdrant.Value) map[string]interface{} {
	metadata := make(map[string]interface{})

	for key, value := range payload {
		switch value.GetKind().(type) {
		case *qdrant.Value_StringValue:
			metadata[key] = value.GetStringValue()
		case *qdrant.Value_IntegerValue:
			metadata[key] = value.GetIntegerValue()
		case *qdrant.Value_DoubleValue:
			metadata[key] = value.GetDoubleValue()
		case *qdrant.Value_BoolValue:
			metadata[key] = value.GetBoolValue()
		case *qdrant.Value_ListValue:
			list := value.GetListValue()
			if list != nil {
				items := make([]string, len(list.Values))
				for i, v := range list.Values {
					items[i] = v.GetStringValue()
				}
				metadata[key] = items
			}
		}
	}

	return metadata
}

// tokenize splits text into tokens for keyword matching
func tokenize(text string) []string {
	// Simple tokenization - split by whitespace and remove punctuation
	tokens := strings.Fields(text)
	cleaned := make([]string, 0, len(tokens))

	for _, token := range tokens {
		// Remove common stop words
		if isStopWord(token) {
			continue
		}
		cleaned = append(cleaned, token)
	}

	return cleaned
}

// isStopWord checks if a word is a common stop word
func isStopWord(word string) bool {
	stopWords := map[string]bool{
		"the": true, "a": true, "an": true, "and": true, "or": true,
		"but": true, "in": true, "on": true, "at": true, "to": true,
		"for": true, "of": true, "with": true, "by": true, "is": true,
		"are": true, "was": true, "were": true,
	}
	return stopWords[word]
}
