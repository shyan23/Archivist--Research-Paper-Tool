package graph

import "time"

// ============================================================================
// ENHANCED KNOWLEDGE GRAPH MODELS
// Based on plans/graph_ideas - Heterogeneous multi-layer graph
// ============================================================================

// ============================================================================
// NODE TYPES
// ============================================================================

// PaperNode represents a research paper (enhanced with more attributes)
type PaperNodeEnhanced struct {
	// Core identification
	Title    string `json:"title"`
	DOI      string `json:"doi,omitempty"`
	ArxivID  string `json:"arxiv_id,omitempty"`
	PDFPath  string `json:"pdf_path"`

	// Temporal
	Year           int       `json:"year"`
	PublishedDate  time.Time `json:"published_date,omitempty"`
	ProcessedAt    time.Time `json:"processed_at"`

	// Content
	Abstract       string   `json:"abstract"`
	Keywords       []string `json:"keywords,omitempty"`

	// Metadata
	Authors        []string `json:"authors"`
	Venue          string   `json:"venue,omitempty"` // Conference/Journal
	Methodologies  []string `json:"methodologies"`
	Datasets       []string `json:"datasets"`
	Metrics        []string `json:"metrics"`

	// Embedding
	EmbeddingID    string `json:"embedding_id"` // Link to Qdrant

	// Analytics (computed)
	CitationCount  int     `json:"citation_count,omitempty"`
	PageRank       float64 `json:"pagerank,omitempty"`
	HIndex         int     `json:"h_index,omitempty"`
}

// AuthorNode represents an individual researcher
type AuthorNode struct {
	Name            string   `json:"name"`
	ORCID           string   `json:"orcid,omitempty"`
	Email           string   `json:"email,omitempty"`
	Affiliation     string   `json:"affiliation,omitempty"`
	Field           string   `json:"field,omitempty"`
	HIndex          int      `json:"h_index,omitempty"`
	TotalCitations  int      `json:"total_citations,omitempty"`
	ActiveSince     int      `json:"active_since,omitempty"` // Year
	PaperCount      int      `json:"paper_count,omitempty"`

	// Analytics
	Centrality      float64  `json:"centrality,omitempty"`
	Influence       float64  `json:"influence,omitempty"`
}

// InstitutionNode represents an organization or university
type InstitutionNode struct {
	Name           string   `json:"name"`
	Country        string   `json:"country,omitempty"`
	City           string   `json:"city,omitempty"`
	Type           string   `json:"type,omitempty"` // university, company, research_lab
	ResearchDomain string   `json:"research_domain,omitempty"`
	Website        string   `json:"website,omitempty"`

	// Analytics
	PaperCount     int      `json:"paper_count,omitempty"`
	TotalCitations int      `json:"total_citations,omitempty"`
	Impact         float64  `json:"impact,omitempty"`
}

// ConceptNode represents a key scientific idea or topic (enhanced)
type ConceptNodeEnhanced struct {
	Name           string    `json:"name"`
	Category       string    `json:"category"` // methodology, architecture, task, etc.
	Description    string    `json:"description,omitempty"`
	EmbeddingID    string    `json:"embedding_id,omitempty"`
	Frequency      int       `json:"frequency"` // How many papers mention it
	FirstSeen      int       `json:"first_seen,omitempty"` // Year

	// Analytics
	TrendScore     float64   `json:"trend_score,omitempty"`
	GrowthRate     float64   `json:"growth_rate,omitempty"`
}

// MethodNode represents specific algorithms or techniques
type MethodNode struct {
	Name           string   `json:"name"`
	Type           string   `json:"type"` // algorithm, architecture, technique
	Description    string   `json:"description,omitempty"`
	Complexity     string   `json:"complexity,omitempty"` // O(n), O(n^2), etc.
	IntroducedBy   string   `json:"introduced_by,omitempty"` // Paper title
	IntroducedYear int      `json:"introduced_year,omitempty"`

	// Usage
	UsageCount     int      `json:"usage_count,omitempty"`
	Variants       []string `json:"variants,omitempty"`
}

// VenueNode represents a conference or journal
type VenueNode struct {
	Name           string  `json:"name"`
	ShortName      string  `json:"short_name"` // CVPR, NeurIPS, etc.
	Type           string  `json:"type"` // conference, journal
	Rank           string  `json:"rank,omitempty"` // A*, A, B, C
	ImpactFactor   float64 `json:"impact_factor,omitempty"`
	AcceptanceRate float64 `json:"acceptance_rate,omitempty"`

	// Analytics
	PaperCount     int     `json:"paper_count,omitempty"`
	CitationCount  int     `json:"citation_count,omitempty"`
}

// DatasetNode represents benchmark datasets (optional)
type DatasetNode struct {
	Name           string   `json:"name"`
	Type           string   `json:"type"` // image, text, audio, etc.
	Size           string   `json:"size,omitempty"`
	Description    string   `json:"description,omitempty"`
	IntroducedYear int      `json:"introduced_year,omitempty"`
	URL            string   `json:"url,omitempty"`

	// Usage
	UsageCount     int      `json:"usage_count,omitempty"`
	BenchmarkFor   []string `json:"benchmark_for,omitempty"` // Task types
}

// ============================================================================
// RELATIONSHIP TYPES
// ============================================================================

// CitationRelationshipEnhanced - Paper → Paper (enhanced with context)
type CitationRelationshipEnhanced struct {
	SourcePaper    string    `json:"source_paper"`
	TargetPaper    string    `json:"target_paper"`
	Importance     string    `json:"importance"` // high, medium, low
	Context        string    `json:"context"` // Surrounding text
	CitationType   string    `json:"citation_type,omitempty"` // background, comparison, methodology
	SectionType    string    `json:"section_type,omitempty"` // intro, methods, results
	Timestamp      time.Time `json:"timestamp,omitempty"`
	Weight         float64   `json:"weight,omitempty"` // 1.0 = standard
}

// AuthorshipRelationship - Paper → Author
type AuthorshipRelationship struct {
	PaperTitle     string `json:"paper_title"`
	AuthorName     string `json:"author_name"`
	Position       int    `json:"position"` // First author = 1
	IsCorresponding bool  `json:"is_corresponding,omitempty"`
}

// AffiliationRelationship - Author → Institution
type AffiliationRelationship struct {
	AuthorName     string    `json:"author_name"`
	InstitutionName string   `json:"institution_name"`
	Role           string    `json:"role,omitempty"` // professor, postdoc, student
	StartYear      int       `json:"start_year,omitempty"`
	EndYear        int       `json:"end_year,omitempty"`
}

// UsesMethodRelationship - Paper → Method
type UsesMethodRelationship struct {
	PaperTitle     string `json:"paper_title"`
	MethodName     string `json:"method_name"`
	IsMainMethod   bool   `json:"is_main_method"` // Primary vs auxiliary
	Description    string `json:"description,omitempty"`
}

// MentionsConceptRelationship - Paper → Concept
type MentionsConceptRelationship struct {
	PaperTitle     string `json:"paper_title"`
	ConceptName    string `json:"concept_name"`
	Frequency      int    `json:"frequency"` // How many times mentioned
	IsCoreTheme    bool   `json:"is_core_theme"` // Main topic vs passing mention
}

// PublishedInRelationship - Paper → Venue
type PublishedInRelationship struct {
	PaperTitle     string    `json:"paper_title"`
	VenueName      string    `json:"venue_name"`
	Year           int       `json:"year"`
	Pages          string    `json:"pages,omitempty"`
	BestPaperAward bool      `json:"best_paper_award,omitempty"`
}

// CoAuthorshipRelationship - Author ↔ Author
type CoAuthorshipRelationship struct {
	Author1        string `json:"author1"`
	Author2        string `json:"author2"`
	JointPapers    int    `json:"joint_papers"` // Number of papers together
	FirstColab     int    `json:"first_colab,omitempty"` // Year
	LastColab      int    `json:"last_colab,omitempty"` // Year
	Weight         float64 `json:"weight"` // Collaboration strength
}

// ExtendsRelationship - Paper → Paper (conceptual lineage)
type ExtendsRelationship struct {
	SourcePaper    string `json:"source_paper"`
	TargetPaper    string `json:"target_paper"` // Paper being extended
	ExtensionType  string `json:"extension_type"` // improves, generalizes, specializes
	Description    string `json:"description,omitempty"`
}

// SimilarityRelationshipEnhanced - Paper ↔ Paper (enhanced with basis)
type SimilarityRelationshipEnhanced struct {
	Paper1         string  `json:"paper1"`
	Paper2         string  `json:"paper2"`
	Score          float64 `json:"score"` // Cosine similarity (0-1)
	Basis          string  `json:"basis"` // semantic, methodological, dataset, results
	SharedConcepts []string `json:"shared_concepts,omitempty"`
	SharedMethods  []string `json:"shared_methods,omitempty"`
}

// UsesDatasetRelationship - Paper → Dataset
type UsesDatasetRelationship struct {
	PaperTitle     string  `json:"paper_title"`
	DatasetName    string  `json:"dataset_name"`
	Purpose        string  `json:"purpose"` // training, validation, testing, benchmark
	Results        string  `json:"results,omitempty"` // Performance metrics
	Metric         string  `json:"metric,omitempty"` // accuracy, F1, etc.
	Score          float64 `json:"score,omitempty"`
}

// ============================================================================
// QUERY RESULT TYPES
// ============================================================================

// GraphPath represents a path through the graph
type GraphPath struct {
	Nodes       []string               `json:"nodes"`
	Relationships []string             `json:"relationships"`
	Length      int                    `json:"length"`
	TotalWeight float64                `json:"total_weight"`
}

// AuthorImpact represents author influence metrics
type AuthorImpact struct {
	Name           string  `json:"name"`
	PaperCount     int     `json:"paper_count"`
	TotalCitations int     `json:"total_citations"`
	HIndex         int     `json:"h_index"`
	PageRank       float64 `json:"pagerank"`
	Centrality     float64 `json:"centrality"`
	TopPapers      []string `json:"top_papers"`
}

// InstitutionImpact represents institutional influence
type InstitutionImpact struct {
	Name           string   `json:"name"`
	PaperCount     int      `json:"paper_count"`
	TotalCitations int      `json:"total_citations"`
	TopAuthors     []string `json:"top_authors"`
	TopPapers      []string `json:"top_papers"`
	ResearchAreas  []string `json:"research_areas"`
}

// ConceptEvolution tracks how concepts emerge and spread
type ConceptEvolution struct {
	Name           string            `json:"name"`
	Timeline       map[int]int       `json:"timeline"` // Year → paper count
	FoundationalPapers []string      `json:"foundational_papers"`
	TrendScore     float64           `json:"trend_score"`
	GrowthRate     float64           `json:"growth_rate"`
}

// MethodLineage tracks evolution of methods
type MethodLineage struct {
	Name           string   `json:"name"`
	IntroducedBy   string   `json:"introduced_by"`
	IntroducedYear int      `json:"introduced_year"`
	Variants       []string `json:"variants"`
	UsedBy         []string `json:"used_by"` // Papers using this method
	ImprovedBy     []string `json:"improved_by"` // Papers improving it
}

// CollaborationNetwork represents co-authorship patterns
type CollaborationNetwork struct {
	Authors        []string                      `json:"authors"`
	Connections    []CoAuthorshipRelationship   `json:"connections"`
	CommunityID    int                           `json:"community_id,omitempty"`
	Density        float64                       `json:"density"`
}

// ============================================================================
// ANALYTICS TYPES
// ============================================================================

// TrendAnalysis represents research trends over time
type TrendAnalysis struct {
	Topic          string         `json:"topic"`
	Timeline       map[int]int    `json:"timeline"` // Year → count
	PeakYear       int            `json:"peak_year"`
	GrowthRate     float64        `json:"growth_rate"`
	PredictedTrend string         `json:"predicted_trend"` // growing, stable, declining
}

// CitationImpactMetrics for papers
type CitationImpactMetrics struct {
	PaperTitle     string  `json:"paper_title"`
	DirectCitations int    `json:"direct_citations"`
	IndirectCitations int  `json:"indirect_citations"` // Citations of citations
	PageRank       float64 `json:"pagerank"`
	HITS_Authority float64 `json:"hits_authority,omitempty"`
	HITS_Hub       float64 `json:"hits_hub,omitempty"`
}
