package worker

import (
	"archivist/internal/app"
	"archivist/internal/rag"
	"context"
	"log"
	"path/filepath"
)

// IndexPaperAfterProcessing indexes a paper after successful processing
func IndexPaperAfterProcessing(ctx context.Context, config *app.Config, paperTitle, latexContent, pdfPath string) error {
	// Initialize FAISS vector store
	indexDir := filepath.Join(".metadata", "vector_index")
	vectorStore, err := rag.NewFAISSVectorStore(indexDir)
	if err != nil {
		log.Printf("  ⚠️  Warning: Failed to create FAISS vector store, skipping indexing: %v", err)
		return nil // Don't fail the whole process if indexing fails
	}

	// Initialize embedding client
	embedClient, err := rag.NewEmbeddingClient(config.Gemini.APIKey)
	if err != nil {
		log.Printf("  ⚠️  Warning: Failed to create embedding client, skipping indexing: %v", err)
		return nil
	}
	defer embedClient.Close()

	// Create indexer
	chunker := rag.NewChunker(rag.DefaultChunkSize, rag.DefaultChunkOverlap)
	indexer := rag.NewIndexer(chunker, embedClient, vectorStore)

	// Index the paper
	log.Printf("  📇 Indexing paper for chat feature...")
	if err := indexer.IndexPaper(ctx, paperTitle, latexContent, pdfPath); err != nil {
		log.Printf("  ⚠️  Warning: Failed to index paper: %v", err)
		return nil // Don't fail the whole process
	}

	log.Printf("  ✅ Paper indexed successfully for chat")
	return nil
}
