package rag

import "context"

// VectorStoreInterface defines the interface for vector storage implementations
type VectorStoreInterface interface {
	// AddDocument adds a single document with its embedding
	AddDocument(ctx context.Context, doc VectorDocument) error

	// AddDocuments adds multiple documents in batch
	AddDocuments(ctx context.Context, docs []VectorDocument) error

	// Search performs vector similarity search
	Search(ctx context.Context, queryEmbedding []float32, topK int, filter map[string]string) ([]SearchResult, error)

	// SearchBySource searches for chunks from a specific paper
	SearchBySource(ctx context.Context, queryEmbedding []float32, source string, topK int) ([]SearchResult, error)

	// GetDocumentsBySource retrieves all chunks for a specific paper
	GetDocumentsBySource(ctx context.Context, source string) ([]VectorDocument, error)

	// DeleteBySource deletes all chunks for a specific paper
	DeleteBySource(ctx context.Context, source string) (int, error)
}
