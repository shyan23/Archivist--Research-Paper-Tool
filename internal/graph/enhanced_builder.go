package graph

import (
	"context"
	"fmt"
	"log"
	"sync"

	"archivist/internal/rag"
	"archivist/internal/vectorstore"
)

// EnhancedGraphBuilder combines Neo4j graph with Qdrant vector store
type EnhancedGraphBuilder struct {
	graphBuilder    *GraphBuilder
	vectorStore     *vectorstore.QdrantClient
	embeddingClient *rag.EmbeddingClient
	citationExtractor *CitationExtractor
	mu              sync.Mutex
}

// NewEnhancedGraphBuilder creates a new enhanced graph builder
func NewEnhancedGraphBuilder(
	graphConfig *GraphConfig,
	vectorConfig *vectorstore.QdrantConfig,
	apiKey string,
	model string,
) (*EnhancedGraphBuilder, error) {
	// Initialize graph builder
	graphBuilder, err := NewGraphBuilder(graphConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create graph builder: %w", err)
	}

	// Initialize vector store
	vectorStore, err := vectorstore.NewQdrantClient(vectorConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create vector store: %w", err)
	}

	// Initialize embedding client
	embeddingClient, err := rag.NewEmbeddingClient(apiKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create embedding client: %w", err)
	}

	// Initialize citation extractor
	citationExtractor, err := NewCitationExtractor(apiKey, model)
	if err != nil {
		return nil, fmt.Errorf("failed to create citation extractor: %w", err)
	}

	return &EnhancedGraphBuilder{
		graphBuilder:      graphBuilder,
		vectorStore:       vectorStore,
		embeddingClient:   embeddingClient,
		citationExtractor: citationExtractor,
	}, nil
}

// Close closes all connections
func (egb *EnhancedGraphBuilder) Close(ctx context.Context) error {
	var errs []error

	if err := egb.graphBuilder.Close(ctx); err != nil {
		errs = append(errs, err)
	}
	if err := egb.vectorStore.Close(); err != nil {
		errs = append(errs, err)
	}
	if err := egb.embeddingClient.Close(); err != nil {
		errs = append(errs, err)
	}
	if err := egb.citationExtractor.Close(); err != nil {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors closing connections: %v", errs)
	}
	return nil
}

// AddPaperWithEmbeddings adds a paper to both graph and vector store
func (egb *EnhancedGraphBuilder) AddPaperWithEmbeddings(ctx context.Context, paper *PaperNode, paperContent string, chunks []string) error {
	egb.mu.Lock()
	defer egb.mu.Unlock()

	// Step 1: Add paper to Neo4j graph
	if err := egb.graphBuilder.AddPaper(ctx, paper); err != nil {
		return fmt.Errorf("failed to add paper to graph: %w", err)
	}

	// Step 2: Generate embeddings for chunks
	embeddings, err := egb.embeddingClient.GenerateBatchEmbeddings(ctx, chunks)
	if err != nil {
		return fmt.Errorf("failed to generate embeddings: %w", err)
	}

	// Step 3: Create paper chunks with metadata
	points := make([]*vectorstore.Point, len(chunks))
	for i, chunk := range chunks {
		paperChunk := &vectorstore.PaperChunk{
			ID:            fmt.Sprintf("%s_chunk_%d", paper.Title, i),
			PaperTitle:    paper.Title,
			PaperID:       paper.Title, // Use title as ID for now
			ChunkIndex:    i,
			ChunkType:     determineChunkType(i, len(chunks)),
			Content:       chunk,
			Embedding:     embeddings[i],
			Year:          paper.Year,
			Authors:       paper.Authors,
			Methodologies: paper.Methodologies,
			Datasets:      paper.Datasets,
			Metrics:       paper.Metrics,
		}
		points[i] = paperChunk.ToQdrantPoint()
	}

	// Step 4: Upsert to Qdrant
	if err := egb.vectorStore.UpsertBatch(ctx, points); err != nil {
		return fmt.Errorf("failed to upsert embeddings: %w", err)
	}

	log.Printf("✓ Added paper '%s' with %d chunks to knowledge graph", paper.Title, len(chunks))
	return nil
}

// ExtractAndAddCitations extracts citations and adds them to the graph
func (egb *EnhancedGraphBuilder) ExtractAndAddCitations(ctx context.Context, paperTitle string, latexContent string) error {
	// Extract citations
	citations, err := egb.citationExtractor.ExtractCitationsFromLatex(ctx, latexContent, paperTitle)
	if err != nil {
		return fmt.Errorf("failed to extract citations: %w", err)
	}

	// Match citations to existing papers in graph
	relationships := egb.citationExtractor.MatchCitationsToGraph(ctx, citations, egb.graphBuilder)

	// Add citation relationships
	for _, rel := range relationships {
		rel.SourcePaper = paperTitle
		if err := egb.graphBuilder.AddCitation(ctx, &rel); err != nil {
			log.Printf("Warning: Failed to add citation %s -> %s: %v", rel.SourcePaper, rel.TargetPaper, err)
		}
	}

	log.Printf("✓ Added %d citation relationships for '%s'", len(relationships), paperTitle)
	return nil
}

// ComputePaperSimilarities computes semantic similarity between papers
func (egb *EnhancedGraphBuilder) ComputePaperSimilarities(ctx context.Context, topK int) error {
	// Get all papers from graph
	stats, err := egb.graphBuilder.GetStats(ctx)
	if err != nil {
		return fmt.Errorf("failed to get graph stats: %w", err)
	}

	log.Printf("Computing similarities for %d papers...", stats.PaperCount)

	// For each paper, find top-K similar papers using vector search
	// This is a simplified version - in production, you'd batch this
	limit := uint32(100)
	points, err := egb.vectorStore.ScrollPoints(ctx, limit, nil)
	if err != nil {
		return fmt.Errorf("failed to scroll points: %w", err)
	}

	// Group chunks by paper
	paperVectors := make(map[string][]float32)
	for _, point := range points {
		if point.Payload == nil {
			continue
		}

		paperTitle := ""
		if val, ok := point.Payload["paper_title"]; ok {
			paperTitle = val.GetStringValue()
		}

		if paperTitle == "" {
			continue
		}

		// Average embeddings for the paper (simple approach)
		if _, exists := paperVectors[paperTitle]; !exists {
			paperVectors[paperTitle] = point.Vectors.GetVector().Data
		}
	}

	// For each paper, find similar papers
	for paperTitle, vector := range paperVectors {
		searchQuery := &vectorstore.SearchQuery{
			Vector:         vector,
			Limit:          uint64(topK + 1), // +1 because it will include itself
			ScoreThreshold: 0.7,               // Minimum similarity
		}

		results, err := egb.vectorStore.Search(ctx, searchQuery)
		if err != nil {
			log.Printf("Warning: Failed to search for '%s': %v", paperTitle, err)
			continue
		}

		// Add similarity relationships (skip self)
		for _, result := range results {
			targetPaper := result.Payload["paper_title"].GetStringValue()
			if targetPaper == paperTitle {
				continue
			}

			similarity := &SimilarityRelationship{
				Paper1: paperTitle,
				Paper2: targetPaper,
				Score:  result.Score,
				Basis:  "semantic",
			}

			if err := egb.graphBuilder.AddSimilarity(ctx, similarity); err != nil {
				log.Printf("Warning: Failed to add similarity %s <-> %s: %v", paperTitle, targetPaper, err)
			}
		}
	}

	log.Printf("✓ Computed similarities for %d papers", len(paperVectors))
	return nil
}

// DeletePaper removes a paper from both graph and vector store
func (egb *EnhancedGraphBuilder) DeletePaper(ctx context.Context, paperTitle string) error {
	// Delete from Neo4j
	if err := egb.graphBuilder.DeletePaper(ctx, paperTitle); err != nil {
		return fmt.Errorf("failed to delete from graph: %w", err)
	}

	// Delete from Qdrant
	if err := egb.vectorStore.DeleteByPaperTitle(ctx, paperTitle); err != nil {
		return fmt.Errorf("failed to delete from vector store: %w", err)
	}

	log.Printf("✓ Deleted paper '%s' from knowledge graph", paperTitle)
	return nil
}

// GetGraphStats returns combined statistics
func (egb *EnhancedGraphBuilder) GetGraphStats(ctx context.Context) (*EnhancedGraphStats, error) {
	// Get Neo4j stats
	graphStats, err := egb.graphBuilder.GetStats(ctx)
	if err != nil {
		return nil, err
	}

	// Get Qdrant stats
	collectionInfo, err := egb.vectorStore.GetCollectionInfo(ctx)
	if err != nil {
		return nil, err
	}

	return &EnhancedGraphStats{
		PaperCount:      graphStats.PaperCount,
		ConceptCount:    graphStats.ConceptCount,
		CitationCount:   graphStats.CitationCount,
		SimilarityCount: graphStats.SimilarityCount,
		VectorCount:     int(*collectionInfo.PointsCount),
		LastUpdated:     graphStats.LastUpdated,
	}, nil
}

// determineChunkType determines the type of chunk based on its position
func determineChunkType(index, total int) string {
	if index == 0 {
		return "abstract"
	} else if index < total/3 {
		return "introduction"
	} else if index < 2*total/3 {
		return "methodology"
	} else {
		return "results"
	}
}
