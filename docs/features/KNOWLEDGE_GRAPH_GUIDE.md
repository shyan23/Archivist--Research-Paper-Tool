# 🧠 Archivist Knowledge Graph System

## Overview

The Archivist Knowledge Graph is a powerful feature that combines **Neo4j** (graph database), **Qdrant** (vector database), and **Gemini embeddings** to create an intelligent research paper exploration system.

### Key Features

✅ **Hybrid Search**: Combines vector similarity, graph traversal, and keyword matching
✅ **Citation Network**: Automatically extracts and visualizes paper citations
✅ **Semantic Similarity**: Finds related papers using embeddings
✅ **Efficient Storage**: Qdrant for vectors, Neo4j for relationships
✅ **Cost-Effective**: Optimized for 10-100 papers with Gemini API
✅ **Fast Queries**: Graph algorithms + vector search in milliseconds

---

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    Archivist CLI                             │
│  Commands: index | search | explore | cite | recommend      │
└────────────────┬────────────────────────────────────────────┘
                 │
        ┌────────┴────────┐
        │                 │
  ┌─────▼──────┐    ┌────▼─────┐    ┌────────┐
  │  Qdrant    │    │  Neo4j   │    │  Redis │
  │ (Vectors)  │    │  (Graph) │    │ (Cache)│
  │            │    │          │    │        │
  │ • Papers   │    │ • Papers │    │ • Hot  │
  │ • Chunks   │    │ • Cites  │    │   data │
  │ • Metadata │    │ • Similar│    │        │
  └────────────┘    └──────────┘    └────────┘
```

---

## Quick Start

### 1. Start Services

```bash
# Start Neo4j, Qdrant, and Redis
docker-compose -f docker-compose-graph.yml up -d

# Check services are running
docker ps

# Expected output:
# - archivist-neo4j (ports 7474, 7687)
# - archivist-qdrant (ports 6333, 6334)
# - archivist-redis (port 6379)
```

### 2. Configure Archivist

Edit `config/config.yaml`:

```yaml
# Knowledge Graph settings
graph:
  enabled: true

  neo4j:
    uri: "bolt://localhost:7687"
    username: "neo4j"
    password: "password"
    database: "archivist"

  citation_extraction:
    enabled: true
    prioritize_in_text: true

  search:
    vector_weight: 0.5   # 50% from vector similarity
    graph_weight: 0.3    # 30% from graph relationships
    keyword_weight: 0.2  # 20% from keyword matching

# Qdrant settings
qdrant:
  host: "localhost"
  port: 6333
  collection_name: "archivist_papers"

  vector:
    size: 768
    distance: "Cosine"

  chunking:
    chunk_size: 512
    chunk_overlap: 50
```

### 3. Build Knowledge Graph

```bash
# Process papers and build graph automatically
./archivist process lib/*.pdf --with-graph

# Or process existing papers into graph
./archivist graph build --from-processed

# Check graph statistics
./archivist graph stats
```

---

## Usage Examples

### Search Papers

```bash
# Semantic search
./archivist search "attention mechanisms in transformers"

# Search with filters
./archivist search "object detection" --year 2023 --dataset COCO

# Graph-based exploration
./archivist explore "ResNet" --depth 2 --show-citations

# Find similar papers
./archivist similar "lib/attention_is_all_you_need.pdf" --top-k 5
```

### Citation Analysis

```bash
# Show citation network
./archivist cite show "Attention Is All You Need"

# Find most cited papers
./archivist cite rank --top 10

# Show citation path between two papers
./archivist cite path "BERT" "GPT-3"
```

### Recommendations

```bash
# Get recommended papers based on your library
./archivist recommend --based-on lib/vit.pdf

# Generate reading roadmap
./archivist roadmap "learn transformers" --steps 5

# Find prerequisite papers
./archivist prereqs "Attention Is All You Need"
```

---

## How It Works

### 1. Paper Processing Pipeline

```
PDF → Parse → Extract Content
              ↓
         LLM Analysis (Gemini)
              ↓
    ┌────────┴────────┐
    ↓                 ↓
Generate Chunks    Extract Citations
    ↓                 ↓
Create Embeddings   Match to Graph
    ↓                 ↓
Store in Qdrant   Store in Neo4j
```

### 2. Hybrid Search

When you search for "attention mechanisms":

**Step 1: Vector Search** (Qdrant)
- Convert query → embedding vector
- Find top-K similar chunks using cosine similarity
- Score: 0.85 for "Attention Is All You Need"

**Step 2: Graph Search** (Neo4j)
- Find seed papers matching keywords
- Traverse citations (papers that cite/are cited)
- Traverse similarity edges
- Score: 0.92 for "Attention Is All You Need"

**Step 3: Keyword Matching**
- Simple token matching in content
- Boost for title matches
- Score: 0.78 for "Attention Is All You Need"

**Step 4: Fusion**
```
Final Score = (0.85 × 0.5) + (0.92 × 0.3) + (0.78 × 0.2)
            = 0.425 + 0.276 + 0.156
            = 0.857
```

### 3. Citation Extraction

Using Gemini LLM to extract:

```json
{
  "references": [
    {
      "index": 1,
      "authors": ["Vaswani et al."],
      "title": "Attention Is All You Need",
      "year": 2017,
      "venue": "NeurIPS"
    }
  ],
  "in_text_citations": [
    {
      "reference_index": 1,
      "context": "Building on the transformer architecture [1]...",
      "importance": "high"
    }
  ]
}
```

---

## Cost Analysis

### For 10-100 Papers

**Gemini API Costs:**

| Operation | API Calls | Cost per Paper | Total (50 papers) |
|-----------|-----------|----------------|-------------------|
| **Paper Analysis** | 1 | $0.002 | $0.10 |
| **Citation Extraction** | 1 | $0.001 | $0.05 |
| **Embeddings** (10 chunks/paper) | 10 | $0.0001 | $0.05 |
| **Total per paper** | - | **$0.003** | **$0.20** |

**Infrastructure:**
- Neo4j Community: **FREE**
- Qdrant: **FREE** (self-hosted)
- Redis: **FREE** (self-hosted)

**Total: ~$0.20 for 50 papers** 🎉

### Comparison with Ollama

| Metric | Gemini API | Ollama (Local) |
|--------|------------|----------------|
| **Cost** | $0.20 for 50 papers | $0 (free) |
| **Speed** | Fast (cloud) | Slow (GPU dependent) |
| **Quality** | High (state-of-art) | Good (model dependent) |
| **Setup** | 5 minutes | 1-2 hours |
| **Privacy** | API calls | Fully offline |

**Recommendation**: For 10-100 papers, **use Gemini API** for best balance of speed, cost, and quality.

---

## Advanced Features

### Graph Algorithms

```bash
# Compute PageRank (paper importance)
./archivist graph pagerank --iterations 20

# Find communities (paper clusters)
./archivist graph communities --algorithm louvain

# Compute similarity between all papers
./archivist graph compute-similarities --top-k 10
```

### Custom Queries

```bash
# Cypher query (Neo4j)
./archivist graph query --cypher "
  MATCH (p:Paper)-[:CITES]->(cited)
  RETURN p.title, count(cited) as citations
  ORDER BY citations DESC
  LIMIT 10
"

# Vector search with custom filters
./archivist search "CNN architectures" \
  --filter "year>=2020" \
  --filter "dataset=ImageNet" \
  --score-threshold 0.8
```

---

## Troubleshooting

### Services Not Starting

```bash
# Check logs
docker-compose -f docker-compose-graph.yml logs neo4j
docker-compose -f docker-compose-graph.yml logs qdrant

# Restart services
docker-compose -f docker-compose-graph.yml restart

# Clean slate
docker-compose -f docker-compose-graph.yml down -v
docker-compose -f docker-compose-graph.yml up -d
```

### Graph Is Empty

```bash
# Rebuild graph from processed papers
./archivist graph rebuild

# Check if papers are indexed
./archivist graph stats

# Manually add paper
./archivist graph add "lib/paper.pdf"
```

### Search Returns No Results

```bash
# Check collection
./archivist graph collections

# Verify embeddings
./archivist graph verify-embeddings

# Re-index papers
./archivist graph reindex
```

---

## Performance Optimization

### For 10-100 Papers (Current Scale)

✅ **Keep vectors in memory** (`on_disk: false`)
✅ **Use gRPC** for Qdrant (`use_grpc: true`)
✅ **Enable Redis caching** for embeddings
✅ **Precompute similarities** offline

### For 100-1000 Papers

- Switch to disk-backed vectors (`on_disk: true`)
- Increase Neo4j memory (`heap_max_size: 4G`)
- Use connection pooling
- Batch operations

### For 1000+ Papers

- Consider Qdrant Cloud or distributed setup
- Use Neo4j Enterprise for clustering
- Implement incremental indexing
- Add Elasticsearch for full-text search

---

## API Integration

### Python Client

```python
import requests

# Search papers
response = requests.post("http://localhost:8080/api/search", json={
    "query": "transformer architecture",
    "top_k": 10,
    "vector_weight": 0.5,
    "graph_weight": 0.3,
    "keyword_weight": 0.2
})

results = response.json()
for paper in results["papers"]:
    print(f"{paper['title']}: {paper['score']}")
```

### Go Client

```go
import "archivist/internal/graph"

// Initialize
builder, _ := graph.NewEnhancedGraphBuilder(graphConfig, qdrantConfig, apiKey, model)
defer builder.Close(ctx)

// Search
searchEngine := graph.NewHybridSearchEngine(builder, embeddingClient)
results, _ := searchEngine.Search(ctx, &vectorstore.HybridSearchQuery{
    Query: "attention mechanisms",
    TopK: 10,
    VectorWeight: 0.5,
    GraphWeight: 0.3,
    KeywordWeight: 0.2,
})
```

---

## Future Enhancements

### Phase 2 (Next 2-4 weeks)

- [ ] Web dashboard for graph visualization
- [ ] Interactive TUI for graph exploration
- [ ] Author co-citation analysis
- [ ] Concept extraction and linking
- [ ] Time-series analysis (research trends)

### Phase 3 (1-2 months)

- [ ] Multi-modal embeddings (figures, tables)
- [ ] Cross-domain paper recommendations
- [ ] Automated literature review generation
- [ ] Integration with reference managers (Zotero, Mendeley)

---

## FAQ

**Q: Why Qdrant instead of FAISS?**
A: Qdrant offers persistence, CRUD operations, metadata filtering, and distributed scaling—all missing in FAISS.

**Q: Can I use this offline?**
A: Partially. Neo4j and Qdrant run locally, but embeddings require Gemini API. Switch to Ollama for fully offline.

**Q: How do I backup my graph?**
A:
```bash
# Neo4j
docker exec archivist-neo4j neo4j-admin dump --database=neo4j

# Qdrant (auto-persisted to ./qdrant_storage)
docker exec archivist-qdrant qdrant-backup
```

**Q: Can I use Claude instead of Gemini?**
A: Yes, modify `internal/graph/citation_extractor.go` to use Claude API. Cost will be higher (~$0.50 for 50 papers).

---

## Support

- 📖 Documentation: `/docs`
- 💬 Issues: GitHub Issues
- 📧 Contact: [your-email]

---

**Built with ❤️ for CS students exploring research**
