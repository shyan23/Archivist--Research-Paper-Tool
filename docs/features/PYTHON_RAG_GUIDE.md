# Python RAG Chatbot Integration Guide

This guide explains how to use the new Python-based RAG (Retrieval Augmented Generation) chatbot system with Archivist.

## Overview

The Python RAG system provides advanced document indexing and chat capabilities with best-in-class embeddings and vector databases:

- **Embeddings**: Sentence Transformers, Gemini, or OpenAI
- **Vector Store**: ChromaDB (recommended), FAISS, or Redis
- **LLM**: Google Gemini (configurable)
- **Features**: Section-aware chunking, multi-paper chat, citations

## Quick Start

### 1. Install Python Dependencies

```bash
# Run the setup script (recommended)
./scripts/setup_python_rag.sh

# Or manually
cd python_rag
pip install -r requirements.txt
```

### 2. Set API Key

```bash
export GEMINI_API_KEY="your_gemini_api_key_here"
```

Get a free API key at: https://aistudio.google.com/app/apikey

### 3. Index Your Papers

The Python RAG system needs to index papers before you can chat with them:

```bash
# Index all papers from tex_files directory
python -m python_rag.cli index tex_files/

# Or index a single paper
python -m python_rag.cli index tex_files/my_paper.tex

# List indexed papers
python -m python_rag.cli list
```

### 4. Start Chatting

**Option A: Standalone CLI**

```bash
# Chat with all indexed papers
python -m python_rag.cli chat

# Chat with specific papers
python -m python_rag.cli chat --papers "Paper 1" "Paper 2"
```

**Option B: With Go TUI (via API server)**

```bash
# Terminal 1: Start Python API server
python -m python_rag.cli server --port 8000

# Terminal 2: Run Archivist TUI
./archivist

# Then select "Chat with Paper" option
```

## Architecture

### Components

1. **Embeddings** (`embeddings.py`):
   - Converts text to vector representations
   - Supports multiple providers (Sentence Transformers, Gemini, OpenAI)
   - Default: `all-MiniLM-L6-v2` (fast, good quality, 384 dims)

2. **Vector Store** (`vector_store.py`):
   - Stores and searches document embeddings
   - Supports ChromaDB (persistent) and FAISS (fast)
   - Default: ChromaDB in `.metadata/chromadb`

3. **Chunker** (`chunker.py`):
   - Splits documents into manageable chunks
   - LaTeX section-aware
   - Overlapping chunks for context continuity

4. **Indexer** (`indexer.py`):
   - Processes papers and stores in vector database
   - Tracks indexed papers
   - Supports re-indexing

5. **Retriever** (`retriever.py`):
   - RAG retrieval logic
   - Finds relevant chunks for queries
   - Supports filtering by paper/section

6. **Chat Engine** (`chat_engine.py`):
   - Manages conversation sessions
   - Combines retrieval + LLM generation
   - Citation tracking

7. **API Server** (`api_server.py`):
   - FastAPI REST API
   - Used by Go TUI for integration
   - Endpoints for indexing, chat, retrieval

### Data Flow

```
User Query
    │
    ├─> Retriever
    │   ├─> Generate Query Embedding
    │   ├─> Search Vector Store
    │   └─> Rank & Filter Results
    │
    ├─> Build Prompt (Query + Context + History)
    │
    ├─> LLM (Gemini)
    │   └─> Generate Response
    │
    └─> Return Response + Citations
```

## Configuration

### Embedding Model Selection

Edit `python_rag/config.py`:

```python
class EmbeddingConfig:
    provider: str = "sentence-transformers"  # or "gemini", "openai"
    model_name: str = "all-MiniLM-L6-v2"    # or "all-mpnet-base-v2"
```

**Recommendations:**
- **Fast & Local**: `all-MiniLM-L6-v2` (384 dims)
- **Better Quality**: `all-mpnet-base-v2` (768 dims)
- **Best for Q&A**: `multi-qa-mpnet-base-dot-v1` (768 dims)
- **Cloud**: Gemini `text-embedding-004` (768 dims)

### Vector Store Selection

```python
class VectorStoreConfig:
    provider: str = "chromadb"  # or "faiss"
    persist_directory: str = ".metadata/chromadb"
```

**Recommendations:**
- **Small datasets (<10k docs)**: ChromaDB
- **Fast search**: FAISS
- **Persistent**: ChromaDB

### Chunking Parameters

```python
class ChunkingConfig:
    chunk_size: int = 1000        # Characters
    chunk_overlap: int = 200      # Characters
    respect_sections: bool = True # LaTeX section awareness
```

## Go Integration

### Using Python RAG from Go

The `internal/python_rag` package provides Go bindings:

```go
import "archivist/internal/python_rag"

// Start Python API server
server := python_rag.NewServer(8000)
if err := server.Start(); err != nil {
    log.Fatal(err)
}
defer server.Stop()

// Get client
client := server.Client()

// Index a paper
indexReq := &python_rag.IndexPaperRequest{
    PaperTitle:   "My Paper",
    LatexContent: latexContent,
    PDFPath:      pdfPath,
}
resp, err := client.IndexPaper(ctx, indexReq)

// Create chat session
sessionResp, err := client.CreateChatSession(ctx, []string{"My Paper"})

// Send message
chatResp, err := client.SendChatMessage(ctx, sessionResp.SessionID, "What is this paper about?")

fmt.Println(chatResp.Content)
fmt.Println("Sources:", chatResp.Citations)
```

### Integration with Existing TUI

The TUI already has chat functionality. To use Python RAG instead of the Go implementation:

1. Start Python API server
2. Modify `internal/tui/chat.go` to use `python_rag.Client`
3. Or run both and let user choose

## API Endpoints

### Health & Info
- `GET /health` - Health check
- `GET /system/info` - System information

### Indexing
- `POST /index/paper` - Index a paper
- `GET /index/papers` - List indexed papers
- `GET /index/paper/{title}` - Get paper info
- `DELETE /index/paper/{title}` - Delete paper index

### Chat
- `POST /chat/session` - Create chat session
- `POST /chat/message` - Send message
- `GET /chat/session/{id}` - Get session
- `DELETE /chat/session/{id}` - Delete session
- `GET /chat/sessions` - List sessions

### Retrieval
- `POST /retrieve` - Retrieve context

Example:

```bash
# Create session
curl -X POST http://localhost:8000/chat/session \
  -H "Content-Type: application/json" \
  -d '{"paper_titles": ["My Paper"]}'

# Send message
curl -X POST http://localhost:8000/chat/message \
  -H "Content-Type: application/json" \
  -d '{
    "session_id": "session_123",
    "message": "What is the main contribution?"
  }'
```

## Troubleshooting

### Python dependencies not installed

```bash
cd python_rag
pip install -r requirements.txt
```

### ChromaDB issues

```bash
# Clear and rebuild
rm -rf .metadata/chromadb
python -m python_rag.cli index tex_files/ --force
```

### API server won't start

```bash
# Check if port is in use
lsof -i :8000

# Use different port
python -m python_rag.cli server --port 9000
```

### Embedding model download fails

```bash
# Pre-download manually
python -c "from sentence_transformers import SentenceTransformer; SentenceTransformer('all-MiniLM-L6-v2')"
```

### Out of memory

- Reduce `chunk_size` in config
- Use smaller embedding model
- Process papers in batches

## Performance Tips

1. **Indexing Speed**:
   - Use Sentence Transformers (local) instead of API calls
   - Smaller embedding models are faster
   - Batch processing is more efficient

2. **Search Speed**:
   - FAISS is faster than ChromaDB for large datasets
   - Reduce `top_k` if you don't need many results
   - Lower `chunk_size` means more chunks but faster retrieval

3. **Chat Response Time**:
   - Reduce `max_context_length` to send less to LLM
   - Use faster models (e.g., `gemini-2.0-flash` vs `gemini-pro`)
   - Lower `top_k` to retrieve fewer chunks

## Comparison: Python RAG vs Go RAG

| Feature | Python RAG | Go RAG |
|---------|-----------|--------|
| Embedding Models | Multiple (ST, Gemini, OpenAI) | Gemini only |
| Vector Stores | ChromaDB, FAISS | Redis Stack, FAISS |
| Performance | Moderate (Python) | Fast (Go) |
| ML Ecosystem | Excellent | Limited |
| Ease of Use | Very Easy | Moderate |
| Dependencies | Many (pip) | Few (Go modules) |
| Best For | Experimentation, ML workflows | Production, performance |

## Next Steps

- Read `python_rag/README.md` for detailed API documentation
- Check example scripts in `examples/`
- Experiment with different embedding models
- Try different chunking strategies
- Integrate with your existing workflow

## Support

For issues, check:
- Python RAG logs: Look for errors in terminal
- API server logs: Check uvicorn output
- Go logs: Check Archivist logs

Common issues are documented in `python_rag/README.md`.
