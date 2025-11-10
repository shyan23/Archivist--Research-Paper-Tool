package graph

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// GraphConfig holds Neo4j configuration
type GraphConfig struct {
	URI      string
	Username string
	Password string
	Database string
}

// GraphBuilder handles Neo4j knowledge graph construction
type GraphBuilder struct {
	driver neo4j.DriverWithContext
	config *GraphConfig
}

// NewGraphBuilder creates a new graph builder
func NewGraphBuilder(config *GraphConfig) (*GraphBuilder, error) {
	driver, err := neo4j.NewDriverWithContext(
		config.URI,
		neo4j.BasicAuth(config.Username, config.Password, ""),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create Neo4j driver: %w", err)
	}

	// Verify connectivity
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := driver.VerifyConnectivity(ctx); err != nil {
		driver.Close(ctx)
		return nil, fmt.Errorf("failed to connect to Neo4j: %w", err)
	}

	return &GraphBuilder{
		driver: driver,
		config: config,
	}, nil
}

// Close closes the Neo4j driver
func (gb *GraphBuilder) Close(ctx context.Context) error {
	return gb.driver.Close(ctx)
}

// InitializeSchema creates indexes and constraints
func (gb *GraphBuilder) InitializeSchema(ctx context.Context) error {
	session := gb.driver.NewSession(ctx, neo4j.SessionConfig{
		DatabaseName: gb.config.Database,
	})
	defer session.Close(ctx)

	queries := []string{
		// Unique constraint on paper title
		"CREATE CONSTRAINT paper_title_unique IF NOT EXISTS FOR (p:Paper) REQUIRE p.title IS UNIQUE",

		// Indexes for faster lookups
		"CREATE INDEX paper_title_index IF NOT EXISTS FOR (p:Paper) ON (p.title)",
		"CREATE INDEX paper_year_index IF NOT EXISTS FOR (p:Paper) ON (p.year)",
		"CREATE INDEX concept_name_index IF NOT EXISTS FOR (c:Concept) ON (c.name)",
		"CREATE INDEX concept_category_index IF NOT EXISTS FOR (c:Concept) ON (c.category)",
	}

	for _, query := range queries {
		_, err := session.Run(ctx, query, nil)
		if err != nil {
			return fmt.Errorf("failed to execute schema query: %w", err)
		}
	}

	log.Println("✓ Neo4j schema initialized (constraints & indexes)")
	return nil
}

// AddPaper creates or updates a paper node
func (gb *GraphBuilder) AddPaper(ctx context.Context, paper *PaperNode) error {
	session := gb.driver.NewSession(ctx, neo4j.SessionConfig{
		DatabaseName: gb.config.Database,
	})
	defer session.Close(ctx)

	query := `
		MERGE (p:Paper {title: $title})
		SET p.pdf_path = $pdf_path,
			p.processed_at = datetime($processed_at),
			p.embedding_id = $embedding_id,
			p.methodologies = $methodologies,
			p.datasets = $datasets,
			p.metrics = $metrics,
			p.year = $year,
			p.authors = $authors,
			p.abstract = $abstract
		RETURN p.title as title
	`

	params := map[string]interface{}{
		"title":         paper.Title,
		"pdf_path":      paper.PDFPath,
		"processed_at":  paper.ProcessedAt.Format(time.RFC3339),
		"embedding_id":  paper.EmbeddingID,
		"methodologies": paper.Methodologies,
		"datasets":      paper.Datasets,
		"metrics":       paper.Metrics,
		"year":          paper.Year,
		"authors":       paper.Authors,
		"abstract":      paper.Abstract,
	}

	_, err := session.Run(ctx, query, params)
	if err != nil {
		return fmt.Errorf("failed to add paper node: %w", err)
	}

	log.Printf("✓ Added paper node: %s", paper.Title)
	return nil
}

// AddCitation creates a citation relationship
func (gb *GraphBuilder) AddCitation(ctx context.Context, citation *CitationRelationship) error {
	session := gb.driver.NewSession(ctx, neo4j.SessionConfig{
		DatabaseName: gb.config.Database,
	})
	defer session.Close(ctx)

	query := `
		MATCH (source:Paper {title: $source})
		MATCH (target:Paper {title: $target})
		MERGE (source)-[r:CITES {
			importance: $importance,
			context: $context
		}]->(target)
		RETURN source.title, target.title
	`

	params := map[string]interface{}{
		"source":     citation.SourcePaper,
		"target":     citation.TargetPaper,
		"importance": citation.Importance,
		"context":    citation.Context,
	}

	_, err := session.Run(ctx, query, params)
	if err != nil {
		return fmt.Errorf("failed to add citation: %w", err)
	}

	return nil
}

// AddConcept creates or updates a concept node
func (gb *GraphBuilder) AddConcept(ctx context.Context, concept *ConceptNode) error {
	session := gb.driver.NewSession(ctx, neo4j.SessionConfig{
		DatabaseName: gb.config.Database,
	})
	defer session.Close(ctx)

	query := `
		MERGE (c:Concept {name: $name})
		SET c.category = $category,
			c.frequency = $frequency
		RETURN c.name
	`

	params := map[string]interface{}{
		"name":      concept.Name,
		"category":  concept.Category,
		"frequency": concept.Frequency,
	}

	_, err := session.Run(ctx, query, params)
	if err != nil {
		return fmt.Errorf("failed to add concept: %w", err)
	}

	return nil
}

// LinkPaperToConcept creates a USES_CONCEPT relationship
func (gb *GraphBuilder) LinkPaperToConcept(ctx context.Context, rel *ConceptRelationship) error {
	session := gb.driver.NewSession(ctx, neo4j.SessionConfig{
		DatabaseName: gb.config.Database,
	})
	defer session.Close(ctx)

	query := `
		MATCH (p:Paper {title: $paper})
		MERGE (c:Concept {name: $concept})
		MERGE (p)-[r:USES_CONCEPT {section: $section}]->(c)
		RETURN p.title, c.name
	`

	params := map[string]interface{}{
		"paper":   rel.PaperTitle,
		"concept": rel.Concept,
		"section": rel.Section,
	}

	_, err := session.Run(ctx, query, params)
	if err != nil {
		return fmt.Errorf("failed to link paper to concept: %w", err)
	}

	return nil
}

// AddSimilarity creates a SIMILAR_TO relationship
func (gb *GraphBuilder) AddSimilarity(ctx context.Context, sim *SimilarityRelationship) error {
	session := gb.driver.NewSession(ctx, neo4j.SessionConfig{
		DatabaseName: gb.config.Database,
	})
	defer session.Close(ctx)

	query := `
		MATCH (p1:Paper {title: $paper1})
		MATCH (p2:Paper {title: $paper2})
		MERGE (p1)-[r:SIMILAR_TO {
			score: $score,
			basis: $basis
		}]->(p2)
		RETURN p1.title, p2.title
	`

	params := map[string]interface{}{
		"paper1": sim.Paper1,
		"paper2": sim.Paper2,
		"score":  sim.Score,
		"basis":  sim.Basis,
	}

	_, err := session.Run(ctx, query, params)
	if err != nil {
		return fmt.Errorf("failed to add similarity: %w", err)
	}

	return nil
}

// GetPaper retrieves a paper node by title
func (gb *GraphBuilder) GetPaper(ctx context.Context, title string) (*PaperNode, error) {
	session := gb.driver.NewSession(ctx, neo4j.SessionConfig{
		DatabaseName: gb.config.Database,
	})
	defer session.Close(ctx)

	query := `
		MATCH (p:Paper {title: $title})
		RETURN p.title as title,
			   p.pdf_path as pdf_path,
			   p.methodologies as methodologies,
			   p.datasets as datasets,
			   p.metrics as metrics,
			   p.year as year,
			   p.authors as authors,
			   p.abstract as abstract
	`

	result, err := session.Run(ctx, query, map[string]interface{}{"title": title})
	if err != nil {
		return nil, fmt.Errorf("failed to get paper: %w", err)
	}

	if result.Next(ctx) {
		record := result.Record()
		paper := &PaperNode{
			Title: record.Values[0].(string),
		}

		if val, ok := record.Get("pdf_path"); ok && val != nil {
			paper.PDFPath = val.(string)
		}
		if val, ok := record.Get("methodologies"); ok && val != nil {
			if methodologies, ok := val.([]interface{}); ok {
				for _, m := range methodologies {
					paper.Methodologies = append(paper.Methodologies, m.(string))
				}
			}
		}
		// Add more field extractions as needed...

		return paper, nil
	}

	return nil, fmt.Errorf("paper not found: %s", title)
}

// PaperExists checks if a paper exists in the graph
func (gb *GraphBuilder) PaperExists(ctx context.Context, title string) (bool, error) {
	session := gb.driver.NewSession(ctx, neo4j.SessionConfig{
		DatabaseName: gb.config.Database,
	})
	defer session.Close(ctx)

	query := "MATCH (p:Paper {title: $title}) RETURN count(p) as count"
	result, err := session.Run(ctx, query, map[string]interface{}{"title": title})
	if err != nil {
		return false, err
	}

	if result.Next(ctx) {
		count := result.Record().Values[0].(int64)
		return count > 0, nil
	}

	return false, nil
}

// GetStats returns graph statistics
func (gb *GraphBuilder) GetStats(ctx context.Context) (*GraphStats, error) {
	session := gb.driver.NewSession(ctx, neo4j.SessionConfig{
		DatabaseName: gb.config.Database,
	})
	defer session.Close(ctx)

	query := `
		MATCH (p:Paper)
		OPTIONAL MATCH (c:Concept)
		OPTIONAL MATCH ()-[cites:CITES]->()
		OPTIONAL MATCH ()-[sim:SIMILAR_TO]->()
		RETURN count(DISTINCT p) as papers,
			   count(DISTINCT c) as concepts,
			   count(DISTINCT cites) as citations,
			   count(DISTINCT sim) as similarities
	`

	result, err := session.Run(ctx, query, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get stats: %w", err)
	}

	if result.Next(ctx) {
		record := result.Record()
		return &GraphStats{
			PaperCount:      int(record.Values[0].(int64)),
			ConceptCount:    int(record.Values[1].(int64)),
			CitationCount:   int(record.Values[2].(int64)),
			SimilarityCount: int(record.Values[3].(int64)),
			LastUpdated:     time.Now(),
		}, nil
	}

	return nil, fmt.Errorf("failed to retrieve stats")
}

// DeletePaper removes a paper and all its relationships
func (gb *GraphBuilder) DeletePaper(ctx context.Context, title string) error {
	session := gb.driver.NewSession(ctx, neo4j.SessionConfig{
		DatabaseName: gb.config.Database,
	})
	defer session.Close(ctx)

	query := `
		MATCH (p:Paper {title: $title})
		DETACH DELETE p
	`

	_, err := session.Run(ctx, query, map[string]interface{}{"title": title})
	if err != nil {
		return fmt.Errorf("failed to delete paper: %w", err)
	}

	log.Printf("✓ Deleted paper: %s", title)
	return nil
}

// ClearGraph removes all nodes and relationships (use with caution!)
func (gb *GraphBuilder) ClearGraph(ctx context.Context) error {
	session := gb.driver.NewSession(ctx, neo4j.SessionConfig{
		DatabaseName: gb.config.Database,
	})
	defer session.Close(ctx)

	query := "MATCH (n) DETACH DELETE n"
	_, err := session.Run(ctx, query, nil)
	if err != nil {
		return fmt.Errorf("failed to clear graph: %w", err)
	}

	log.Println("⚠️  Cleared entire graph")
	return nil
}
