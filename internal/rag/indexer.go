package rag

import (
	"context"
	"crypto/md5"
	"fmt"
	"log"
	"os"
)

// Indexer handles indexing of papers into the vector store
type Indexer struct {
	chunker     *Chunker
	embedClient *EmbeddingClient
	vectorStore VectorStoreInterface
}

// NewIndexer creates a new indexer
func NewIndexer(chunker *Chunker, embedClient *EmbeddingClient, vectorStore VectorStoreInterface) *Indexer {
	return &Indexer{
		chunker:     chunker,
		embedClient: embedClient,
		vectorStore: vectorStore,
	}
}

// IndexPaper indexes a paper by reading its LaTeX content and PDF
func (i *Indexer) IndexPaper(ctx context.Context, paperTitle, latexContent, pdfPath string) error {
	if paperTitle == "" {
		return fmt.Errorf("paper title is required")
	}

	log.Printf("  📇 Indexing paper: %s", paperTitle)

	// Chunk the LaTeX content
	log.Println("  ✂️  Chunking LaTeX content...")
	chunks, err := i.chunker.ChunkLaTeXContent(latexContent, paperTitle)
	if err != nil {
		return fmt.Errorf("failed to chunk content: %w", err)
	}

	if len(chunks) == 0 {
		return fmt.Errorf("no chunks generated from content")
	}

	log.Printf("  ✓ Created %d chunks", len(chunks))

	// Extract text for embedding
	texts := make([]string, len(chunks))
	for idx, chunk := range chunks {
		texts[idx] = chunk.Text
	}

	// Generate embeddings in batch
	log.Println("  🧮 Generating embeddings...")
	embeddings, err := i.embedClient.GenerateBatchEmbeddings(ctx, texts)
	if err != nil {
		return fmt.Errorf("failed to generate embeddings: %w", err)
	}

	log.Printf("  ✓ Generated %d embeddings", len(embeddings))

	// Create vector documents
	docs := make([]VectorDocument, len(chunks))
	for idx, chunk := range chunks {
		docID := generateDocID(paperTitle, chunk.ChunkIndex)

		docs[idx] = VectorDocument{
			ID:         docID,
			ChunkText:  chunk.Text,
			Embedding:  embeddings[idx],
			Source:     paperTitle,
			Section:    chunk.Section,
			ChunkIndex: chunk.ChunkIndex,
			Metadata: map[string]string{
				"source":      paperTitle,
				"section":     chunk.Section,
				"chunk_index": fmt.Sprintf("%d", chunk.ChunkIndex),
			},
		}

		// Add PDF path if available
		if pdfPath != "" {
			docs[idx].Metadata["pdf_path"] = pdfPath
		}
	}

	// Store in vector database
	log.Println("  💾 Storing vectors in database...")
	if err := i.vectorStore.AddDocuments(ctx, docs); err != nil {
		return fmt.Errorf("failed to store vectors: %w", err)
	}

	log.Printf("  ✅ Successfully indexed paper: %s (%d chunks)", paperTitle, len(chunks))

	return nil
}

// IndexPaperFromPDF indexes a paper from PDF and its generated LaTeX
func (i *Indexer) IndexPaperFromPDF(ctx context.Context, pdfPath, latexContent string) error {
	// Extract title from PDF path
	paperTitle := extractTitleFromPath(pdfPath)

	return i.IndexPaper(ctx, paperTitle, latexContent, pdfPath)
}

// ReindexPaper removes old indices and creates new ones
func (i *Indexer) ReindexPaper(ctx context.Context, paperTitle, latexContent, pdfPath string) error {
	log.Printf("  🔄 Reindexing paper: %s", paperTitle)

	// Delete existing chunks
	deleted, err := i.vectorStore.DeleteBySource(ctx, paperTitle)
	if err != nil {
		log.Printf("  ⚠️  Warning: Failed to delete old chunks: %v", err)
	} else if deleted > 0 {
		log.Printf("  🗑️  Deleted %d old chunks", deleted)
	}

	// Index the paper
	return i.IndexPaper(ctx, paperTitle, latexContent, pdfPath)
}

// CheckIfIndexed checks if a paper is already indexed
func (i *Indexer) CheckIfIndexed(ctx context.Context, paperTitle string) (bool, int, error) {
	docs, err := i.vectorStore.GetDocumentsBySource(ctx, paperTitle)
	if err != nil {
		return false, 0, err
	}

	return len(docs) > 0, len(docs), nil
}

// GetIndexedPapers returns a list of all indexed papers
func (i *Indexer) GetIndexedPapers(ctx context.Context) ([]string, error) {
	// This is a simplified version - in production, you'd maintain a separate index
	// For now, we'll use a dummy implementation
	// TODO: Implement proper paper listing from Redis
	return []string{}, nil
}

// Helper functions

func generateDocID(paperTitle string, chunkIndex int) string {
	// Create a unique document ID
	hash := md5.Sum([]byte(paperTitle))
	return fmt.Sprintf("%x_chunk_%d", hash, chunkIndex)
}

func extractTitleFromPath(pdfPath string) string {
	// Extract filename without extension
	filename := pdfPath
	if idx := len(filename) - 1; idx >= 0 {
		for i := idx; i >= 0; i-- {
			if filename[i] == '/' || filename[i] == '\\' {
				filename = filename[i+1:]
				break
			}
		}
	}

	// Remove .pdf extension
	if len(filename) > 4 && filename[len(filename)-4:] == ".pdf" {
		filename = filename[:len(filename)-4]
	}

	return filename
}

// ReadPDFContent reads PDF content (placeholder - actual implementation would use PDF parser)
func ReadPDFContent(pdfPath string) (string, error) {
	// Check if file exists
	if _, err := os.Stat(pdfPath); os.IsNotExist(err) {
		return "", fmt.Errorf("PDF file not found: %s", pdfPath)
	}

	// In actual implementation, use a PDF parser
	// For now, return empty string as LaTeX content will be used
	return "", nil
}
