#!/bin/bash

set -e  # Exit on error

echo "🚀 Setting up Archivist Knowledge Graph..."
echo ""

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Check if Docker is running
if ! docker info > /dev/null 2>&1; then
    echo -e "${RED}❌ Docker is not running. Please start Docker first.${NC}"
    exit 1
fi

echo -e "${GREEN}✓${NC} Docker is running"

# 1. Check if Redis is already running
echo ""
echo "🔍 Checking Redis status..."

# Check if system Redis is running
if pgrep -x redis-server > /dev/null; then
    echo -e "${YELLOW}⚠️  System Redis is running on port 6379${NC}"
    echo -e "${GREEN}✓${NC} Using system Redis (no Docker container needed)"
    USING_SYSTEM_REDIS=true
elif docker ps --format '{{.Names}}' | grep -q "archivist-redis"; then
    echo -e "${GREEN}✓${NC} Docker Redis is already running"
    USING_SYSTEM_REDIS=false
elif docker ps -a --format '{{.Names}}' | grep -q "archivist-redis"; then
    echo "📦 Starting existing Redis container..."
    docker start archivist-redis
    USING_SYSTEM_REDIS=false
else
    echo "📦 Creating and starting Redis..."
    docker-compose up -d redis
    USING_SYSTEM_REDIS=false
fi

# 2. Start Neo4j service
echo ""
echo "📦 Starting Neo4j..."
docker-compose up -d neo4j

# 2. Wait for Neo4j to be ready
echo ""
echo "⏳ Waiting for Neo4j to be ready..."
MAX_RETRIES=30
RETRY_COUNT=0

while [ $RETRY_COUNT -lt $MAX_RETRIES ]; do
    if curl -s http://localhost:7474 > /dev/null 2>&1; then
        echo -e "${GREEN}✓${NC} Neo4j is ready!"
        break
    fi

    RETRY_COUNT=$((RETRY_COUNT + 1))
    if [ $RETRY_COUNT -eq $MAX_RETRIES ]; then
        echo -e "${RED}❌ Neo4j failed to start after ${MAX_RETRIES} attempts${NC}"
        echo "Check logs with: docker-compose logs neo4j"
        exit 1
    fi

    echo -n "."
    sleep 2
done

echo ""

# 3. Wait for Redis to be ready
echo "⏳ Checking Redis connection..."
RETRY_COUNT=0

while [ $RETRY_COUNT -lt $MAX_RETRIES ]; do
    # Test system Redis or Docker Redis
    if [ "$USING_SYSTEM_REDIS" = true ]; then
        if redis-cli ping > /dev/null 2>&1; then
            echo -e "${GREEN}✓${NC} Redis is ready!"
            break
        fi
    else
        if docker exec archivist-redis redis-cli ping > /dev/null 2>&1; then
            echo -e "${GREEN}✓${NC} Redis is ready!"
            break
        fi
    fi

    RETRY_COUNT=$((RETRY_COUNT + 1))
    if [ $RETRY_COUNT -eq $MAX_RETRIES ]; then
        echo -e "${RED}❌ Redis is not responding after ${MAX_RETRIES} attempts${NC}"
        if [ "$USING_SYSTEM_REDIS" = false ]; then
            echo "Check logs with: docker logs archivist-redis"
        fi
        exit 1
    fi

    echo -n "."
    sleep 1
done

echo ""

# 4. Initialize Neo4j schema (if graph-init tool exists)
if [ -f "cmd/graph-init/main.go" ]; then
    echo "🔧 Initializing Neo4j schema..."
    go run cmd/graph-init/main.go
else
    echo -e "${YELLOW}⚠️  graph-init tool not found, skipping schema initialization${NC}"
    echo -e "${YELLOW}   Schema will be initialized on first use${NC}"
fi

echo ""

# 5. Create papers directory for manual citations
echo "📁 Creating papers directory for manual citations..."
mkdir -p papers
echo -e "${GREEN}✓${NC} Papers directory created"

echo ""

# 6. Print access information
echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${GREEN}✅ Knowledge Graph setup complete!${NC}"
echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""
echo "📊 Access Neo4j Browser:"
echo "   URL: http://localhost:7474"
echo "   Username: neo4j"
echo "   Password: password"
echo ""
echo "📦 Redis is running on: localhost:6379"
echo ""
echo "📝 Manual citations directory: ./papers/"
echo "   Place <paper_name>_citations.yaml files here"
echo ""
echo "🚀 Ready to use explore mode!"
echo "   Try: archivist process lib/ --enable-graph"
echo "   Then: archivist explore \"attention mechanisms\""
echo ""
echo -e "${YELLOW}Note:${NC} To stop services: docker-compose down"
echo -e "${YELLOW}Note:${NC} To view logs: docker-compose logs -f neo4j redis"
echo ""
