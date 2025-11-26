# Graph Service Testing Guide

Quick guide to test the knowledge graph microservice independently.

## 🧪 Test 1: Start Services

```bash
# Start all services
cd /home/shyan/Desktop/Code/Archivist
docker-compose -f docker-compose-graph.yml up -d

# Wait for all services to be healthy (30-60 seconds)
watch docker ps

# Expected: All containers show "healthy" status
```

## 🧪 Test 2: Verify Connectivity

```bash
# Test Kafka
docker exec archivist-kafka kafka-topics.sh --list --bootstrap-server localhost:9092

# Test Neo4j
docker exec archivist-neo4j cypher-shell -u neo4j -p password "RETURN 1"

# Test Redis
docker exec archivist-redis redis-cli ping

# Test Graph Service
curl http://localhost:8081/health
```

## 🧪 Test 3: Manual Paper Submission

```bash
# Submit a test paper directly to the API
curl -X POST http://localhost:8081/api/graph/add-paper \
  -H "Content-Type: application/json" \
  -d '{
    "paper_title": "Attention Is All You Need",
    "latex_content": "\\documentclass{article}\n\\author{Vaswani et al.}\n\\begin{document}\nWe propose the Transformer model using self-attention mechanisms trained on WMT 2014 dataset.\n\\end{document}",
    "pdf_path": "/tmp/attention.pdf",
    "priority": 1
  }'

# Expected response:
# {
#   "status": "queued",
#   "message": "Paper queued for graph building",
#   "job_id": "some-uuid",
#   "queue_position": 1
# }
```

## 🧪 Test 4: Check Job Status

```bash
# Replace JOB_ID with the one from previous step
curl http://localhost:8081/api/graph/job/JOB_ID

# Watch logs
docker logs -f archivist-graph-service

# You should see:
# - Metadata extraction
# - Adding paper node
# - Adding authors
# - Adding methods/datasets
```

## 🧪 Test 5: Query Neo4j

```bash
# Check graph stats
curl http://localhost:8081/api/graph/stats

# Expected:
# {
#   "paper_count": 1,
#   "author_count": 1+,
#   "citation_count": 0,
#   ...
# }

# Query Neo4j directly
docker exec archivist-neo4j cypher-shell -u neo4j -p password \
  "MATCH (p:Paper) RETURN p.title, p.year LIMIT 10"
```

## 🧪 Test 6: End-to-End with Archivist

```bash
# Process a real paper (this will auto-publish to Kafka)
./archivist process lib/some_paper.pdf

# The paper will:
# 1. Get processed by Archivist (LaTeX generation)
# 2. Be published to Kafka topic "paper.processed"
# 3. Be consumed by graph-service
# 4. Get added to Neo4j graph

# Check it's in the graph
curl http://localhost:8081/api/graph/stats
```

## 🧪 Test 7: Kafka Message Flow

```bash
# Install kafkacat for debugging
sudo apt install kafkacat

# Monitor messages on the topic
kafkacat -b localhost:9092 -t paper.processed -C

# In another terminal, process a paper
./archivist process lib/test_paper.pdf

# You should see the JSON message appear in kafkacat output
```

## 🧪 Test 8: Performance Test

```bash
# Submit multiple papers
for file in lib/*.pdf; do
  ./archivist process "$file"
done

# Monitor worker queue
watch -n 1 'curl -s http://localhost:8081/api/graph/queue-stats | jq'

# You should see:
# - queue_size decreasing
# - processed_count increasing
# - active_workers showing all workers busy
```

## 🧪 Test 9: Graph Queries

```bash
# Get paper details
curl http://localhost:8081/api/graph/paper/Attention%20Is%20All%20You%20Need

# Find most cited papers
curl -X POST http://localhost:8081/api/graph/query \
  -H "Content-Type: application/json" \
  -d '{
    "query_type": "most_cited",
    "parameters": {"limit": 10}
  }'

# Find papers by author
curl -X POST http://localhost:8081/api/graph/query \
  -H "Content-Type: application/json" \
  -d '{
    "query_type": "author_papers",
    "parameters": {"author_name": "Vaswani"}
  }'
```

## 🧪 Test 10: Redis Caching

```bash
# Process same paper twice
./archivist process lib/paper.pdf

# First time: Graph building happens
# Second time: Check logs

docker logs archivist-graph-service | grep "already in graph"

# You should see: "Paper already in graph (cached)"
```

## 🔍 Troubleshooting Tests

### Kafka Not Receiving Messages

```bash
# Check Go code is publishing
grep "Publishing to Kafka" archivist.log

# Check Kafka topic exists
docker exec archivist-kafka kafka-topics.sh --describe \
  --topic paper.processed \
  --bootstrap-server localhost:9092
```

### Graph Service Not Processing

```bash
# Check consumer is running
docker logs archivist-graph-service | grep "Kafka consumer started"

# Check for errors
docker logs archivist-graph-service | grep "ERROR"

# Restart service
docker-compose -f docker-compose-graph.yml restart graph-service
```

### Neo4j Empty

```bash
# Verify papers are being added
docker exec archivist-neo4j cypher-shell -u neo4j -p password \
  "MATCH (n) RETURN labels(n), count(n)"

# If empty, check constraints
docker exec archivist-neo4j cypher-shell -u neo4j -p password \
  "SHOW CONSTRAINTS"
```

## ✅ Success Criteria

After all tests, you should have:

- ✅ All Docker containers running and healthy
- ✅ Kafka topic `paper.processed` exists
- ✅ Neo4j contains papers, authors, methods, datasets
- ✅ Graph service processing jobs without errors
- ✅ Redis cache preventing duplicate processing
- ✅ End-to-end flow working (Archivist → Kafka → Graph)

## 📊 Expected Output

After processing 10 papers, you should see:

```json
{
  "paper_count": 10,
  "author_count": 25-50,
  "citation_count": 10-30,
  "method_count": 20-40,
  "dataset_count": 5-15,
  "venue_count": 5-10
}
```

## 🎯 Next Steps

Once tests pass:
1. Process your entire PDF library
2. Build custom graph queries
3. Create visualizations
4. Export data for analysis

Happy testing! 🚀
