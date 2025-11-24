# Archivist RAG Chatbot - Testing Guide

Simple standalone RAG chatbot for research papers with PDF support.

## Quick Start

### 1. Setup

```bash
cd /home/shyan/Desktop/Code/Archivist/python_rag

# Set your Gemini API key
export GEMINI_API_KEY="your_gemini_api_key_here"
```

### 2. Start the Server

```bash
# Option A: Using the run script
./run_server.sh

# Option B: Direct Python (on custom port)
PORT=8001 python simple_server.py
```

The server will start on http://localhost:8000 (or your custom port).

### 3. Open API Documentation

Open your browser and go to:
- **Swagger UI**: http://localhost:8001/docs
- **ReDoc**: http://localhost:8001/redoc

This is where you can test all API endpoints!

## Testing with FastAPI Docs

### Step 1: Index a PDF Paper

1. Open http://localhost:8001/docs
2. Click on `POST /index/pdf`
3. Click **"Try it out"**
4. Enter the request body:

```json
{
  "pdf_path": "/home/shyan/Desktop/Code/Archivist/lib/NIPS-2017-attention-is-all-you-need-Paper.pdf"
}
```

5. Click **"Execute"**
6. You should see a success response with the number of chunks indexed

**Available PDFs in `/lib`**:
- `NIPS-2017-attention-is-all-you-need-Paper.pdf`
- `2209.03561v2.pdf`
- `Focus.pdf`
- `2510.08492v1.pdf`
- `2510.18234v1.pdf`
- And more...

### Step 2: Create a Chat Session

1. Click on `POST /chat/session`
2. Click **"Try it out"**
3. Enter:

```json
{
  "paper_titles": []
}
```

(Empty array = chat with all indexed papers)

4. Click **"Execute"**
5. **Copy the `session_id`** from the response!

### Step 3: Chat with the Papers

1. Click on `POST /chat/message`
2. Click **"Try it out"**
3. Enter:

```json
{
  "session_id": "paste_your_session_id_here",
  "message": "What is the attention mechanism?"
}
```

4. Click **"Execute"**
5. See the AI response with citations!

### Step 4: Try More Queries

Keep using the same `session_id` to continue the conversation:

```json
{
  "session_id": "your_session_id",
  "message": "How does multi-head attention work?"
}
```

```json
{
  "session_id": "your_session_id",
  "message": "What are the key contributions of this paper?"
}
```

## Other Useful Endpoints

### Check Server Health

```
GET /health
```

Returns the number of indexed papers.

### List Indexed Papers

```
GET /index/papers
```

See all papers currently in the system.

### Get System Info

```
GET /info
```

See configuration, available PDFs in `/lib`, and more.

### View Chat History

```
GET /chat/session/{session_id}
```

See all messages in a conversation.

## Configuration

The system uses:
- **Vector Store**: FAISS (CPU-friendly, no server needed)
- **Embeddings**: Sentence Transformers (`all-MiniLM-L6-v2`)
- **LLM**: Google Gemini 2.0 Flash
- **PDF Extraction**: PyPDF2

All configured in `config.py`.

## For Go Integration

The FastAPI server provides a simple HTTP API that the Go chatbot can call:

### 1. Index a Paper (Go → Python)
```http
POST /index/pdf
Content-Type: application/json

{
  "pdf_path": "/path/to/paper.pdf"
}
```

### 2. Create Chat Session (Go → Python)
```http
POST /chat/session
Content-Type: application/json

{
  "paper_titles": ["Paper Title 1", "Paper Title 2"]
}
```

Response:
```json
{
  "session_id": "generated-uuid",
  "paper_titles": [...],
  "message": "Session created"
}
```

### 3. Send Message (Go → Python)
```http
POST /chat/message
Content-Type: application/json

{
  "session_id": "session-uuid",
  "message": "What is this paper about?"
}
```

Response:
```json
{
  "role": "assistant",
  "content": "This paper introduces...",
  "citations": ["Paper Title 1"],
  "timestamp": 1234567890.123
}
```

## Troubleshooting

### Port Already in Use
```bash
# Use a different port
PORT=8002 python simple_server.py
```

### API Key Not Set
```bash
export GEMINI_API_KEY="your_key_here"
```

### Missing Dependencies
```bash
pip install -r requirements_minimal.txt
```

### PDF Not Found
Make sure the PDF path is absolute and exists:
```bash
ls /home/shyan/Desktop/Code/Archivist/lib/*.pdf
```

## Example Workflow

```bash
# 1. Start server
PORT=8001 python simple_server.py

# 2. In another terminal, test with curl:

# Index a paper
curl -X POST "http://localhost:8001/index/pdf" \
  -H "Content-Type: application/json" \
  -d '{"pdf_path": "/home/shyan/Desktop/Code/Archivist/lib/NIPS-2017-attention-is-all-you-need-Paper.pdf"}'

# Create session
SESSION=$(curl -s -X POST "http://localhost:8001/chat/session" \
  -H "Content-Type: application/json" \
  -d '{"paper_titles": []}' | jq -r '.session_id')

# Chat
curl -X POST "http://localhost:8001/chat/message" \
  -H "Content-Type: application/json" \
  -d "{\"session_id\": \"$SESSION\", \"message\": \"What is the attention mechanism?\"}"
```

## Next Steps

- Index more papers from `/lib`
- Try different questions
- Test multi-paper conversations
- Integrate with Go chatbot backend

---

**Server Status**:
- Health: http://localhost:8001/health
- Docs: http://localhost:8001/docs
