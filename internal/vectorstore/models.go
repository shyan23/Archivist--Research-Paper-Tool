package vectorstore

import (
	qdrant "github.com/qdrant/go-client/qdrant"
)

// Point represents a vector point to be stored in Qdrant
type Point struct {
	ID      string                       // UUID
	Vector  []float32                    // Embedding vector
	Payload map[string]*qdrant.Value     // Metadata
}

// SearchQuery represents a search request
type SearchQuery struct {
	Vector         []float32          // Query vector
	Limit          uint64             // Number of results
	ScoreThreshold float32            // Minimum similarity score
	Filter         *qdrant.Filter     // Optional filters
}

// SearchResult represents a search hit
type SearchResult struct {
	ID      string                       // Point UUID
	Score   float64                      // Similarity score
	Payload map[string]*qdrant.Value     // Metadata
}

// PaperChunk represents a chunk of paper content with embeddings
type PaperChunk struct {
	ID           string   `json:"id"`
	PaperTitle   string   `json:"paper_title"`
	PaperID      string   `json:"paper_id"`
	ChunkIndex   int      `json:"chunk_index"`
	ChunkType    string   `json:"chunk_type"` // "abstract", "methodology", "results", "full"
	Content      string   `json:"content"`
	Embedding    []float32 `json:"-"` // Not serialized

	// Metadata for filtering
	Year          int      `json:"year,omitempty"`
	Authors       []string `json:"authors,omitempty"`
	Methodologies []string `json:"methodologies,omitempty"`
	Datasets      []string `json:"datasets,omitempty"`
	Metrics       []string `json:"metrics,omitempty"`
}

// ToQdrantPoint converts PaperChunk to Qdrant Point
func (pc *PaperChunk) ToQdrantPoint() *Point {
	payload := make(map[string]*qdrant.Value)

	payload["paper_title"] = &qdrant.Value{Kind: &qdrant.Value_StringValue{StringValue: pc.PaperTitle}}
	payload["paper_id"] = &qdrant.Value{Kind: &qdrant.Value_StringValue{StringValue: pc.PaperID}}
	payload["chunk_index"] = &qdrant.Value{Kind: &qdrant.Value_IntegerValue{IntegerValue: int64(pc.ChunkIndex)}}
	payload["chunk_type"] = &qdrant.Value{Kind: &qdrant.Value_StringValue{StringValue: pc.ChunkType}}
	payload["content"] = &qdrant.Value{Kind: &qdrant.Value_StringValue{StringValue: pc.Content}}

	if pc.Year > 0 {
		payload["year"] = &qdrant.Value{Kind: &qdrant.Value_IntegerValue{IntegerValue: int64(pc.Year)}}
	}

	if len(pc.Authors) > 0 {
		authors := make([]*qdrant.Value, len(pc.Authors))
		for i, author := range pc.Authors {
			authors[i] = &qdrant.Value{Kind: &qdrant.Value_StringValue{StringValue: author}}
		}
		payload["authors"] = &qdrant.Value{Kind: &qdrant.Value_ListValue{ListValue: &qdrant.ListValue{Values: authors}}}
	}

	if len(pc.Methodologies) > 0 {
		methodologies := make([]*qdrant.Value, len(pc.Methodologies))
		for i, method := range pc.Methodologies {
			methodologies[i] = &qdrant.Value{Kind: &qdrant.Value_StringValue{StringValue: method}}
		}
		payload["methodologies"] = &qdrant.Value{Kind: &qdrant.Value_ListValue{ListValue: &qdrant.ListValue{Values: methodologies}}}
	}

	if len(pc.Datasets) > 0 {
		datasets := make([]*qdrant.Value, len(pc.Datasets))
		for i, dataset := range pc.Datasets {
			datasets[i] = &qdrant.Value{Kind: &qdrant.Value_StringValue{StringValue: dataset}}
		}
		payload["datasets"] = &qdrant.Value{Kind: &qdrant.Value_ListValue{ListValue: &qdrant.ListValue{Values: datasets}}}
	}

	return &Point{
		ID:      pc.ID,
		Vector:  pc.Embedding,
		Payload: payload,
	}
}

// HybridSearchQuery combines vector, graph, and keyword search
type HybridSearchQuery struct {
	Query          string             // Natural language query
	QueryVector    []float32          // Query embedding
	VectorWeight   float64            // Weight for vector similarity (0-1)
	GraphWeight    float64            // Weight for graph-based score (0-1)
	KeywordWeight  float64            // Weight for keyword matching (0-1)
	TopK           int                // Number of results
	Filters        map[string]interface{} // Metadata filters
	TraversalDepth int                // Graph traversal depth
}

// HybridSearchResult combines results from multiple search strategies
type HybridSearchResult struct {
	PaperTitle    string             `json:"paper_title"`
	ChunkContent  string             `json:"chunk_content"`
	VectorScore   float64            `json:"vector_score"`
	GraphScore    float64            `json:"graph_score"`
	KeywordScore  float64            `json:"keyword_score"`
	HybridScore   float64            `json:"hybrid_score"`
	Rank          int                `json:"rank"`
	Metadata      map[string]interface{} `json:"metadata"`
}
