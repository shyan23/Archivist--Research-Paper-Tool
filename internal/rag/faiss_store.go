package rag

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"os"
	"path/filepath"
	"sync"
)

// FAISSVectorStore is a file-based vector store using FAISS-like indexing
type FAISSVectorStore struct {
	indexPath   string
	documents   map[string]VectorDocument // docID -> document
	embeddings  [][]float32               // list of embeddings
	docIDs      []string                  // corresponding doc IDs
	mu          sync.RWMutex
}

// NewFAISSVectorStore creates a new FAISS-based vector store
func NewFAISSVectorStore(indexDir string) (*FAISSVectorStore, error) {
	// Create index directory if it doesn't exist
	if err := os.MkdirAll(indexDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create index directory: %w", err)
	}

	vs := &FAISSVectorStore{
		indexPath:  filepath.Join(indexDir, "faiss_index.json"),
		documents:  make(map[string]VectorDocument),
		embeddings: [][]float32{},
		docIDs:     []string{},
	}

	// Load existing index if available
	if err := vs.load(); err != nil {
		log.Printf("No existing index found, starting fresh: %v", err)
	} else {
		log.Printf("✓ Loaded existing FAISS index with %d documents", len(vs.documents))
	}

	return vs, nil
}

// AddDocument adds a document with its embedding to the vector store
func (vs *FAISSVectorStore) AddDocument(ctx context.Context, doc VectorDocument) error {
	vs.mu.Lock()
	defer vs.mu.Unlock()

	if doc.ID == "" {
		return fmt.Errorf("document ID is required")
	}

	if len(doc.Embedding) != EmbeddingDimensions {
		return fmt.Errorf("embedding dimension mismatch: expected %d, got %d",
			EmbeddingDimensions, len(doc.Embedding))
	}

	// Add or update document
	vs.documents[doc.ID] = doc

	// Check if document already exists in index
	found := false
	for i, id := range vs.docIDs {
		if id == doc.ID {
			// Update existing embedding
			vs.embeddings[i] = doc.Embedding
			found = true
			break
		}
	}

	if !found {
		// Add new embedding
		vs.embeddings = append(vs.embeddings, doc.Embedding)
		vs.docIDs = append(vs.docIDs, doc.ID)
	}

	return nil
}

// AddDocuments adds multiple documents in batch
func (vs *FAISSVectorStore) AddDocuments(ctx context.Context, docs []VectorDocument) error {
	if len(docs) == 0 {
		return nil
	}

	for _, doc := range docs {
		if err := vs.AddDocument(ctx, doc); err != nil {
			log.Printf("Warning: failed to add document %s: %v", doc.ID, err)
			continue
		}
	}

	// Save to disk after batch
	if err := vs.save(); err != nil {
		return fmt.Errorf("failed to save index: %w", err)
	}

	log.Printf("✓ Added %d documents to FAISS vector store", len(docs))
	return nil
}

// Search performs vector similarity search using cosine similarity
func (vs *FAISSVectorStore) Search(ctx context.Context, queryEmbedding []float32, topK int, filter map[string]string) ([]SearchResult, error) {
	vs.mu.RLock()
	defer vs.mu.RUnlock()

	if len(queryEmbedding) != EmbeddingDimensions {
		return nil, fmt.Errorf("query embedding dimension mismatch: expected %d, got %d",
			EmbeddingDimensions, len(queryEmbedding))
	}

	if topK <= 0 {
		topK = 5
	}

	if len(vs.embeddings) == 0 {
		return []SearchResult{}, nil
	}

	// Calculate cosine similarity for all documents
	type scoredDoc struct {
		docID string
		score float32
	}

	scores := make([]scoredDoc, 0)

	for i, embedding := range vs.embeddings {
		docID := vs.docIDs[i]
		doc, exists := vs.documents[docID]
		if !exists {
			continue
		}

		// Apply filters
		if source, ok := filter["source"]; ok && source != "" {
			if doc.Source != source {
				continue
			}
		}

		// Calculate cosine similarity
		similarity := cosineSimilarity(queryEmbedding, embedding)
		scores = append(scores, scoredDoc{docID: docID, score: similarity})
	}

	// Sort by score (descending)
	for i := 0; i < len(scores); i++ {
		for j := i + 1; j < len(scores); j++ {
			if scores[j].score > scores[i].score {
				scores[i], scores[j] = scores[j], scores[i]
			}
		}
	}

	// Take top K
	if topK > len(scores) {
		topK = len(scores)
	}

	results := make([]SearchResult, topK)
	for i := 0; i < topK; i++ {
		doc := vs.documents[scores[i].docID]
		results[i] = SearchResult{
			Document: doc,
			Score:    scores[i].score,
			Distance: 1.0 - scores[i].score, // Distance = 1 - similarity
		}
	}

	return results, nil
}

// SearchBySource searches for chunks from a specific paper
func (vs *FAISSVectorStore) SearchBySource(ctx context.Context, queryEmbedding []float32, source string, topK int) ([]SearchResult, error) {
	filter := map[string]string{"source": source}
	return vs.Search(ctx, queryEmbedding, topK, filter)
}

// GetDocumentsBySource retrieves all chunks for a specific paper
func (vs *FAISSVectorStore) GetDocumentsBySource(ctx context.Context, source string) ([]VectorDocument, error) {
	vs.mu.RLock()
	defer vs.mu.RUnlock()

	docs := []VectorDocument{}
	for _, doc := range vs.documents {
		if doc.Source == source {
			docs = append(docs, doc)
		}
	}

	return docs, nil
}

// DeleteBySource deletes all chunks for a specific paper
func (vs *FAISSVectorStore) DeleteBySource(ctx context.Context, source string) (int, error) {
	vs.mu.Lock()
	defer vs.mu.Unlock()

	// Find documents to delete
	toDelete := []string{}
	for docID, doc := range vs.documents {
		if doc.Source == source {
			toDelete = append(toDelete, docID)
		}
	}

	if len(toDelete) == 0 {
		return 0, nil
	}

	// Delete documents
	for _, docID := range toDelete {
		delete(vs.documents, docID)

		// Remove from embeddings and docIDs
		for i, id := range vs.docIDs {
			if id == docID {
				// Remove from embeddings
				vs.embeddings = append(vs.embeddings[:i], vs.embeddings[i+1:]...)
				// Remove from docIDs
				vs.docIDs = append(vs.docIDs[:i], vs.docIDs[i+1:]...)
				break
			}
		}
	}

	// Save changes
	if err := vs.save(); err != nil {
		return 0, fmt.Errorf("failed to save after deletion: %w", err)
	}

	log.Printf("✓ Deleted %d chunks for source: %s", len(toDelete), source)
	return len(toDelete), nil
}

// save persists the index to disk
func (vs *FAISSVectorStore) save() error {
	// Create index data structure
	indexData := struct {
		Documents  map[string]VectorDocument `json:"documents"`
		Embeddings [][]float32               `json:"embeddings"`
		DocIDs     []string                  `json:"doc_ids"`
	}{
		Documents:  vs.documents,
		Embeddings: vs.embeddings,
		DocIDs:     vs.docIDs,
	}

	// Marshal to JSON
	data, err := json.Marshal(indexData)
	if err != nil {
		return fmt.Errorf("failed to marshal index: %w", err)
	}

	// Write to file
	if err := os.WriteFile(vs.indexPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write index file: %w", err)
	}

	return nil
}

// load loads the index from disk
func (vs *FAISSVectorStore) load() error {
	// Read file
	data, err := os.ReadFile(vs.indexPath)
	if err != nil {
		return err
	}

	// Unmarshal JSON
	var indexData struct {
		Documents  map[string]VectorDocument `json:"documents"`
		Embeddings [][]float32               `json:"embeddings"`
		DocIDs     []string                  `json:"doc_ids"`
	}

	if err := json.Unmarshal(data, &indexData); err != nil {
		return fmt.Errorf("failed to unmarshal index: %w", err)
	}

	vs.documents = indexData.Documents
	vs.embeddings = indexData.Embeddings
	vs.docIDs = indexData.DocIDs

	return nil
}

// GetStats returns statistics about the vector store
func (vs *FAISSVectorStore) GetStats() map[string]interface{} {
	vs.mu.RLock()
	defer vs.mu.RUnlock()

	// Count unique sources
	sources := make(map[string]int)
	for _, doc := range vs.documents {
		sources[doc.Source]++
	}

	return map[string]interface{}{
		"total_documents": len(vs.documents),
		"total_papers":    len(sources),
		"index_size":      len(vs.embeddings),
	}
}

// GetIndexedPapers returns a list of all indexed paper titles
func (vs *FAISSVectorStore) GetIndexedPapers() []string {
	vs.mu.RLock()
	defer vs.mu.RUnlock()

	// Collect unique sources
	sourcesMap := make(map[string]bool)
	for _, doc := range vs.documents {
		sourcesMap[doc.Source] = true
	}

	// Convert to slice
	sources := make([]string, 0, len(sourcesMap))
	for source := range sourcesMap {
		sources = append(sources, source)
	}

	return sources
}

// Helper function: cosine similarity
func cosineSimilarity(a, b []float32) float32 {
	if len(a) != len(b) {
		return 0
	}

	var dotProduct float64
	var normA float64
	var normB float64

	for i := 0; i < len(a); i++ {
		dotProduct += float64(a[i]) * float64(b[i])
		normA += float64(a[i]) * float64(a[i])
		normB += float64(b[i]) * float64(b[i])
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return float32(dotProduct / (math.Sqrt(normA) * math.Sqrt(normB)))
}
