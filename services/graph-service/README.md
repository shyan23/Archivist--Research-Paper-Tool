# Graph Service - Knowledge Graph Microservice

A standalone Python microservice that builds Neo4j knowledge graphs from research papers using Kafka for async communication.

## 🏗️ Architecture

```
┌────────────────────────────────────────────────────┐
│         Archivist (Go)                              │
│                                                    │
│  Process PDF → Generate LaTeX → Compile PDF       │
│                      │                             │
│                      ▼                             │
│           📤 Publish to Kafka                      │
│              topic: paper.processed                │
└───────────────────┬────────────────────────────────┘
                    │
                    │ Async Message Queue
                    ▼
┌────────────────────────────────────────────────────┐
│     Graph Service (Python + FastAPI)               │
│                                                    │
│  📥 Kafka Consumer → Worker Queue → Neo4j          │
│     │                    │                         │
│     ▼                    ▼                         │
│  Extract Metadata    Build Graph                   │
│  (Gemini LLM)        (Authors, Citations, etc.)    │
└────────────────────────────────────────────────────┘
                    │
                    ▼
         ┌──────────────────────┐
         │   Neo4j Database     │
         │                      │
         │  • Papers            │
         │  • Authors           │
         │  • Citations         │
         │  • Methods           │
         │  • Datasets          │
         └──────────────────────┘
```

## ✨ Features

- **🔌 Kafka Integration**: Async message-driven architecture
- **🧠 Smart Metadata Extraction**: Uses Gemini LLM to extract authors, citations, methods
- **⚡ Concurrent Processing**: 4 worker threads process papers in parallel
- **🔄 Non-blocking**: Main paper processing never waits for graph building
- **💾 Redis Caching**: Avoids reprocessing same papers
- **📊 REST API**: Query graph, get stats, manage jobs
- **🐳 Docker Ready**: Complete containerized setup

## 🚀 Quick Start

### 1. Start Services

```bash
# Start all services (Kafka, Neo4j, Redis, Graph Service)
docker-compose -f docker-compose-graph.yml up -d

# Check services are running
docker ps

# Expected containers:
# - archivist-kafka (port 9092)
# - archivist-neo4j (ports 7474, 7687)
# - archivist-redis (port 6379)
# - archivist-graph-service (port 8081)
```

### 2. Verify Services

```bash
# Check graph service health
curl http://localhost:8081/health

# Check Neo4j browser
open http://localhost:7474

# Check graph stats
curl http://localhost:8081/api/graph/stats
```

### 3. Process Papers

```bash
# Process papers (they will automatically be added to graph)
./archivist process lib/*.pdf
```

## 📡 Kafka Topic

The service listens to the `paper.processed` topic for messages like:

```json
{
  "paper_title": "Attention Is All You Need",
  "latex_content": "\\documentclass{article}...",
  "pdf_path": "/path/to/paper.pdf",
  "processed_at": "2025-11-13T10:30:00Z",
  "priority": 0
}
```

## 🛠️ API Endpoints

### Health & Stats

```bash
# Health check
GET /health

# Graph statistics
GET /api/graph/stats

# Worker queue statistics
GET /api/graph/queue-stats
```

### Manual Paper Addition (Optional)

```bash
# Add paper manually (if not using Kafka)
POST /api/graph/add-paper
{
  "paper_title": "Paper Title",
  "latex_content": "...",
  "pdf_path": "/path/to/paper.pdf"
}

# Check job status
GET /api/graph/job/{job_id}
```

### Graph Queries

```bash
# Get paper details
GET /api/graph/paper/{paper_title}

# Custom queries
POST /api/graph/query
{
  "query_type": "most_cited",
  "parameters": {"limit": 10}
}

# Delete paper
DELETE /api/graph/paper/{paper_title}
```

## 🔧 Configuration

### Environment Variables

Set these in `docker-compose-graph.yml` or `.env`:

```bash
# Kafka
KAFKA_BOOTSTRAP_SERVERS=kafka:9092

# Neo4j
NEO4J_URI=bolt://neo4j:7687
NEO4J_USER=neo4j
NEO4J_PASSWORD=password

# Redis
REDIS_URL=redis://redis:6379

# Gemini AI
GEMINI_API_KEY=your_api_key_here
```

### Worker Configuration

Edit `app/main.py`:

```python
# Number of concurrent workers
num_workers=4  # Adjust based on your CPU/memory
```

## 📊 Monitoring

### View Logs

```bash
# Graph service logs
docker logs -f archivist-graph-service

# Kafka logs
docker logs -f archivist-kafka

# Neo4j logs
docker logs -f archivist-neo4j
```

### Check Queue Status

```bash
curl http://localhost:8081/api/graph/queue-stats
```

Output:
```json
{
  "queue_size": 5,
  "processed_count": 23,
  "failed_count": 1,
  "active_workers": 4,
  "is_running": true
}
```

## 🧪 Testing

### Test Kafka Connection

```bash
# Install kafkacat
sudo apt install kafkacat

# List topics
kafkacat -b localhost:9092 -L

# Consume messages
kafkacat -b localhost:9092 -t paper.processed -C
```

### Test Graph Service Directly

```bash
# Test with sample data
curl -X POST http://localhost:8081/api/graph/add-paper \
  -H "Content-Type: application/json" \
  -d '{
    "paper_title": "Test Paper",
    "latex_content": "\\documentclass{article}\\begin{document}Test\\end{document}",
    "pdf_path": "/tmp/test.pdf"
  }'
```

### Query Neo4j Directly

```cypher
// Open Neo4j browser at http://localhost:7474

// Count all papers
MATCH (p:Paper) RETURN count(p)

// Show recent papers
MATCH (p:Paper)
RETURN p.title, p.year
ORDER BY p.processed_at DESC
LIMIT 10

// Find paper citations
MATCH (p1:Paper)-[r:CITES]->(p2:Paper)
RETURN p1.title, p2.title
LIMIT 10
```

## 🐛 Troubleshooting

### Service Not Starting

```bash
# Check if ports are available
sudo lsof -i :8081
sudo lsof -i :9092

# Restart services
docker-compose -f docker-compose-graph.yml restart

# View detailed logs
docker-compose -f docker-compose-graph.yml logs --follow
```

### Kafka Connection Issues

```bash
# Verify Kafka is running
docker exec archivist-kafka kafka-topics.sh --list --bootstrap-server localhost:9092

# Create topic manually (if needed)
docker exec archivist-kafka kafka-topics.sh \
  --create \
  --bootstrap-server localhost:9092 \
  --topic paper.processed \
  --partitions 3 \
  --replication-factor 1
```

### Neo4j Connection Issues

```bash
# Check Neo4j is running
docker exec archivist-neo4j cypher-shell -u neo4j -p password "RETURN 1"

# Reset Neo4j (WARNING: Deletes all data)
docker-compose -f docker-compose-graph.yml down -v
docker-compose -f docker-compose-graph.yml up -d
```

### Papers Not Being Added

```bash
# Check Kafka messages are being produced
kafkacat -b localhost:9092 -t paper.processed -C

# Check worker queue
curl http://localhost:8081/api/graph/queue-stats

# Check graph service logs
docker logs archivist-graph-service | grep "ERROR\|WARNING"
```

## 📈 Performance

### Expected Throughput

- **Metadata Extraction**: ~2-3 seconds per paper (Gemini API)
- **Graph Building**: ~1-2 seconds per paper (Neo4j writes)
- **Total**: ~4-5 seconds per paper
- **With 4 workers**: Can process ~50 papers concurrently

### Scaling

For large paper collections (100+ papers):

1. Increase workers: Change `num_workers=8` in `main.py`
2. Increase Kafka partitions for parallel consumption
3. Add more graph-service containers
4. Use Neo4j Enterprise for clustering

## 🔐 Security

- **Neo4j**: Change default password in production
- **Kafka**: Enable authentication for production
- **API**: Add API key authentication if exposing publicly

## 📚 Related Documentation

- [Neo4j Graph Structure](../../docs/GRAPH_STRUCTURE.md)
- [Knowledge Graph Guide](../../docs/KNOWLEDGE_GRAPH_GUIDE.md)
- [Main Documentation](../../docs/ARCHIVIST_DOCUMENTATION.md)

## 🤝 Contributing

The service is modular - you can:
- Add new metadata extractors
- Add custom graph queries
- Extend the Neo4j schema
- Add new Kafka topics

## 📝 License

Same as main Archivist project.
