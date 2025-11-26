# Archivist Search Engine Microservice

A modular, standalone Python microservice for searching and downloading academic papers from multiple sources.

## Features

- **Multi-source search**: arXiv, OpenReview (ICLR, NeurIPS), ACL Anthology (EMNLP, ACL, etc.)
- **RESTful API**: FastAPI-based with automatic documentation
- **Async/Parallel**: Concurrent searches across all sources
- **Modular design**: Easy to add new providers
- **Type-safe**: Full Pydantic models for request/response validation

## Architecture

```
services/search-engine/
├── app/
│   ├── __init__.py
│   ├── main.py              # FastAPI application
│   ├── models.py            # Pydantic data models
│   └── providers/
│       ├── __init__.py
│       ├── base.py          # Base provider interface
│       ├── arxiv_provider.py
│       ├── openreview_provider.py
│       └── acl_provider.py
├── requirements.txt
├── run.py                   # Startup script
└── README.md
```

## Installation & Running

### Option 1: Docker (Recommended)

The easiest way to run the search engine is using Docker:

#### Quick Start

```bash
cd services/search-engine

# Copy the environment template
cp .env.example .env

# Edit .env and add your Gemini API key
# GEMINI_API_KEY=your_key_here

# Build and run with docker-compose
docker-compose up -d

# Check service health
curl http://localhost:8000/health
```

#### Docker Commands

```bash
# Start the service
docker-compose up -d

# View logs
docker-compose logs -f

# Stop the service
docker-compose down

# Rebuild after code changes
docker-compose up -d --build

# Stop and remove volumes (clears vector store data)
docker-compose down -v
```

The service will be available at:
- API: `http://localhost:8000`
- Interactive docs: `http://localhost:8000/docs`
- ReDoc: `http://localhost:8000/redoc`

#### Docker Features

- **Automatic restart**: Service restarts on failure
- **Health checks**: Built-in health monitoring
- **Data persistence**: Vector store data persisted in `./data` directory
- **Easy updates**: Simple rebuild and restart process

### Option 2: Manual Installation (Development)

For local development without Docker:

#### 1. Create a virtual environment

```bash
cd services/search-engine
python3 -m venv venv
source venv/bin/activate  # On Windows: venv\Scripts\activate
```

#### 2. Install dependencies

```bash
pip install -r requirements.txt
```

#### 3. Set environment variables

```bash
# Set your Gemini API key
export GEMINI_API_KEY=your_key_here
```

#### 4. Run the service

**Development mode (with auto-reload):**

```bash
python run.py
```

**Production mode:**

```bash
uvicorn app.main:app --host 0.0.0.0 --port 8000
```

The service will be available at:
- API: `http://localhost:8000`
- Interactive docs: `http://localhost:8000/docs`
- ReDoc: `http://localhost:8000/redoc`

## API Endpoints

### 1. Search Papers

**POST** `/api/search`

Search for papers across multiple sources.

**Request body:**
```json
{
  "query": "unified multimodal LLM architecture",
  "max_results": 20,
  "sources": ["arxiv", "openreview", "acl"],
  "start_date": "2023-01-01T00:00:00",
  "end_date": "2025-12-31T23:59:59"
}
```

**Response:**
```json
{
  "query": "unified multimodal LLM architecture",
  "total": 15,
  "sources_searched": ["arXiv", "OpenReview", "ACL"],
  "results": [
    {
      "title": "Paper Title",
      "authors": ["Author 1", "Author 2"],
      "abstract": "Paper abstract...",
      "published_at": "2024-03-15T00:00:00",
      "pdf_url": "https://arxiv.org/pdf/2403.12345.pdf",
      "source_url": "https://arxiv.org/abs/2403.12345",
      "source": "arXiv",
      "venue": "arXiv",
      "id": "2403.12345",
      "categories": ["cs.CV", "cs.AI"]
    }
  ]
}
```

### 2. Download Paper

**POST** `/api/download`

Download a paper PDF.

**Request body:**
```json
{
  "pdf_url": "https://arxiv.org/pdf/2403.12345.pdf",
  "filename": "my_paper.pdf"
}
```

**Response:**
```json
{
  "success": true,
  "filename": "my_paper.pdf",
  "size_bytes": 1234567,
  "message": "PDF downloaded successfully to /tmp/archivist_downloads/my_paper.pdf"
}
```

### 3. List Providers

**GET** `/api/providers`

List all available search providers.

### 4. Health Check

**GET** `/health`

Check service health status.

## Usage Examples

### Using cURL

```bash
# Search for papers
curl -X POST http://localhost:8000/api/search \
  -H "Content-Type: application/json" \
  -d '{
    "query": "transformer architecture",
    "max_results": 10,
    "sources": ["arxiv"]
  }'

# Download a paper
curl -X POST http://localhost:8000/api/download \
  -H "Content-Type: application/json" \
  -d '{
    "pdf_url": "https://arxiv.org/pdf/1706.03762.pdf",
    "filename": "attention_is_all_you_need.pdf"
  }'
```

### Using Python

```python
import requests

# Search
response = requests.post(
    "http://localhost:8000/api/search",
    json={
        "query": "vision transformers",
        "max_results": 20,
        "sources": ["arxiv", "openreview"]
    }
)
results = response.json()

# Download
for paper in results["results"][:5]:
    response = requests.post(
        "http://localhost:8000/api/download",
        json={
            "pdf_url": paper["pdf_url"],
            "filename": f"{paper['id']}.pdf"
        }
    )
    print(f"Downloaded: {response.json()['filename']}")
```

### Using Go (from Archivist)

```go
import (
    "bytes"
    "encoding/json"
    "net/http"
)

type SearchQuery struct {
    Query      string   `json:"query"`
    MaxResults int      `json:"max_results"`
    Sources    []string `json:"sources"`
}

func searchPapers(query string) (*SearchResponse, error) {
    searchQuery := SearchQuery{
        Query:      query,
        MaxResults: 20,
        Sources:    []string{"arxiv", "openreview", "acl"},
    }

    body, _ := json.Marshal(searchQuery)
    resp, err := http.Post(
        "http://localhost:8000/api/search",
        "application/json",
        bytes.NewBuffer(body),
    )
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    var result SearchResponse
    json.NewDecoder(resp.Body).Decode(&result)
    return &result, nil
}
```

## Adding a New Provider

1. Create a new provider in `app/providers/`:

```python
from .base import SearchProvider
from ..models import SearchQuery, SearchResult

class NewProvider(SearchProvider):
    def name(self) -> str:
        return "NewSource"

    async def search(self, query: SearchQuery) -> List[SearchResult]:
        # Implement search logic
        pass

    async def download_pdf(self, url: str, output_path: str) -> bool:
        # Implement download logic
        pass
```

2. Register in `app/main.py`:

```python
from .providers import NewProvider

providers = {
    "arxiv": ArxivProvider(),
    "openreview": OpenReviewProvider(),
    "acl": ACLProvider(),
    "newsource": NewProvider()  # Add here
}
```

## Configuration

The service runs on port 8000 by default. To change:

```bash
# In run.py or command line
uvicorn app.main:app --host 0.0.0.0 --port 9000
```

## Testing

```bash
# Install test dependencies
pip install pytest pytest-asyncio httpx

# Run tests
pytest tests/
```

## Deployment

### Docker Deployment

The service includes production-ready Docker configuration:

**Dockerfile highlights:**
- Multi-stage build for smaller image size
- Python 3.10 slim base image
- Health check configuration
- Data persistence for vector store
- Non-root user execution

**Docker Compose features:**
- Automatic restarts on failure
- Volume mounting for data persistence
- Environment variable configuration
- Network isolation
- Health monitoring

To deploy in production:

```bash
# Use production environment file
cp .env.example .env.production
# Edit with production settings

# Run with production config
docker-compose -f docker-compose.yml --env-file .env.production up -d

# Monitor logs
docker-compose logs -f --tail=100
```

### systemd service (Linux)

Create `/etc/systemd/system/archivist-search.service`:

```ini
[Unit]
Description=Archivist Search Engine
After=network.target

[Service]
Type=simple
User=your-user
WorkingDirectory=/path/to/services/search-engine
ExecStart=/path/to/venv/bin/python run.py
Restart=always

[Install]
WantedBy=multi-user.target
```

## License

Part of the Archivist project.
