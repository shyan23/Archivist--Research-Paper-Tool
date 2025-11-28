## 🧩 Archivist Knowledge Graph Structure

Complete documentation of the heterogeneous multi-layer knowledge graph based on `plans/graph_ideas`.

---

## Overview

The Archivist knowledge graph is a **heterogeneous, multi-relational, temporal-aware graph** that captures the complete landscape of academic research.

### Key Features

✅ **10 Node Types** - Papers, Authors, Institutions, Concepts, Methods, Venues, Datasets
✅ **12 Relationship Types** - Citations, Co-authorship, Affiliations, Usage, Similarity
✅ **Directed & Weighted** - Captures knowledge flow with strength metrics
✅ **Temporal Awareness** - Track trends and evolution over time
✅ **Semantic Layer** - Vector embeddings for similarity search
✅ **Multi-Layer Connectivity** - Cross-entity traversal (Paper → Author → Institution)
✅ **Analytics Ready** - PageRank, centrality, community detection
✅ **Metadata Rich** - Contextual details for every entity
✅ **Expandable** - Easy to add Patents, Code repos, etc.
✅ **Hybrid Queries** - Symbolic (Cypher) + Semantic (Vectors)

---

## Node Types

### 1. Paper Node

**Central node** - represents a research document

```cypher
(:Paper {
  // Identification
  title: "Attention Is All You Need",
  doi: "10.5555/3295222.3295349",
  arxiv_id: "1706.03762",
  pdf_path: "lib/attention.pdf",

  // Temporal
  year: 2017,
  published_date: datetime("2017-06-12"),
  processed_at: datetime(),

  // Content
  abstract: "The dominant sequence...",
  keywords: ["attention", "transformer", "NLP"],

  // Metadata
  authors: ["Vaswani", "Shazeer", ...],
  venue: "NeurIPS",
  methodologies: ["self-attention", "positional-encoding"],
  datasets: ["WMT 2014"],
  metrics: ["BLEU", "perplexity"],

  // Embedding
  embedding_id: "paper_uuid_123",

  // Analytics (computed)
  citation_count: 45000,
  pagerank: 0.0234,
  h_index: 150
})
```

**Use cases:**
- Find most cited papers
- Track research trends
- Recommend related papers

---

### 2. Author Node

**Researcher entity**

```cypher
(:Author {
  name: "Ashish Vaswani",
  orcid: "0000-0002-1234-5678",
  email: "vaswani@google.com",
  affiliation: "Google Brain",
  field: "Machine Learning",
  h_index: 42,
  total_citations: 50000,
  active_since: 2012,
  paper_count: 25,

  // Analytics
  centrality: 0.85,
  influence: 0.92
})
```

**Use cases:**
- Find prolific authors
- Analyze collaboration networks
- Track research careers

---

### 3. Institution Node

**Organization or university**

```cypher
(:Institution {
  name: "Google Brain",
  country: "USA",
  city: "Mountain View",
  type: "company",
  research_domain: "AI/ML",
  website: "https://research.google/teams/brain/",

  // Analytics
  paper_count: 500,
  total_citations: 100000,
  impact: 0.95
})
```

**Use cases:**
- Institutional rankings
- Geographic research analysis
- Industry vs academia comparison

---

### 4. Concept Node

**Key scientific idea or topic**

```cypher
(:Concept {
  name: "self-attention",
  category: "methodology",
  description: "Mechanism for...",
  embedding_id: "concept_uuid_456",
  frequency: 1200,
  first_seen: 2014,

  // Analytics
  trend_score: 0.89,
  growth_rate: 2.5
})
```

**Use cases:**
- Topic discovery
- Trend analysis
- Concept evolution tracking

---

### 5. Method Node

**Specific algorithm or technique**

```cypher
(:Method {
  name: "Multi-Head Attention",
  type: "algorithm",
  description: "Extension of attention...",
  complexity: "O(n^2)",
  introduced_by: "Attention Is All You Need",
  introduced_year: 2017,
  usage_count: 3400,
  variants: ["Sparse Attention", "Linear Attention"]
})
```

**Use cases:**
- Method lineage tracking
- Find improvements/variants
- Complexity comparison

---

### 6. Venue Node

**Conference or journal**

```cypher
(:Venue {
  name: "Neural Information Processing Systems",
  short_name: "NeurIPS",
  type: "conference",
  rank: "A*",
  impact_factor: 9.2,
  acceptance_rate: 0.21,
  paper_count: 1500,
  citation_count: 200000
})
```

**Use cases:**
- Venue quality filtering
- Publication trend analysis
- Conference comparison

---

### 7. Dataset Node

**Benchmark datasets**

```cypher
(:Dataset {
  name: "ImageNet",
  type: "image",
  size: "14M images",
  description: "Large-scale visual...",
  introduced_year: 2009,
  url: "https://www.image-net.org/",
  usage_count: 5000,
  benchmark_for: ["classification", "detection"]
})
```

**Use cases:**
- Find papers using specific datasets
- Benchmark comparison
- Dataset popularity tracking

---

## Relationship Types

### 1. CITES (Paper → Paper)

**Academic citation** - captures knowledge flow

```cypher
(:Paper)-[:CITES {
  importance: "high",
  context: "We build upon the transformer...",
  citation_type: "methodology",
  section_type: "methods",
  timestamp: datetime(),
  weight: 1.0
}]->(:Paper)
```

**Importance levels:**
- `high` - Foundational, baseline comparison, main methodology
- `medium` - Related work, supporting evidence
- `low` - Brief mention, tangential

**Citation types:**
- `background` - Prior work
- `comparison` - Baseline/competing method
- `methodology` - Core technique used
- `results` - Performance comparison

---

### 2. WRITTEN_BY (Paper → Author)

**Authorship**

```cypher
(:Paper)-[:WRITTEN_BY {
  position: 1,
  is_corresponding: true
}]->(:Author)
```

---

### 3. AFFILIATED_WITH (Author → Institution)

**Institutional membership**

```cypher
(:Author)-[:AFFILIATED_WITH {
  role: "professor",
  start_year: 2015,
  end_year: 2020
}]->(:Institution)
```

---

### 4. USES_METHOD (Paper → Method)

**Methodological dependency**

```cypher
(:Paper)-[:USES_METHOD {
  is_main_method: true,
  description: "Core architecture"
}]->(:Method)
```

---

### 5. MENTIONS (Paper → Concept)

**Semantic connection**

```cypher
(:Paper)-[:MENTIONS {
  frequency: 15,
  is_core_theme: true
}]->(:Concept)
```

---

### 6. PUBLISHED_IN (Paper → Venue)

**Publication source**

```cypher
(:Paper)-[:PUBLISHED_IN {
  year: 2017,
  pages: "5998-6008",
  best_paper_award: true
}]->(:Venue)
```

---

### 7. CO_AUTHORED_WITH (Author ↔ Author)

**Collaboration network**

```cypher
(:Author)-[:CO_AUTHORED_WITH {
  joint_papers: 5,
  first_colab: 2015,
  last_colab: 2020,
  weight: 0.8
}]-(:Author)
```

---

### 8. EXTENDS (Paper → Paper)

**Conceptual lineage**

```cypher
(:Paper)-[:EXTENDS {
  extension_type: "improves",
  description: "Adds positional bias"
}]->(:Paper)
```

**Extension types:**
- `improves` - Enhanced performance
- `generalizes` - Broader applicability
- `specializes` - Domain-specific adaptation

---

### 9. SIMILAR_TO (Paper ↔ Paper)

**Semantic similarity**

```cypher
(:Paper)-[:SIMILAR_TO {
  score: 0.87,
  basis: "semantic",
  shared_concepts: ["attention", "transformer"],
  shared_methods: ["self-attention"]
}]-(:Paper)
```

**Basis types:**
- `semantic` - Embedding similarity
- `methodological` - Similar techniques
- `dataset` - Same benchmarks
- `results` - Similar findings

---

### 10. USES_DATASET (Paper → Dataset)

**Dataset usage**

```cypher
(:Paper)-[:USES_DATASET {
  purpose: "training",
  results: "Top-1 accuracy",
  metric: "accuracy",
  score: 0.88
}]->(:Dataset)
```

**Purposes:**
- `training` - Model training
- `validation` - Hyperparameter tuning
- `testing` - Final evaluation
- `benchmark` - Comparison with others

---

## Graph Queries

### Example 1: Find Influential Papers

```cypher
// Most cited papers in the last 5 years
MATCH (p:Paper)
WHERE p.year >= 2019
RETURN p.title, p.citation_count
ORDER BY p.citation_count DESC
LIMIT 10
```

### Example 2: Author Collaboration Network

```cypher
// Find co-authors within 2 hops
MATCH path = (a:Author {name: "Vaswani"})-[:CO_AUTHORED_WITH*1..2]-(colleague)
RETURN colleague.name, length(path) as distance
ORDER BY distance
```

### Example 3: Method Evolution

```cypher
// Track how attention mechanisms evolved
MATCH (paper:Paper)-[:USES_METHOD]->(m:Method {name: "self-attention"})
RETURN paper.title, paper.year
ORDER BY paper.year
```

### Example 4: Institution Impact

```cypher
// Top institutions by citation count
MATCH (i:Institution)<-[:AFFILIATED_WITH]-(a:Author)<-[:WRITTEN_BY]-(p:Paper)
RETURN i.name, sum(p.citation_count) as total_citations
ORDER BY total_citations DESC
LIMIT 10
```

### Example 5: Cross-Layer Query

```cypher
// Which institutions contributed to transformers?
MATCH (i:Institution)<-[:AFFILIATED_WITH]-(a:Author)<-[:WRITTEN_BY]-(p:Paper)
      -[:MENTIONS]->(c:Concept {name: "transformer"})
RETURN i.name, count(DISTINCT p) as paper_count
ORDER BY paper_count DESC
```

### Example 6: Temporal Trend

```cypher
// Growth of "attention" concept over time
MATCH (p:Paper)-[:MENTIONS]->(c:Concept {name: "attention"})
RETURN p.year, count(p) as paper_count
ORDER BY p.year
```

---

## Analytics Features

### 1. PageRank

```cypher
// Compute paper importance
CALL gds.pageRank.write({
  nodeProjection: 'Paper',
  relationshipProjection: 'CITES',
  writeProperty: 'pagerank'
})
```

### 2. Community Detection

```cypher
// Find research communities
CALL gds.louvain.write({
  nodeProjection: 'Paper',
  relationshipProjection: 'SIMILAR_TO',
  writeProperty: 'community'
})
```

### 3. Centrality

```cypher
// Find knowledge bridge authors
CALL gds.betweenness.write({
  nodeProjection: 'Author',
  relationshipProjection: 'CO_AUTHORED_WITH',
  writeProperty: 'centrality'
})
```

---

## Hybrid Queries (Symbolic + Semantic)

### Example: "Find papers similar to X but not citing Y"

```python
# Step 1: Vector search for similar papers
similar_papers = qdrant_client.search(
    collection_name="archivist_papers",
    query_vector=embedding_of_X,
    limit=100
)

# Step 2: Filter with graph query
for paper in similar_papers:
    query = """
    MATCH (p:Paper {title: $paper_title})
    WHERE NOT (p)-[:CITES]->(:Paper {title: $excluded_title})
    RETURN p
    """
    # Execute and combine results
```

---

## Integration with Qdrant

### Paper Chunks with Graph Metadata

```python
{
    "id": "paper_chunk_123",
    "vector": [0.1, 0.2, ...],
    "payload": {
        "paper_title": "Attention Is All You Need",
        "chunk_type": "methodology",
        "year": 2017,
        "authors": ["Vaswani", "Shazeer", ...],
        "venue": "NeurIPS",
        "methodologies": ["self-attention"],
        "citation_count": 45000,  // From Neo4j
        "pagerank": 0.0234         // From Neo4j
    }
}
```

---

## Future Enhancements

### Phase 2
- [ ] Add Patent nodes
- [ ] Add Code repository nodes
- [ ] Add Figure/Table nodes (multi-modal)
- [ ] Implement HITS algorithm
- [ ] Add temporal graph snapshots

### Phase 3
- [ ] Add Video/Lecture nodes
- [ ] Add Grant/Funding nodes
- [ ] Add Review/Commentary relationships
- [ ] Implement graph neural networks
- [ ] Add knowledge graph embedding (TransE, RotatE)

---

## Implementation Status

| Feature | Status | File |
|---------|--------|------|
| **Node Types** | ✅ Complete | `enhanced_models.go` |
| **Relationship Types** | ✅ Complete | `enhanced_models.go` |
| **Neo4j Builder** | ✅ Complete | `enhanced_neo4j_builder.go` |
| **Schema Initialization** | ✅ Complete | `enhanced_neo4j_builder.go` |
| **Basic Queries** | ✅ Complete | `enhanced_neo4j_builder.go` |
| **Analytics Queries** | ⚠️ Partial | `enhanced_neo4j_builder.go` |
| **Graph Algorithms** | ⏭️ Next Phase | TBD |
| **TUI Integration** | ⏭️ Next Phase | TBD |

---

## Usage Example

```go
// Initialize enhanced builder
builder, _ := graph.NewEnhancedNeo4jBuilder(config)
defer builder.Close(ctx)

// Initialize schema
builder.InitializeEnhancedSchema(ctx)

// Add nodes
author := &graph.AuthorNode{
    Name: "Ashish Vaswani",
    Field: "Machine Learning",
    HIndex: 42,
}
builder.AddAuthor(ctx, author)

// Add relationships
authorship := &graph.AuthorshipRelationship{
    PaperTitle: "Attention Is All You Need",
    AuthorName: "Ashish Vaswani",
    Position: 1,
    IsCorresponding: true,
}
builder.LinkPaperToAuthor(ctx, authorship)

// Query analytics
impact, _ := builder.GetAuthorImpact(ctx, "Ashish Vaswani")
fmt.Printf("Author: %s, Papers: %d, Citations: %d\n",
    impact.Name, impact.PaperCount, impact.TotalCitations)
```

---

**This structure captures the complete research landscape for powerful academic exploration! 🚀**
