package graph

import "time"

// PaperNode represents a paper in the knowledge graph
type PaperNode struct {
	Title          string    `json:"title"`
	PDFPath        string    `json:"pdf_path"`
	ProcessedAt    time.Time `json:"processed_at"`
	EmbeddingID    string    `json:"embedding_id"` // Link to FAISS vector
	Methodologies  []string  `json:"methodologies"`
	Datasets       []string  `json:"datasets"`
	Metrics        []string  `json:"metrics"`
	Year           int       `json:"year,omitempty"`
	Authors        []string  `json:"authors,omitempty"`
	Abstract       string    `json:"abstract,omitempty"`
}

// ConceptNode represents a concept in the knowledge graph
type ConceptNode struct {
	Name      string `json:"name"`
	Category  string `json:"category"` // "methodology", "architecture", "dataset", "metric"
	Frequency int    `json:"frequency"`
}

// CitationRelationship represents a citation between papers
type CitationRelationship struct {
	SourcePaper string `json:"source_paper"`
	TargetPaper string `json:"target_paper"`
	Importance  string `json:"importance"` // "high", "medium", "low"
	Context     string `json:"context"`    // Surrounding text
}

// ConceptRelationship represents a paper using a concept
type ConceptRelationship struct {
	PaperTitle string `json:"paper_title"`
	Concept    string `json:"concept"`
	Section    string `json:"section"`
}

// SimilarityRelationship represents semantic similarity between papers
type SimilarityRelationship struct {
	Paper1 string  `json:"paper1"`
	Paper2 string  `json:"paper2"`
	Score  float64 `json:"score"`
	Basis  string  `json:"basis"` // "semantic", "methodological", "dataset"
}

// GraphUpdateJob represents an async graph update task
type GraphUpdateJob struct {
	PaperTitle   string
	LatexContent string
	Citations    *CitationData
	PDFPath      string
	Priority     int
}

// CitationData holds extracted citation information
type CitationData struct {
	References      []Reference
	InTextCitations []InTextCitation
	ManualOverrides map[string]string
}

// Reference represents a formal reference from bibliography
type Reference struct {
	Index   int      `json:"index"` // [1], [2], etc.
	Authors []string `json:"authors"`
	Title   string   `json:"title"`
	Year    int      `json:"year"`
	Venue   string   `json:"venue"`
	RawText string   `json:"raw_text"`
}

// InTextCitation represents a citation found in the main text
type InTextCitation struct {
	ReferenceIndex int    `json:"reference_index"`
	Context        string `json:"context"`    // Surrounding text
	Importance     string `json:"importance"` // "high", "medium", "low"
}

// SearchResult represents a result from hybrid search
type SearchResult struct {
	PaperTitle   string
	VectorScore  float64
	GraphScore   float64
	KeywordScore float64
	HybridScore  float64
	IsCited      bool
	CitationCount int
	Metadata     map[string]interface{}
}

// GraphStats represents statistics about the knowledge graph
type GraphStats struct {
	PaperCount      int
	ConceptCount    int
	CitationCount   int
	SimilarityCount int
	LastUpdated     time.Time
}
