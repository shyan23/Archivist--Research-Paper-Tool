package graph

import (
	"context"
	"fmt"
	"log"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// EnhancedNeo4jBuilder handles the heterogeneous multi-layer knowledge graph
type EnhancedNeo4jBuilder struct {
	*GraphBuilder // Embed the basic builder
}

// NewEnhancedNeo4jBuilder creates a new enhanced Neo4j builder
func NewEnhancedNeo4jBuilder(config *GraphConfig) (*EnhancedNeo4jBuilder, error) {
	basicBuilder, err := NewGraphBuilder(config)
	if err != nil {
		return nil, err
	}

	return &EnhancedNeo4jBuilder{
		GraphBuilder: basicBuilder,
	}, nil
}

// InitializeEnhancedSchema creates all indexes and constraints for the heterogeneous graph
func (eb *EnhancedNeo4jBuilder) InitializeEnhancedSchema(ctx context.Context) error {
	session := eb.driver.NewSession(ctx, neo4j.SessionConfig{
		DatabaseName: eb.config.Database,
	})
	defer session.Close(ctx)

	queries := []string{
		// Paper constraints
		"CREATE CONSTRAINT paper_title_unique IF NOT EXISTS FOR (p:Paper) REQUIRE p.title IS UNIQUE",
		"CREATE CONSTRAINT paper_doi_unique IF NOT EXISTS FOR (p:Paper) REQUIRE p.doi IS UNIQUE",

		// Author constraints
		"CREATE CONSTRAINT author_name_unique IF NOT EXISTS FOR (a:Author) REQUIRE a.name IS UNIQUE",

		// Institution constraints
		"CREATE CONSTRAINT institution_name_unique IF NOT EXISTS FOR (i:Institution) REQUIRE i.name IS UNIQUE",

		// Concept constraints
		"CREATE CONSTRAINT concept_name_unique IF NOT EXISTS FOR (c:Concept) REQUIRE c.name IS UNIQUE",

		// Method constraints
		"CREATE CONSTRAINT method_name_unique IF NOT EXISTS FOR (m:Method) REQUIRE m.name IS UNIQUE",

		// Venue constraints
		"CREATE CONSTRAINT venue_name_unique IF NOT EXISTS FOR (v:Venue) REQUIRE v.name IS UNIQUE",

		// Dataset constraints
		"CREATE CONSTRAINT dataset_name_unique IF NOT EXISTS FOR (d:Dataset) REQUIRE d.name IS UNIQUE",

		// Indexes for fast lookups
		"CREATE INDEX paper_year_index IF NOT EXISTS FOR (p:Paper) ON (p.year)",
		"CREATE INDEX paper_venue_index IF NOT EXISTS FOR (p:Paper) ON (p.venue)",
		"CREATE INDEX author_field_index IF NOT EXISTS FOR (a:Author) ON (a.field)",
		"CREATE INDEX institution_country_index IF NOT EXISTS FOR (i:Institution) ON (i.country)",
		"CREATE INDEX concept_category_index IF NOT EXISTS FOR (c:Concept) ON (c.category)",
		"CREATE INDEX method_type_index IF NOT EXISTS FOR (m:Method) ON (m.type)",
		"CREATE INDEX venue_type_index IF NOT EXISTS FOR (v:Venue) ON (v.type)",

		// Composite indexes for complex queries
		"CREATE INDEX paper_year_venue_index IF NOT EXISTS FOR (p:Paper) ON (p.year, p.venue)",
	}

	for _, query := range queries {
		_, err := session.Run(ctx, query, nil)
		if err != nil {
			return fmt.Errorf("failed to execute schema query: %w", err)
		}
	}

	log.Println("✓ Enhanced Neo4j schema initialized (all node types & indexes)")
	return nil
}

// ============================================================================
// NODE CREATION METHODS
// ============================================================================

// AddAuthor creates an author node
func (eb *EnhancedNeo4jBuilder) AddAuthor(ctx context.Context, author *AuthorNode) error {
	session := eb.driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: eb.config.Database})
	defer session.Close(ctx)

	query := `
		MERGE (a:Author {name: $name})
		SET a.orcid = $orcid,
			a.email = $email,
			a.affiliation = $affiliation,
			a.field = $field,
			a.h_index = $h_index,
			a.total_citations = $total_citations,
			a.active_since = $active_since,
			a.paper_count = $paper_count
		RETURN a.name
	`

	params := map[string]interface{}{
		"name":            author.Name,
		"orcid":           author.ORCID,
		"email":           author.Email,
		"affiliation":     author.Affiliation,
		"field":           author.Field,
		"h_index":         author.HIndex,
		"total_citations": author.TotalCitations,
		"active_since":    author.ActiveSince,
		"paper_count":     author.PaperCount,
	}

	_, err := session.Run(ctx, query, params)
	return err
}

// AddInstitution creates an institution node
func (eb *EnhancedNeo4jBuilder) AddInstitution(ctx context.Context, inst *InstitutionNode) error {
	session := eb.driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: eb.config.Database})
	defer session.Close(ctx)

	query := `
		MERGE (i:Institution {name: $name})
		SET i.country = $country,
			i.city = $city,
			i.type = $type,
			i.research_domain = $research_domain,
			i.website = $website
		RETURN i.name
	`

	params := map[string]interface{}{
		"name":            inst.Name,
		"country":         inst.Country,
		"city":            inst.City,
		"type":            inst.Type,
		"research_domain": inst.ResearchDomain,
		"website":         inst.Website,
	}

	_, err := session.Run(ctx, query, params)
	return err
}

// AddMethod creates a method node
func (eb *EnhancedNeo4jBuilder) AddMethod(ctx context.Context, method *MethodNode) error {
	session := eb.driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: eb.config.Database})
	defer session.Close(ctx)

	query := `
		MERGE (m:Method {name: $name})
		SET m.type = $type,
			m.description = $description,
			m.complexity = $complexity,
			m.introduced_by = $introduced_by,
			m.introduced_year = $introduced_year,
			m.usage_count = $usage_count,
			m.variants = $variants
		RETURN m.name
	`

	params := map[string]interface{}{
		"name":            method.Name,
		"type":            method.Type,
		"description":     method.Description,
		"complexity":      method.Complexity,
		"introduced_by":   method.IntroducedBy,
		"introduced_year": method.IntroducedYear,
		"usage_count":     method.UsageCount,
		"variants":        method.Variants,
	}

	_, err := session.Run(ctx, query, params)
	return err
}

// AddVenue creates a venue node
func (eb *EnhancedNeo4jBuilder) AddVenue(ctx context.Context, venue *VenueNode) error {
	session := eb.driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: eb.config.Database})
	defer session.Close(ctx)

	query := `
		MERGE (v:Venue {name: $name})
		SET v.short_name = $short_name,
			v.type = $type,
			v.rank = $rank,
			v.impact_factor = $impact_factor,
			v.acceptance_rate = $acceptance_rate
		RETURN v.name
	`

	params := map[string]interface{}{
		"name":            venue.Name,
		"short_name":      venue.ShortName,
		"type":            venue.Type,
		"rank":            venue.Rank,
		"impact_factor":   venue.ImpactFactor,
		"acceptance_rate": venue.AcceptanceRate,
	}

	_, err := session.Run(ctx, query, params)
	return err
}

// AddDataset creates a dataset node
func (eb *EnhancedNeo4jBuilder) AddDataset(ctx context.Context, dataset *DatasetNode) error {
	session := eb.driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: eb.config.Database})
	defer session.Close(ctx)

	query := `
		MERGE (d:Dataset {name: $name})
		SET d.type = $type,
			d.size = $size,
			d.description = $description,
			d.introduced_year = $introduced_year,
			d.url = $url,
			d.usage_count = $usage_count,
			d.benchmark_for = $benchmark_for
		RETURN d.name
	`

	params := map[string]interface{}{
		"name":            dataset.Name,
		"type":            dataset.Type,
		"size":            dataset.Size,
		"description":     dataset.Description,
		"introduced_year": dataset.IntroducedYear,
		"url":             dataset.URL,
		"usage_count":     dataset.UsageCount,
		"benchmark_for":   dataset.BenchmarkFor,
	}

	_, err := session.Run(ctx, query, params)
	return err
}

// ============================================================================
// RELATIONSHIP CREATION METHODS
// ============================================================================

// LinkPaperToAuthor creates authorship relationship
func (eb *EnhancedNeo4jBuilder) LinkPaperToAuthor(ctx context.Context, rel *AuthorshipRelationship) error {
	session := eb.driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: eb.config.Database})
	defer session.Close(ctx)

	query := `
		MATCH (p:Paper {title: $paper_title})
		MERGE (a:Author {name: $author_name})
		MERGE (p)-[r:WRITTEN_BY {position: $position, is_corresponding: $is_corresponding}]->(a)
		RETURN p.title, a.name
	`

	params := map[string]interface{}{
		"paper_title":      rel.PaperTitle,
		"author_name":      rel.AuthorName,
		"position":         rel.Position,
		"is_corresponding": rel.IsCorresponding,
	}

	_, err := session.Run(ctx, query, params)
	return err
}

// LinkAuthorToInstitution creates affiliation relationship
func (eb *EnhancedNeo4jBuilder) LinkAuthorToInstitution(ctx context.Context, rel *AffiliationRelationship) error {
	session := eb.driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: eb.config.Database})
	defer session.Close(ctx)

	query := `
		MATCH (a:Author {name: $author_name})
		MERGE (i:Institution {name: $institution_name})
		MERGE (a)-[r:AFFILIATED_WITH {
			role: $role,
			start_year: $start_year,
			end_year: $end_year
		}]->(i)
		RETURN a.name, i.name
	`

	params := map[string]interface{}{
		"author_name":       rel.AuthorName,
		"institution_name":  rel.InstitutionName,
		"role":              rel.Role,
		"start_year":        rel.StartYear,
		"end_year":          rel.EndYear,
	}

	_, err := session.Run(ctx, query, params)
	return err
}

// LinkPaperToMethod creates uses-method relationship
func (eb *EnhancedNeo4jBuilder) LinkPaperToMethod(ctx context.Context, rel *UsesMethodRelationship) error {
	session := eb.driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: eb.config.Database})
	defer session.Close(ctx)

	query := `
		MATCH (p:Paper {title: $paper_title})
		MERGE (m:Method {name: $method_name})
		MERGE (p)-[r:USES_METHOD {
			is_main_method: $is_main_method,
			description: $description
		}]->(m)
		RETURN p.title, m.name
	`

	params := map[string]interface{}{
		"paper_title":    rel.PaperTitle,
		"method_name":    rel.MethodName,
		"is_main_method": rel.IsMainMethod,
		"description":    rel.Description,
	}

	_, err := session.Run(ctx, query, params)
	return err
}

// LinkPaperToVenue creates published-in relationship
func (eb *EnhancedNeo4jBuilder) LinkPaperToVenue(ctx context.Context, rel *PublishedInRelationship) error {
	session := eb.driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: eb.config.Database})
	defer session.Close(ctx)

	query := `
		MATCH (p:Paper {title: $paper_title})
		MERGE (v:Venue {name: $venue_name})
		MERGE (p)-[r:PUBLISHED_IN {
			year: $year,
			pages: $pages,
			best_paper_award: $best_paper_award
		}]->(v)
		RETURN p.title, v.name
	`

	params := map[string]interface{}{
		"paper_title":      rel.PaperTitle,
		"venue_name":       rel.VenueName,
		"year":             rel.Year,
		"pages":            rel.Pages,
		"best_paper_award": rel.BestPaperAward,
	}

	_, err := session.Run(ctx, query, params)
	return err
}

// LinkCoAuthors creates co-authorship relationship
func (eb *EnhancedNeo4jBuilder) LinkCoAuthors(ctx context.Context, rel *CoAuthorshipRelationship) error {
	session := eb.driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: eb.config.Database})
	defer session.Close(ctx)

	query := `
		MATCH (a1:Author {name: $author1})
		MATCH (a2:Author {name: $author2})
		MERGE (a1)-[r:CO_AUTHORED_WITH {
			joint_papers: $joint_papers,
			first_colab: $first_colab,
			last_colab: $last_colab,
			weight: $weight
		}]-(a2)
		RETURN a1.name, a2.name
	`

	params := map[string]interface{}{
		"author1":      rel.Author1,
		"author2":      rel.Author2,
		"joint_papers": rel.JointPapers,
		"first_colab":  rel.FirstColab,
		"last_colab":   rel.LastColab,
		"weight":       rel.Weight,
	}

	_, err := session.Run(ctx, query, params)
	return err
}

// LinkPaperToDataset creates uses-dataset relationship
func (eb *EnhancedNeo4jBuilder) LinkPaperToDataset(ctx context.Context, rel *UsesDatasetRelationship) error {
	session := eb.driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: eb.config.Database})
	defer session.Close(ctx)

	query := `
		MATCH (p:Paper {title: $paper_title})
		MERGE (d:Dataset {name: $dataset_name})
		MERGE (p)-[r:USES_DATASET {
			purpose: $purpose,
			results: $results,
			metric: $metric,
			score: $score
		}]->(d)
		RETURN p.title, d.name
	`

	params := map[string]interface{}{
		"paper_title":  rel.PaperTitle,
		"dataset_name": rel.DatasetName,
		"purpose":      rel.Purpose,
		"results":      rel.Results,
		"metric":       rel.Metric,
		"score":        rel.Score,
	}

	_, err := session.Run(ctx, query, params)
	return err
}

// AddExtensionRelationship creates extends/improves relationship
func (eb *EnhancedNeo4jBuilder) AddExtensionRelationship(ctx context.Context, rel *ExtendsRelationship) error {
	session := eb.driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: eb.config.Database})
	defer session.Close(ctx)

	query := `
		MATCH (source:Paper {title: $source_paper})
		MATCH (target:Paper {title: $target_paper})
		MERGE (source)-[r:EXTENDS {
			extension_type: $extension_type,
			description: $description
		}]->(target)
		RETURN source.title, target.title
	`

	params := map[string]interface{}{
		"source_paper":   rel.SourcePaper,
		"target_paper":   rel.TargetPaper,
		"extension_type": rel.ExtensionType,
		"description":    rel.Description,
	}

	_, err := session.Run(ctx, query, params)
	return err
}

// ============================================================================
// ANALYTICS QUERY METHODS
// ============================================================================

// GetAuthorImpact computes author influence metrics
func (eb *EnhancedNeo4jBuilder) GetAuthorImpact(ctx context.Context, authorName string) (*AuthorImpact, error) {
	session := eb.driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: eb.config.Database})
	defer session.Close(ctx)

	query := `
		MATCH (a:Author {name: $author_name})<-[:WRITTEN_BY]-(p:Paper)
		WITH a, count(p) as paper_count, collect(p.title) as papers
		RETURN a.name as name,
			   paper_count,
			   a.total_citations as total_citations,
			   a.h_index as h_index,
			   papers[..5] as top_papers
	`

	result, err := session.Run(ctx, query, map[string]interface{}{"author_name": authorName})
	if err != nil {
		return nil, err
	}

	if result.Next(ctx) {
		record := result.Record()
		impact := &AuthorImpact{
			Name:           record.Values[0].(string),
			PaperCount:     int(record.Values[1].(int64)),
			TotalCitations: int(record.Values[2].(int64)),
			HIndex:         int(record.Values[3].(int64)),
		}

		if topPapers, ok := record.Values[4].([]interface{}); ok {
			for _, p := range topPapers {
				impact.TopPapers = append(impact.TopPapers, p.(string))
			}
		}

		return impact, nil
	}

	return nil, fmt.Errorf("author not found: %s", authorName)
}

// GetCollaborationNetwork retrieves co-authorship network
func (eb *EnhancedNeo4jBuilder) GetCollaborationNetwork(ctx context.Context, authorName string, depth int) (*CollaborationNetwork, error) {
	session := eb.driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: eb.config.Database})
	defer session.Close(ctx)

	query := `
		MATCH path = (a:Author {name: $author_name})-[:CO_AUTHORED_WITH*1..` + fmt.Sprintf("%d", depth) + `]-(colleague:Author)
		WITH collect(DISTINCT colleague.name) as authors,
		     collect({author1: a.name, author2: colleague.name, weight: last(relationships(path)).weight}) as connections
		RETURN authors, connections
	`

	result, err := session.Run(ctx, query, map[string]interface{}{"author_name": authorName})
	if err != nil {
		return nil, err
	}

	if result.Next(ctx) {
		record := result.Record()
		network := &CollaborationNetwork{}

		// Extract authors
		if authors, ok := record.Values[0].([]interface{}); ok {
			for _, a := range authors {
				network.Authors = append(network.Authors, a.(string))
			}
		}

		// Extract connections
		// This would need more processing in a real implementation

		return network, nil
	}

	return nil, fmt.Errorf("no collaboration network found for: %s", authorName)
}
