# Graph Service API Upgrades

## ✨ New Features Added

### 1. Semantic Search & Recommendations
- **Hybrid Storage**: Metadata in Neo4j, vectors in Qdrant
- **Gemini Embeddings**: text-embedding-004 model
- **Similarity Threshold**: 0.85 (moderate, configurable)

**New Endpoints:**
```bash
# Semantic search with natural language
POST /api/graph/search/semantic?query=attention+mechanism&top_k=10&threshold=0.7

# Get similar papers (recommendations)
GET /api/graph/recommend/{paper_title}?top_k=10
```

### 2. Citation Impact Analysis
All metrics implemented:
- ✅ H-index calculation
- ✅ Citation timeline (temporal trends)
- ✅ Influential citations
- ✅ Citation context extraction

**New Endpoints:**
```bash
# H-index for author
GET /api/graph/citations/h-index/{author_name}

# Citation timeline
GET /api/graph/citations/timeline/{paper_title}

# Most influential citations
GET /api/graph/citations/influential/{paper_title}?top_k=10

# Citation contexts (why papers cite this)
GET /api/graph/citations/contexts/{paper_title}

# Complete analysis (all metrics combined)
GET /api/graph/citations/analysis/{paper_title}
```

### 3. Advanced Graph Queries
- Path finding between papers
- Trending methods by year
- SIMILAR_TO relationships

**New Endpoints:**
```bash
# Find connection path
GET /api/graph/path?paper1=title1&paper2=title2&max_hops=5

# Trending methods
GET /api/graph/trending/{year}?top_k=10
```

## 📦 New Dependencies

Added to `requirements.txt`:
```
qdrant-client==1.7.0
numpy==1.26.3
```

## 🏗️ Architecture Updates

### Before:
```
Neo4j (graph) ← Workers ← Kafka
```

### After:
```
         ┌─ Neo4j (metadata + relationships)
         │
Workers ─┤
         │
         └─ Qdrant (embeddings + semantic search)
```

## 🚀 Usage Examples

### Semantic Search
```bash
curl -X POST "http://localhost:8081/api/graph/search/semantic" \
  -H "Content-Type: application/json" \
  -d '{
    "query": "papers about transformers and attention",
    "top_k": 5,
    "threshold": 0.7
  }'
```

### H-Index Calculation
```bash
curl "http://localhost:8081/api/graph/citations/h-index/Ashish%20Vaswani"
```

### Find Paper Connection
```bash
curl "http://localhost:8081/api/graph/path?paper1=Attention%20Is%20All%20You%20Need&paper2=BERT"
```

### Get Recommendations
```bash
curl "http://localhost:8081/api/graph/recommend/Attention%20Is%20All%20You%20Need?top_k=10"
```

### Citation Analysis
```bash
# Timeline
curl "http://localhost:8081/api/graph/citations/timeline/Attention%20Is%20All%20You%20Need"

# Influential papers that cite this
curl "http://localhost:8081/api/graph/citations/influential/Attention%20Is%20All%20You%20Need"

# Why papers cite this (contexts)
curl "http://localhost:8081/api/graph/citations/contexts/Attention%20Is%20All%20You%20Need"

# Complete analysis
curl "http://localhost:8081/api/graph/citations/analysis/Attention%20Is%20All%20You%20Need"
```

### Trending Methods
```bash
curl "http://localhost:8081/api/graph/trending/2023?top_k=10"
```

## 🔧 Configuration

Add to your environment or `docker-compose-graph.yml`:

```yaml
environment:
  - QDRANT_URL=http://qdrant:6333  # Or localhost:6333
  - GEMINI_API_KEY=your_api_key_here
```

## 📊 Response Examples

### H-Index Response
```json
{
  "status": "success",
  "author": "Ashish Vaswani",
  "h_index": 15,
  "total_papers": 23,
  "total_citations": 45123,
  "highly_cited_papers": [
    {
      "title": "Attention Is All You Need",
      "year": 2017,
      "citations": 42000
    }
  ]
}
```

### Semantic Search Response
```json
{
  "status": "success",
  "query": "attention mechanism",
  "results": [
    {
      "title": "Attention Is All You Need",
      "paper_id": "arxiv-1706.03762",
      "similarity_score": 0.92,
      "metadata": {
        "year": 2017,
        "authors": ["Ashish Vaswani", "..."]
      }
    }
  ],
  "count": 5
}
```

### Path Finding Response
```json
{
  "status": "success",
  "paper1": "Attention Is All You Need",
  "paper2": "GPT-3",
  "path": [
    {"paper": "Attention Is All You Need", "relationship": "CITES"},
    {"paper": "BERT", "relationship": "CITES"},
    {"paper": "GPT-2", "relationship": "CITES"},
    {"paper": "GPT-3"}
  ],
  "hops": 3
}
```

## 🎯 Next Steps

1. **Go CLI Integration**: Add `./archivist graph add paper.pdf` command
2. **TUI Graph Explorer**: Add menu with dashboard, search, and visualization
3. **Background Embedding Generation**: Auto-generate embeddings for new papers
4. **Similarity Edge Creation**: Auto-create SIMILAR_TO relationships

## 📝 Notes

- Embeddings are generated using Gemini's `text-embedding-004` model
- Similarity threshold of 0.85 is used for SIMILAR_TO edges (configurable)
- Citation context extraction uses simple keyword-based classification
- H-index calculation follows standard formula
- All new endpoints are async and non-blocking
