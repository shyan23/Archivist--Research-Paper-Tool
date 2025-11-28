# Archivist Search Engine - Quick Start Guide

## Overview

The Archivist now includes a powerful search engine to find and download research papers from multiple academic sources:

- **arXiv**: AI/ML papers, Computer Vision, NLP
- **OpenReview**: ICLR, NeurIPS conference papers
- **ACL Anthology**: EMNLP, ACL, and other NLP conferences

## Architecture

```
┌─────────────────────────────────────────────────────┐
│        Archivist CLI (Go)                            │
│        ./archivist search "query"                    │
└───────────────────┬─────────────────────────────────┘
                    │ HTTP REST API
┌───────────────────▼─────────────────────────────────┐
│   Search Microservice (Python/FastAPI)              │
│   - Modular provider system                          │
│   - Async/parallel searches                          │
│   - Running on http://localhost:8000                 │
└─────┬──────────┬──────────┬─────────────────────────┘
      │          │          │
   ┌──▼──┐   ┌──▼──┐   ┌───▼───┐
   │arXiv│   │Open │   │  ACL  │
   │     │   │Review│  │Anthology│
   └─────┘   └─────┘   └───────┘
```

## Setup (One-time)

### 1. Install Python dependencies

```bash
cd services/search-engine

# Create virtual environment
python3 -m venv venv

# Activate virtual environment
source venv/bin/activate  # On macOS/Linux
# OR
venv\Scripts\activate     # On Windows

# Install dependencies
pip install -r requirements.txt
```

### 2. Rebuild Archivist CLI

```bash
cd /home/shyan/Desktop/Code/Archivist
go build -o archivist cmd/main/main.go
```

## Usage

### Step 1: Start the Search Microservice

In a **separate terminal**:

```bash
cd /home/shyan/Desktop/Code/Archivist/services/search-engine
source venv/bin/activate
python run.py
```

You should see:
```
INFO:     Started server process
INFO:     Uvicorn running on http://0.0.0.0:8000
```

The service is now running with:
- API Documentation: http://localhost:8000/docs
- Alternative Docs: http://localhost:8000/redoc

### Step 2: Search for Papers

In your **main terminal**:

```bash
# Basic search
./archivist search "unified multimodal LLM architecture"

# Limit results
./archivist search "transformer architecture" --max-results 10

# Search specific sources only
./archivist search "vision transformers" --sources arxiv,openreview

# Search and download interactively
./archivist search "attention mechanism" --download
```

## Command Options

```bash
./archivist search [query] [flags]

Flags:
  -n, --max-results int      Maximum number of results (default 20)
  -s, --sources strings      Filter by sources: arxiv, openreview, acl
  -d, --download             Download selected papers to lib/
      --service-url string   Search service URL (default "http://localhost:8000")
  -h, --help                 Help for search
```

## Examples

### Example 1: Find recent papers on a topic

```bash
./archivist search "diffusion models" --max-results 15
```

**Output:**
```
🔍 Searching for: diffusion models
   Sources: all (arXiv, OpenReview, ACL)
   Max results: 15

✓ Found 15 papers
  Sources searched: [arXiv OpenReview ACL]

[1] Denoising Diffusion Probabilistic Models
    Source: arXiv | Venue: arXiv | Published: 2024-03-15
    Authors: Jonathan Ho, Ajay Jain, Pieter Abbeel
    We present high quality image synthesis results using diffusion probabilistic models...
    PDF: https://arxiv.org/pdf/2006.11239.pdf
    Source: https://arxiv.org/abs/2006.11239
    Categories: [cs.LG, cs.CV]

[2] ...
```

### Example 2: Search only arXiv

```bash
./archivist search "graph neural networks" --sources arxiv --max-results 10
```

### Example 3: Search and download papers

```bash
./archivist search "BERT language model" --download
```

This will:
1. Show search results
2. Present an interactive menu to select papers
3. Download selected PDFs to your `lib/` directory
4. Papers are then ready to be processed with `./archivist process`!

## Advanced Usage

### Using the Python API directly

You can also use the microservice directly via HTTP:

```bash
# Search
curl -X POST http://localhost:8000/api/search \
  -H "Content-Type: application/json" \
  -d '{
    "query": "attention mechanism",
    "max_results": 10,
    "sources": ["arxiv"]
  }'

# Check service health
curl http://localhost:8000/health
```

### Interactive API Documentation

Visit http://localhost:8000/docs to:
- See all available endpoints
- Test the API interactively
- View request/response schemas

## Workflow: Search → Download → Process

Complete workflow to analyze papers:

```bash
# Terminal 1: Start search service
cd services/search-engine
source venv/bin/activate
python run.py

# Terminal 2: Search and download papers
cd /home/shyan/Desktop/Code/Archivist

# 1. Search and download papers
./archivist search "vision transformers" --download --max-results 5

# 2. Process downloaded papers
./archivist process lib/

# 3. Chat with processed papers
./archivist chat
```

## Troubleshooting

### "search service is not running"

**Problem:** The Python microservice isn't running.

**Solution:**
```bash
cd services/search-engine
source venv/bin/activate
python run.py
```

### "Failed to download PDF"

**Problem:** Some sources may have rate limiting or require authentication.

**Solutions:**
- Try again later
- Download fewer papers at once
- Some conference papers may not be publicly available yet

### Port 8000 already in use

**Problem:** Another service is using port 8000.

**Solution:** Change the port:
```bash
# In services/search-engine/run.py, change:
uvicorn.run(..., port=9000)  # Use different port

# Then update the Archivist command:
./archivist search "query" --service-url http://localhost:9000
```

## Modular Design Benefits

The search engine is designed to be **fully modular**:

1. **Standalone Service**: Can be used independently from Archivist
2. **Add New Providers**: Easy to add support for new paper sources
3. **Reusable**: Can integrate into other projects via REST API
4. **Language Agnostic**: Any language that speaks HTTP can use it

### Adding a New Provider

See `services/search-engine/README.md` for detailed instructions on adding new search sources.

## API Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/` | GET | Service info and health |
| `/health` | GET | Health check |
| `/api/search` | POST | Search papers |
| `/api/download` | POST | Download a PDF |
| `/api/providers` | GET | List available providers |
| `/docs` | GET | Interactive API docs |

## Next Steps

After downloading papers:

1. **Process them**: `./archivist process lib/your_paper.pdf`
2. **Build knowledge graph**: Automatic with graph feature enabled
3. **Chat with papers**: `./archivist chat`
4. **Export reports**: LaTeX reports are generated automatically

Happy researching! 🎓
