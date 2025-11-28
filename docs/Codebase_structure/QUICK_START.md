# ⚡ Archivist Knowledge Graph - Quick Start

## 5-Minute Setup

### 1. Start Services (30 seconds)

```bash
cd /home/shyan/Desktop/Code/Archivist
./scripts/setup-graph.sh
```

### 2. Configure API Key (1 minute)

Edit `config/config.yaml`:

```yaml
gemini:
  model: "models/gemini-2.0-flash-exp"
  # Add your API key here ↓
  api_key: "YOUR_GEMINI_API_KEY"

qdrant:
  host: "localhost"
  port: 6333
  collection_name: "archivist_papers"

graph:
  enabled: true
  neo4j:
    uri: "bolt://localhost:7687"
    username: "neo4j"
    password: "password"
```

Get API key: https://makersuite.google.com/app/apikey

### 3. Install Go Dependencies (1 minute)

```bash
go get github.com/qdrant/go-client
go get github.com/neo4j/neo4j-go-driver/v5
go mod tidy
```

### 4. Build & Test (2 minutes)

```bash
# Build
go build -o archivist cmd/main/main.go

# Test with one paper
./archivist process lib/your_paper.pdf --with-graph

# Check graph stats
./archivist graph stats

# Search
./archivist search "attention mechanisms"
```

---

## Common Commands

```bash
# Process papers
./archivist process lib/*.pdf --with-graph

# Search
./archivist search "transformer architecture" --top-k 10

# Find similar papers
./archivist similar lib/paper.pdf

# Show citations
./archivist cite show "Paper Title"

# Graph stats
./archivist graph stats

# Rebuild graph
./archivist graph rebuild
```

---

## Verify Setup

```bash
# Check services
curl http://localhost:7474        # Neo4j (should show web UI)
curl http://localhost:6333/healthz  # Qdrant (should return health status)
redis-cli ping                     # Redis (should return PONG)

# Check Archivist
./archivist --version
./archivist graph stats
```

---

## Troubleshooting

| Issue | Solution |
|-------|----------|
| **Services won't start** | `docker-compose -f docker-compose-graph.yml restart` |
| **"Connection refused"** | `docker ps` to check services, wait 30s for startup |
| **"Collection not found"** | Restart Archivist, it auto-creates on first run |
| **No search results** | Check `./archivist graph stats` - papers indexed? |
| **API key error** | Verify API key in `config/config.yaml` |

---

## Cost Estimate

For **50 papers**:
- Embeddings: $0.05
- Citation extraction: $0.05
- Analysis: $0.10
- **Total: ~$0.20** 💰

---

## Next Steps

1. Read the full guide: [KNOWLEDGE_GRAPH_GUIDE.md](./KNOWLEDGE_GRAPH_GUIDE.md)
2. Check implementation: [IMPLEMENTATION_SUMMARY.md](./IMPLEMENTATION_SUMMARY.md)
3. Review dependencies: [DEPENDENCIES.md](./DEPENDENCIES.md)

---

**Need help?** Open an issue on GitHub
