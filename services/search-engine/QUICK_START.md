# Archivist Search Engine - Quick Start

## Overview

Simple and powerful arXiv paper search with **intelligent fuzzy matching**.

### Features
- ✅ **arXiv Search** - Search millions of research papers
- ✅ **Fuzzy Matching** - Handles typos and variations
- ✅ **Abbreviation Expansion** - Automatically expands CNN, BERT, GAN, etc.
- ✅ **Relevance Ranking** - Results sorted by match quality

## Installation

```bash
cd services/search-engine

# Create virtual environment
python3 -m venv venv
source venv/bin/activate

# Install dependencies
pip install -r requirements.txt
```

## Usage

### 1. Start the Service

```bash
source venv/bin/activate
python run.py
```

Service runs on: `http://localhost:8000`

### 2. Search for Papers

**Example:**
```bash
curl -X POST http://localhost:8000/api/search \
  -H "Content-Type: application/json" \
  -d '{"query": "unified multimodal llm", "max_results": 5}'
```

**Response:**
```json
{
  "query": "unified multimodal llm",
  "total": 3,
  "results": [
    {
      "title": "PixelBytes: Catching Unified Embedding for Multimodal Generation",
      "authors": ["Fabien Furfaro"],
      "abstract": "This report introduces PixelBytes...",
      "published_at": "2024-09-03T06:02:02",
      "pdf_url": "http://arxiv.org/pdf/2409.15512.pdf",
      "source_url": "http://arxiv.org/abs/2409.15512v2",
      "relevance_score": 0.9,
      "fuzzy_score": 90.0
    }
  ]
}
```

## Search Parameters

```json
{
  "query": "search terms",           // Required
  "max_results": 20,                 // Optional (default: 20)
  "fuzzy_threshold": 70,             // Optional (default: 70, range: 0-100)
  "start_date": "2023-01-01",       // Optional
  "end_date": "2025-12-31"           // Optional
}
```

## Fuzzy Matching Examples

### Handles Typos
```bash
# Works even with typos!
curl -X POST http://localhost:8000/api/search \
  -d '{"query": "convolutional nueral netwroks"}'  # typos in "neural networks"
```

### Abbreviation Expansion
The system automatically expands:
- `CNN` → "convolutional neural network"
- `BERT` → "bidirectional encoder representations from transformers"
- `GAN` → "generative adversarial network"
- `GPT` → "generative pre-trained transformer"
- `LSTM` → "long short-term memory"
- And many more...

```bash
# Search with abbreviation
curl -X POST http://localhost:8000/api/search \
  -d '{"query": "gnn for graph classification"}'
# Automatically searches for "graph neural networks"
```

## Understanding Scores

Each result includes:
- **relevance_score** (0-1): Overall match quality
- **fuzzy_score** (0-100): Fuzzy string match percentage
  - 90-100: Excellent match
  - 70-89: Good match
  - Below 70: Filtered out

## API Documentation

Interactive API docs: `http://localhost:8000/docs`

## Common Queries

```bash
# Search for recent papers
curl -X POST http://localhost:8000/api/search \
  -d '{"query": "transformer architecture", "start_date": "2024-01-01"}'

# High precision search
curl -X POST http://localhost:8000/api/search \
  -d '{"query": "attention mechanisms", "fuzzy_threshold": 85}'

# Get more results
curl -X POST http://localhost:8000/api/search \
  -d '{"query": "deep learning", "max_results": 50}'
```

## Integration

The search service is designed to work with the main Archivist CLI:

```bash
# From the main Archivist directory
./archivist search "unified multimodal llm"
```

## Troubleshooting

**Service won't start?**
```bash
# Check if port 8000 is in use
lsof -i :8000

# Use different port
# Edit run.py and change port=8000 to port=9000
```

**No results?**
- Try lowering `fuzzy_threshold`
- Use broader search terms
- Check spelling

**Slow searches?**
- First search is slower (API connection)
- Subsequent searches are faster

## What's Next?

This is the basic implementation. Future features:
- Semantic search with Gemini embeddings
- OpenReview integration
- ACL Anthology integration
- Vector database for offline search

## Notes

- Currently searches arXiv only
- Requires internet connection
- No API key needed for arXiv
- Results are cached for performance

Happy searching! 🔍
