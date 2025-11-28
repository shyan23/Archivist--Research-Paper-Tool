# 📦 Archivist Dependencies & Installation

## Go Dependencies

Add these to your `go.mod`:

```bash
# Navigate to project root
cd /home/shyan/Desktop/Code/Archivist

# Install Qdrant Go client
go get github.com/qdrant/go-client

# Install Neo4j Go driver (already added)
go get github.com/neo4j/neo4j-go-driver/v5

# Install Gemini Go SDK (already added)
go get github.com/google/generative-ai-go

# Install gRPC (for Qdrant)
go get google.golang.org/grpc
go get google.golang.org/grpc/credentials/insecure
```

## System Dependencies

### Docker & Docker Compose

```bash
# Ubuntu/Debian
sudo apt-get update
sudo apt-get install docker.io docker-compose

# Start Docker service
sudo systemctl start docker
sudo systemctl enable docker

# Add user to docker group (optional, avoids sudo)
sudo usermod -aG docker $USER
```

### For local development (without Docker):

#### Install Neo4j

```bash
# Option 1: Using apt (Ubuntu/Debian)
wget -O - https://debian.neo4j.com/neotechnology.gpg.key | sudo apt-key add -
echo 'deb https://debian.neo4j.com stable latest' | sudo tee /etc/apt/sources.list.d/neo4j.list
sudo apt-get update
sudo apt-get install neo4j

# Start Neo4j
sudo systemctl start neo4j
sudo systemctl enable neo4j

# Option 2: Using Docker (recommended)
docker run -d \
  --name neo4j \
  -p 7474:7474 -p 7687:7687 \
  -e NEO4J_AUTH=neo4j/password \
  neo4j:5.15-community
```

#### Install Qdrant

```bash
# Option 1: Using Docker (recommended)
docker run -d \
  --name qdrant \
  -p 6333:6333 -p 6334:6334 \
  -v $(pwd)/qdrant_storage:/qdrant/storage \
  qdrant/qdrant:v1.7.4

# Option 2: Binary download
wget https://github.com/qdrant/qdrant/releases/download/v1.7.4/qdrant-x86_64-unknown-linux-gnu.tar.gz
tar -xzf qdrant-x86_64-unknown-linux-gnu.tar.gz
./qdrant
```

#### Install Redis

```bash
# Ubuntu/Debian
sudo apt-get install redis-server

# Start Redis
sudo systemctl start redis-server
sudo systemctl enable redis-server

# Or using Docker
docker run -d --name redis -p 6379:6379 redis:7.2-alpine
```

## Verification

### Check Go Dependencies

```bash
cd /home/shyan/Desktop/Code/Archivist
go mod download
go mod verify
```

### Check Services

```bash
# Neo4j
curl http://localhost:7474
# Should return Neo4j browser page

# Qdrant
curl http://localhost:6333/healthz
# Should return: {"title":"Qdrant","version":"1.7.4"}

# Redis
redis-cli ping
# Should return: PONG
```

## Build Archivist

```bash
cd /home/shyan/Desktop/Code/Archivist

# Build the binary
go build -o archivist cmd/main/main.go

# Or run directly
go run cmd/main/main.go --help
```

## Troubleshooting

### "cannot find package" errors

```bash
go mod tidy
go get -u ./...
```

### Neo4j connection refused

```bash
# Check if Neo4j is running
docker ps | grep neo4j

# Check logs
docker logs archivist-neo4j

# Restart
docker restart archivist-neo4j
```

### Qdrant gRPC errors

```bash
# Ensure both ports are exposed
docker run -p 6333:6333 -p 6334:6334 qdrant/qdrant

# Or use HTTP only (update config.yaml: use_grpc: false)
```

### Redis connection issues

```bash
# Test connection
redis-cli ping

# Check config
redis-cli CONFIG GET bind

# Allow external connections (if needed)
sudo nano /etc/redis/redis.conf
# Change: bind 127.0.0.1 to bind 0.0.0.0
```

## Environment Variables

Create a `.env` file in the project root:

```bash
# Gemini API Key (required for embeddings)
GEMINI_API_KEY=your_api_key_here

# Neo4j credentials (if different from config)
NEO4J_URI=bolt://localhost:7687
NEO4J_USERNAME=neo4j
NEO4J_PASSWORD=password

# Qdrant settings (if different from config)
QDRANT_HOST=localhost
QDRANT_PORT=6333

# Redis settings (if different from config)
REDIS_ADDR=localhost:6379
```

## Next Steps

1. **Start services**: `./scripts/setup-graph.sh`
2. **Configure**: Edit `config/config.yaml` with your API key
3. **Process papers**: `./archivist process lib/*.pdf --with-graph`
4. **Search**: `./archivist search "your query"`

For detailed usage, see [KNOWLEDGE_GRAPH_GUIDE.md](./KNOWLEDGE_GRAPH_GUIDE.md)
