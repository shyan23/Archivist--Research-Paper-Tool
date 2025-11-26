# Advanced Search Features - User Guide

## Overview

The Archivist Search Engine now supports advanced search capabilities including:

1. **Semantic Search** - Find papers by meaning, not just keywords using AI embeddings
2. **Fuzzy Matching** - Handles typos and variations in search terms
3. **Hybrid Search** - Combines keyword, semantic, and fuzzy search for best results

## Search Modes

### 1. Keyword Search (Traditional)
Standard keyword-based search from external APIs (arXiv, OpenReview, ACL).

```bash
curl -X POST http://localhost:8000/api/search \
  -H "Content-Type: application/json" \
  -d '{
    "query": "vision transformers",
    "search_mode": "keyword",
    "max_results": 10
  }'
```

### 2. Semantic Search
Find papers by semantic similarity using vector embeddings.

**Benefits:**
- Finds papers with similar concepts even if keywords differ
- Understands context and meaning
- Discovers related work you might have missed

```bash
curl -X POST http://localhost:8000/api/search \
  -H "Content-Type: application/json" \
  -d '{
    "query": "attention mechanisms in neural networks",
    "search_mode": "semantic",
    "max_results": 10
  }'
```

**Example Use Cases:**
- Query: "neural networks for image classification"
  - Will find papers about CNNs, ResNet, VGG, etc. even without those keywords
- Query: "understanding language with transformers"
  - Will find BERT, GPT, T5 papers based on concept similarity

### 3. Fuzzy Search
Handles typos and abbreviations intelligently.

```bash
curl -X POST http://localhost:8000/api/search \
  -H "Content-Type: application/json" \
  -d '{
    "query": "convolutional nueral netwroks",
    "search_mode": "fuzzy",
    "fuzzy_threshold": 70,
    "max_results": 10
  }'
```

**Automatically expands abbreviations:**
- "CNN" → "Convolutional Neural Network"
- "BERT" → "Bidirectional Encoder Representations from Transformers"
- "GAN" → "Generative Adversarial Network"
- And many more...

### 4. Hybrid Search (Recommended)
Combines all methods for optimal results.

```bash
curl -X POST http://localhost:8000/api/search \
  -H "Content-Type: application/json" \
  -d '{
    "query": "transformer architectures",
    "search_mode": "hybrid",
    "semantic_weight": 0.7,
    "max_results": 20
  }'
```

**Parameters:**
- `semantic_weight`: Weight for semantic similarity (0-1, default: 0.7)
  - Higher = More emphasis on semantic meaning
  - Lower = More emphasis on keyword matching
- `fuzzy_threshold`: Minimum fuzzy match score (0-100, default: 70)

## Building the Vector Database

For semantic search to work, you need to index papers into the vector database.

### Method 1: Index from Search Results (Easy)

Search papers and automatically index them:

```bash
curl -X POST http://localhost:8000/api/index/from-search \
  -H "Content-Type: application/json" \
  -d '{
    "query": "deep learning",
    "max_results": 50
  }'
```

This will:
1. Search for papers using keyword search
2. Automatically index them in the vector database
3. Make them available for semantic search

### Method 2: Index Single Paper

```bash
curl -X POST http://localhost:8000/api/index/paper \
  -H "Content-Type: application/json" \
  -d '{
    "paper_id": "arxiv-2301.12345",
    "title": "Attention Is All You Need",
    "abstract": "The dominant sequence transduction models...",
    "authors": ["Ashish Vaswani", "Noam Shazeer"],
    "metadata": {
      "source": "arXiv",
      "venue": "NeurIPS",
      "published_at": "2017-06-12T00:00:00"
    }
  }'
```

### Method 3: Batch Index Multiple Papers

```bash
curl -X POST http://localhost:8000/api/index/batch \
  -H "Content-Type: application/json" \
  -d '[
    {
      "paper_id": "paper1",
      "title": "Paper 1 Title",
      "abstract": "Paper 1 Abstract...",
      "authors": ["Author 1"]
    },
    {
      "paper_id": "paper2",
      "title": "Paper 2 Title",
      "abstract": "Paper 2 Abstract...",
      "authors": ["Author 2"]
    }
  ]'
```

## Vector Store Management

### Check Vector Store Status

```bash
curl http://localhost:8000/api/vector-store/info
```

Returns:
```json
{
  "collection_name": "papers",
  "vectors_count": 1250,
  "embedding_dim": 384,
  "model_name": "sentence-transformers/all-MiniLM-L6-v2",
  "status": "ready"
}
```

### Clear Vector Store

```bash
curl -X DELETE http://localhost:8000/api/vector-store/clear
```

## Complete Workflow Example

### Step 1: Start the Service

```bash
cd services/search-engine
source venv/bin/activate
python run.py
```

### Step 2: Build Initial Index

Index papers on various topics to build your knowledge base:

```bash
# Index deep learning papers
curl -X POST http://localhost:8000/api/index/from-search \
  -H "Content-Type: application/json" \
  -d '{"query": "deep learning", "max_results": 50}'

# Index computer vision papers
curl -X POST http://localhost:8000/api/index/from-search \
  -H "Content-Type: application/json" \
  -d '{"query": "computer vision", "max_results": 50}'

# Index NLP papers
curl -X POST http://localhost:8000/api/index/from-search \
  -H "Content-Type: application/json" \
  -d '{"query": "natural language processing", "max_results": 50}'
```

### Step 3: Check Status

```bash
curl http://localhost:8000/api/vector-store/info
```

### Step 4: Search with Different Modes

**Keyword search:**
```bash
curl -X POST http://localhost:8000/api/search \
  -H "Content-Type: application/json" \
  -d '{"query": "attention mechanism", "search_mode": "keyword"}'
```

**Semantic search:**
```bash
curl -X POST http://localhost:8000/api/search \
  -H "Content-Type: application/json" \
  -d '{"query": "how do models focus on important parts", "search_mode": "semantic"}'
```

**Hybrid search (best):**
```bash
curl -X POST http://localhost:8000/api/search \
  -H "Content-Type: application/json" \
  -d '{"query": "transformers", "search_mode": "hybrid"}'
```

## Python Client Example

```python
import requests

BASE_URL = "http://localhost:8000"

# 1. Index papers
response = requests.post(
    f"{BASE_URL}/api/index/from-search",
    json={
        "query": "machine learning",
        "max_results": 30
    }
)
print(f"Indexed: {response.json()}")

# 2. Perform hybrid search
response = requests.post(
    f"{BASE_URL}/api/search",
    json={
        "query": "neural network optimization",
        "search_mode": "hybrid",
        "semantic_weight": 0.7,
        "max_results": 10
    }
)

results = response.json()
print(f"Found {results['total']} papers:")

for i, paper in enumerate(results['results'], 1):
    print(f"\n{i}. {paper['title']}")
    print(f"   Relevance: {paper.get('relevance_score', 0):.2f}")
    print(f"   Similarity: {paper.get('similarity_score', 0):.2f}")
    print(f"   Source: {paper['source']}")
```

## Understanding Scores

Results include multiple score types:

- **relevance_score** (0-1): Overall relevance in hybrid mode
- **similarity_score** (0-1): Semantic similarity from embeddings
- **fuzzy_score** (0-100): Fuzzy string matching score

Higher scores = better match

## Performance Tips

1. **Build a good index**: Index at least 50-100 papers per topic for best results
2. **Use hybrid mode**: Usually gives the best results
3. **Adjust semantic_weight**:
   - Use 0.8-0.9 for conceptual searches
   - Use 0.5-0.6 for specific keyword searches
4. **Lower fuzzy_threshold**: Set to 60-70 for broader matches

## Technical Details

### Embedding Model
- Model: `sentence-transformers/all-MiniLM-L6-v2`
- Dimension: 384
- Fast inference, good quality for research papers

### Vector Database
- Qdrant (local/embedded mode)
- Storage: `./data/qdrant`
- Distance metric: Cosine similarity

### Fuzzy Matching
- RapidFuzz library
- Token set ratio algorithm
- Abbreviation expansion for ML/AI terms

## Troubleshooting

### "No results from semantic search"
→ Index papers first using `/api/index/from-search`

### "Slow first search"
→ Normal - model loads on first use (~2-3 seconds)

### "Out of memory"
→ Reduce batch size or use smaller model

### "Results not relevant"
→ Adjust `semantic_weight` or build larger index

## API Reference

All endpoints are documented at: `http://localhost:8000/docs`

### Key Endpoints

| Endpoint | Method | Purpose |
|----------|--------|---------|
| `/api/search` | POST | Search papers (all modes) |
| `/api/index/paper` | POST | Index single paper |
| `/api/index/batch` | POST | Index multiple papers |
| `/api/index/from-search` | POST | Search & auto-index |
| `/api/vector-store/info` | GET | Check index status |
| `/api/vector-store/clear` | DELETE | Clear all indexed papers |

## Next Steps

1. **Build your index**: Start with broad topics relevant to your research
2. **Experiment with modes**: Try different search modes to see what works best
3. **Fine-tune weights**: Adjust `semantic_weight` based on your needs
4. **Integrate with Archivist**: Use with the main Archivist CLI for complete workflow

Happy searching! 🚀
