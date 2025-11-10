package rag

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"unsafe"

	"github.com/redis/go-redis/v9"
)

const (
	// VectorIndexName is the Redis search index name for vectors
	VectorIndexName = "archivist:vectors:idx"
	// VectorKeyPrefix is the prefix for vector document keys
	VectorKeyPrefix = "archivist:vectors:doc:"
)

// VectorDocument represents a document stored in the vector database
type VectorDocument struct {
	ID          string            `json:"id"`
	ChunkText   string            `json:"chunk_text"`
	Embedding   []float32         `json:"embedding"`
	Source      string            `json:"source"`       // Paper title
	Section     string            `json:"section"`      // Section name
	ChunkIndex  int               `json:"chunk_index"`
	Metadata    map[string]string `json:"metadata"`
}

// SearchResult represents a search result with similarity score
type SearchResult struct {
	Document   VectorDocument `json:"document"`
	Score      float32        `json:"score"`      // Cosine similarity score
	Distance   float32        `json:"distance"`   // Vector distance
}

// VectorStore handles vector storage and retrieval using Redis Stack
type VectorStore struct {
	client     *redis.Client
	indexName  string
	keyPrefix  string
}

// NewVectorStore creates a new vector store
func NewVectorStore(client *redis.Client) (*VectorStore, error) {
	vs := &VectorStore{
		client:    client,
		indexName: VectorIndexName,
		keyPrefix: VectorKeyPrefix,
	}

	// Create vector index if it doesn't exist
	if err := vs.createIndexIfNotExists(context.Background()); err != nil {
		return nil, err
	}

	return vs, nil
}

// createIndexIfNotExists creates the Redis search index for vector similarity
func (vs *VectorStore) createIndexIfNotExists(ctx context.Context) error {
	// Check if index exists
	exists, err := vs.indexExists(ctx)
	if err != nil {
		return err
	}

	if exists {
		log.Printf("✓ Vector index '%s' already exists", vs.indexName)
		return nil
	}

	// Create index using FT.CREATE
	// Note: This requires Redis Stack with RediSearch module
	cmd := []interface{}{
		"FT.CREATE", vs.indexName,
		"ON", "JSON",
		"PREFIX", "1", vs.keyPrefix,
		"SCHEMA",
		"$.chunk_text", "AS", "chunk_text", "TEXT",
		"$.source", "AS", "source", "TAG",
		"$.section", "AS", "section", "TEXT",
		"$.chunk_index", "AS", "chunk_index", "NUMERIC",
		"$.embedding", "AS", "embedding", "VECTOR", "FLAT", "6",
		"TYPE", "FLOAT32",
		"DIM", fmt.Sprintf("%d", EmbeddingDimensions),
		"DISTANCE_METRIC", "COSINE",
	}

	result := vs.client.Do(ctx, cmd...)
	if result.Err() != nil {
		return fmt.Errorf("failed to create vector index: %w", result.Err())
	}

	log.Printf("✓ Created vector index '%s'", vs.indexName)
	return nil
}

// indexExists checks if the search index exists
func (vs *VectorStore) indexExists(ctx context.Context) (bool, error) {
	result := vs.client.Do(ctx, "FT._LIST")
	if result.Err() != nil {
		return false, result.Err()
	}

	indices, ok := result.Val().([]interface{})
	if !ok {
		return false, nil
	}

	for _, idx := range indices {
		if idxName, ok := idx.(string); ok && idxName == vs.indexName {
			return true, nil
		}
	}

	return false, nil
}

// AddDocument adds a document with its embedding to the vector store
func (vs *VectorStore) AddDocument(ctx context.Context, doc VectorDocument) error {
	if doc.ID == "" {
		return fmt.Errorf("document ID is required")
	}

	if len(doc.Embedding) != EmbeddingDimensions {
		return fmt.Errorf("embedding dimension mismatch: expected %d, got %d",
			EmbeddingDimensions, len(doc.Embedding))
	}

	key := vs.keyPrefix + doc.ID

	// Convert document to JSON
	jsonData, err := json.Marshal(doc)
	if err != nil {
		return fmt.Errorf("failed to marshal document: %w", err)
	}

	// Store as JSON document
	err = vs.client.Do(ctx, "JSON.SET", key, "$", string(jsonData)).Err()
	if err != nil {
		return fmt.Errorf("failed to store document: %w", err)
	}

	return nil
}

// AddDocuments adds multiple documents in batch
func (vs *VectorStore) AddDocuments(ctx context.Context, docs []VectorDocument) error {
	if len(docs) == 0 {
		return nil
	}

	pipe := vs.client.Pipeline()

	for _, doc := range docs {
		if doc.ID == "" {
			continue
		}

		key := vs.keyPrefix + doc.ID
		jsonData, err := json.Marshal(doc)
		if err != nil {
			log.Printf("Warning: failed to marshal document %s: %v", doc.ID, err)
			continue
		}

		pipe.Do(ctx, "JSON.SET", key, "$", string(jsonData))
	}

	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to add documents in batch: %w", err)
	}

	log.Printf("✓ Added %d documents to vector store", len(docs))
	return nil
}

// Search performs vector similarity search
func (vs *VectorStore) Search(ctx context.Context, queryEmbedding []float32, topK int, filter map[string]string) ([]SearchResult, error) {
	if len(queryEmbedding) != EmbeddingDimensions {
		return nil, fmt.Errorf("query embedding dimension mismatch: expected %d, got %d",
			EmbeddingDimensions, len(queryEmbedding))
	}

	if topK <= 0 {
		topK = 5
	}

	// Build query
	query := "*"
	if source, ok := filter["source"]; ok && source != "" {
		query = fmt.Sprintf("@source:{%s}", escapeRedisTag(source))
	}

	// Convert embedding to bytes for Redis
	embBytes := float32SliceToBytes(queryEmbedding)

	// Build FT.SEARCH command with KNN
	cmd := []interface{}{
		"FT.SEARCH", vs.indexName,
		query,
		"RETURN", "4", "chunk_text", "source", "section", "chunk_index",
		"SORTBY", "__embedding_score",
		"DIALECT", "2",
		"LIMIT", "0", fmt.Sprintf("%d", topK),
		"PARAMS", "2", "vec", embBytes,
	}

	result := vs.client.Do(ctx, cmd...)
	if result.Err() != nil {
		return nil, fmt.Errorf("vector search failed: %w", result.Err())
	}

	// Parse results
	results, err := vs.parseSearchResults(ctx, result.Val())
	if err != nil {
		return nil, fmt.Errorf("failed to parse search results: %w", err)
	}

	return results, nil
}

// SearchBySource searches for chunks from a specific paper
func (vs *VectorStore) SearchBySource(ctx context.Context, queryEmbedding []float32, source string, topK int) ([]SearchResult, error) {
	filter := map[string]string{"source": source}
	return vs.Search(ctx, queryEmbedding, topK, filter)
}

// GetDocumentsBySource retrieves all chunks for a specific paper
func (vs *VectorStore) GetDocumentsBySource(ctx context.Context, source string) ([]VectorDocument, error) {
	query := fmt.Sprintf("@source:{%s}", escapeRedisTag(source))

	cmd := []interface{}{
		"FT.SEARCH", vs.indexName,
		query,
		"LIMIT", "0", "1000", // Max 1000 chunks per paper
	}

	result := vs.client.Do(ctx, cmd...)
	if result.Err() != nil {
		return nil, fmt.Errorf("failed to get documents: %w", result.Err())
	}

	docs, err := vs.parseDocuments(result.Val())
	if err != nil {
		return nil, err
	}

	return docs, nil
}

// DeleteBySource deletes all chunks for a specific paper
func (vs *VectorStore) DeleteBySource(ctx context.Context, source string) (int, error) {
	docs, err := vs.GetDocumentsBySource(ctx, source)
	if err != nil {
		return 0, err
	}

	if len(docs) == 0 {
		return 0, nil
	}

	pipe := vs.client.Pipeline()
	for _, doc := range docs {
		key := vs.keyPrefix + doc.ID
		pipe.Del(ctx, key)
	}

	_, err = pipe.Exec(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to delete documents: %w", err)
	}

	log.Printf("✓ Deleted %d chunks for source: %s", len(docs), source)
	return len(docs), nil
}

// parseSearchResults parses Redis search results into SearchResult structs
func (vs *VectorStore) parseSearchResults(ctx context.Context, val interface{}) ([]SearchResult, error) {
	// Redis FT.SEARCH returns: [total_count, key1, [field1, value1, ...], key2, [...], ...]
	arr, ok := val.([]interface{})
	if !ok || len(arr) < 1 {
		return []SearchResult{}, nil
	}

	var results []SearchResult

	// Skip total count, process documents
	for i := 1; i < len(arr); i += 2 {
		if i+1 >= len(arr) {
			break
		}

		docKey, ok := arr[i].(string)
		if !ok {
			continue
		}

		// Get full document from Redis
		doc, err := vs.getDocumentByKey(ctx, docKey)
		if err != nil {
			log.Printf("Warning: failed to get document %s: %v", docKey, err)
			continue
		}

		results = append(results, SearchResult{
			Document: doc,
			Score:    0.0, // Will be calculated if needed
		})
	}

	return results, nil
}

// getDocumentByKey retrieves a document by its Redis key
func (vs *VectorStore) getDocumentByKey(ctx context.Context, key string) (VectorDocument, error) {
	result := vs.client.Do(ctx, "JSON.GET", key)
	if result.Err() != nil {
		return VectorDocument{}, result.Err()
	}

	jsonStr, ok := result.Val().(string)
	if !ok {
		return VectorDocument{}, fmt.Errorf("invalid document format")
	}

	var doc VectorDocument
	if err := json.Unmarshal([]byte(jsonStr), &doc); err != nil {
		return VectorDocument{}, err
	}

	return doc, nil
}

// parseDocuments parses document list from search results
func (vs *VectorStore) parseDocuments(val interface{}) ([]VectorDocument, error) {
	results, err := vs.parseSearchResults(context.Background(), val)
	if err != nil {
		return nil, err
	}

	docs := make([]VectorDocument, len(results))
	for i, result := range results {
		docs[i] = result.Document
	}

	return docs, nil
}

// Helper functions

func escapeRedisTag(tag string) string {
	// Escape special characters for Redis TAG fields
	replacer := []string{
		",", "\\,",
		".", "\\.",
		"<", "\\<",
		">", "\\>",
		"{", "\\{",
		"}", "\\}",
		"[", "\\[",
		"]", "\\]",
		"\"", "\\\"",
		"'", "\\'",
		":", "\\:",
		";", "\\;",
		"!", "\\!",
		"@", "\\@",
		"#", "\\#",
		"$", "\\$",
		"%", "\\%",
		"^", "\\^",
		"&", "\\&",
		"*", "\\*",
		"(", "\\(",
		")", "\\)",
		"-", "\\-",
		"+", "\\+",
		"=", "\\=",
		"~", "\\~",
	}
	r := strings.NewReplacer(replacer...)
	return r.Replace(tag)
}

func float32SliceToBytes(data []float32) []byte {
	bytes := make([]byte, len(data)*4)
	for i, v := range data {
		bits := *(*uint32)(unsafe.Pointer(&v))
		bytes[i*4] = byte(bits)
		bytes[i*4+1] = byte(bits >> 8)
		bytes[i*4+2] = byte(bits >> 16)
		bytes[i*4+3] = byte(bits >> 24)
	}
	return bytes
}
