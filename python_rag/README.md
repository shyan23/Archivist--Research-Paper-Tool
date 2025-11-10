# Archivist Python RAG System

Advanced Retrieval Augmented Generation (RAG) system for chatting with research papers, built in Python with best-in-class components.

## Features

- **Multiple Embedding Providers**:
  - Sentence Transformers (local, fast)
  - Google Gemini (cloud, high quality)
  - OpenAI (cloud, state-of-the-art)

- **Vector Stores**:
  - ChromaDB (recommended, persistent)
  - FAISS (fast search)
  - Redis Stack (optional)

- **Smart Document Processing**:
  - LaTeX-aware chunking
  - Section detection
  - Overlapping chunks for context

- **RAG Chat Engine**:
  - Context-aware conversations
  - Citation tracking
  - Multi-paper support
  - Session management

- **Flexible Interfaces**:
  - CLI for standalone use
  - FastAPI server for Go integration
  - Python library for custom applications

## Quick Start

### 1. Installation

```bash
cd python_rag
pip install -r requirements.txt
```

### 2. Set API Key

```bash
export GEMINI_API_KEY="your_gemini_api_key_here"
```

### 3. Index Papers

```bash
# Index a single paper
python -m python_rag.cli index ../tex_files/paper_name.tex

# Index all papers in a directory
python -m python_rag.cli index ../tex_files/

# List indexed papers
python -m python_rag.cli list
```

### 4. Start Chatting

```bash
# Chat with all indexed papers
python -m python_rag.cli chat

# Chat with specific papers
python -m python_rag.cli chat --papers "Paper 1" "Paper 2"
```

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│                    FastAPI Server                        │
│                  (Go Integration)                        │
└────────────────────┬────────────────────────────────────┘
                     │
┌────────────────────▼────────────────────────────────────┐
│                  Chat Engine                             │
│          (Gemini + RAG Retrieval)                        │
└────────────────────┬────────────────────────────────────┘
                     │
         ┌───────────┴──────────┐
         │                      │
┌────────▼──────┐      ┌───────▼────────┐
│   Retriever   │      │  LLM (Gemini)  │
│  (RAG Logic)  │      │   Generation   │
└────────┬──────┘      └────────────────┘
         │
    ┌────┴─────┐
    │          │
┌───▼──┐  ┌───▼────┐
│ Vec  │  │ Embed  │
│Store │  │ Model  │
└──────┘  └────────┘
```

## Configuration

Edit `config.py` or use environment variables:

```python
# Embedding model
EMBEDDING_PROVIDER = "sentence-transformers"  # or "gemini", "openai"
EMBEDDING_MODEL = "all-MiniLM-L6-v2"         # Fast and good

# Vector store
VECTOR_STORE = "chromadb"                    # or "faiss"

# Chunking
CHUNK_SIZE = 1000                            # Characters
CHUNK_OVERLAP = 200                          # Characters

# Retrieval
TOP_K = 5                                    # Chunks to retrieve
SCORE_THRESHOLD = 0.3                        # Minimum similarity
```

## API Server (for Go Integration)

### Start Server

```bash
# Start on default port 8000
python -m python_rag.cli server

# Custom port
python -m python_rag.cli server --port 9000
```

### API Endpoints

**Index a paper:**
```bash
POST /index/paper
{
  "paper_title": "Attention Is All You Need",
  "latex_content": "\\section{Introduction}...",
  "pdf_path": "/path/to/paper.pdf"
}
```

**Create chat session:**
```bash
POST /chat/session
{
  "paper_titles": ["Paper 1", "Paper 2"]
}
```

**Send message:**
```bash
POST /chat/message
{
  "session_id": "session_123",
  "message": "What is the main contribution?"
}
```

**Retrieve context:**
```bash
POST /retrieve
{
  "query": "How does attention work?",
  "paper_titles": ["Attention Is All You Need"],
  "top_k": 5
}
```

## Go Integration

See `go_bridge.go` for a helper to call the Python API from Go:

```go
import "archivist/internal/python_rag"

// Start Python server
server := python_rag.NewPythonRAGServer()
server.Start()
defer server.Stop()

// Index a paper
server.IndexPaper("Paper Title", latexContent, pdfPath)

// Chat
sessionID := server.CreateChatSession([]string{"Paper Title"})
response := server.SendMessage(sessionID, "What is the methodology?")
```

## Python Library Usage

```python
from python_rag.config import RAGConfig
from python_rag.embeddings import create_embedding_provider
from python_rag.vector_store import create_vector_store
from python_rag.chunker import TextChunker
from python_rag.indexer import DocumentIndexer
from python_rag.retriever import Retriever
from python_rag.chat_engine import create_chat_engine

# Initialize
config = RAGConfig.from_env()

embedder = create_embedding_provider(
    provider="sentence-transformers",
    model_name="all-MiniLM-L6-v2"
)

vector_store = create_vector_store(
    provider="chromadb",
    persist_directory=".metadata/chromadb"
)

# Index a paper
chunker = TextChunker(chunk_size=1000, chunk_overlap=200)
indexer = DocumentIndexer(chunker, embedder, vector_store)
indexer.index_paper("Paper Title", latex_content)

# Chat
retriever = Retriever(vector_store, embedder)
chat_engine = create_chat_engine(retriever, gemini_api_key="...")

session = chat_engine.create_session(["Paper Title"])
response = chat_engine.chat(session.session_id, "What is this paper about?")

print(response.content)
print(f"Sources: {response.citations}")
```

## Best Practices

### Embedding Model Selection

- **For fast, local processing**: `all-MiniLM-L6-v2` (384 dims)
- **For best quality**: `all-mpnet-base-v2` (768 dims)
- **For Q&A tasks**: `multi-qa-mpnet-base-dot-v1` (768 dims)
- **For cloud with API**: Gemini `text-embedding-004` (768 dims)

### Vector Store Selection

- **Small datasets (<10k docs)**: ChromaDB or FAISS
- **Large datasets**: FAISS for speed
- **Distributed setup**: Redis Stack

### Chunking Parameters

- **Dense papers (math-heavy)**: Smaller chunks (800-1000 chars)
- **Narrative papers**: Larger chunks (1500-2000 chars)
- **Overlap**: 10-20% of chunk size

## Troubleshooting

### ChromaDB Issues
```bash
# Clear and rebuild
rm -rf .metadata/chromadb
python -m python_rag.cli index ../tex_files/ --force
```

### Embedding Model Download
```bash
# Pre-download model
python -c "from sentence_transformers import SentenceTransformer; SentenceTransformer('all-MiniLM-L6-v2')"
```

### API Connection
```bash
# Test server
curl http://localhost:8000/health

# Test indexing
curl -X POST http://localhost:8000/index/paper \
  -H "Content-Type: application/json" \
  -d '{"paper_title": "test", "latex_content": "..."}'
```

## Performance

- **Indexing**: ~500 papers/hour (with Sentence Transformers)
- **Retrieval**: <100ms per query
- **Chat response**: 2-5 seconds (depends on LLM)

## License

Part of the Archivist project.
