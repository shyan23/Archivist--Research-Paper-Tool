#!/bin/bash

set -e  # Exit on error

echo "🚀 Setting up Archivist Knowledge Graph with Kafka..."
echo ""

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Navigate to project root
cd "$(dirname "$0")/.."

# Check if Docker is running
if ! docker info > /dev/null 2>&1; then
    echo -e "${RED}❌ Docker is not running. Please start Docker first.${NC}"
    exit 1
fi

echo -e "${GREEN}✓${NC} Docker is running"

# Check if docker-compose-graph.yml exists
if [ ! -f "docker-compose-graph.yml" ]; then
    echo -e "${RED}❌ docker-compose-graph.yml not found!${NC}"
    exit 1
fi

# Check for GEMINI_API_KEY
if [ -z "$GEMINI_API_KEY" ]; then
    echo -e "${YELLOW}⚠️  GEMINI_API_KEY not set in environment${NC}"
    echo -e "${YELLOW}   The graph service will not be able to extract embeddings${NC}"
    echo -e "${YELLOW}   Set it with: export GEMINI_API_KEY=your-key${NC}"
    echo ""
fi

# 1. Start all services
echo ""
echo "📦 Starting services (Kafka, Neo4j, Qdrant, Redis, Graph Service)..."
docker-compose -f docker-compose-graph.yml up -d

echo ""

# 2. Wait for Kafka to be ready
echo "⏳ Waiting for Kafka to be ready..."
MAX_RETRIES=30
RETRY_COUNT=0

while [ $RETRY_COUNT -lt $MAX_RETRIES ]; do
    if docker exec archivist-kafka kafka-topics.sh --list --bootstrap-server localhost:9092 > /dev/null 2>&1; then
        echo -e "${GREEN}✓${NC} Kafka is ready!"
        break
    fi

    RETRY_COUNT=$((RETRY_COUNT + 1))
    if [ $RETRY_COUNT -eq $MAX_RETRIES ]; then
        echo -e "${RED}❌ Kafka failed to start after ${MAX_RETRIES} attempts${NC}"
        echo "Check logs with: docker-compose -f docker-compose-graph.yml logs kafka"
        exit 1
    fi

    echo -n "."
    sleep 2
done

echo ""

# 3. Wait for Neo4j to be ready
echo "⏳ Waiting for Neo4j to be ready..."
RETRY_COUNT=0

while [ $RETRY_COUNT -lt $MAX_RETRIES ]; do
    if curl -s http://localhost:7474 > /dev/null 2>&1; then
        echo -e "${GREEN}✓${NC} Neo4j is ready!"
        break
    fi

    RETRY_COUNT=$((RETRY_COUNT + 1))
    if [ $RETRY_COUNT -eq $MAX_RETRIES ]; then
        echo -e "${RED}❌ Neo4j failed to start after ${MAX_RETRIES} attempts${NC}"
        echo "Check logs with: docker-compose -f docker-compose-graph.yml logs neo4j"
        exit 1
    fi

    echo -n "."
    sleep 2
done

echo ""

# 4. Wait for Qdrant to be ready
echo "⏳ Waiting for Qdrant to be ready..."
RETRY_COUNT=0

while [ $RETRY_COUNT -lt $MAX_RETRIES ]; do
    if curl -s http://localhost:6333/healthz > /dev/null 2>&1; then
        echo -e "${GREEN}✓${NC} Qdrant is ready!"
        break
    fi

    RETRY_COUNT=$((RETRY_COUNT + 1))
    if [ $RETRY_COUNT -eq $MAX_RETRIES ]; then
        echo -e "${RED}❌ Qdrant failed to start after ${MAX_RETRIES} attempts${NC}"
        echo "Check logs with: docker-compose -f docker-compose-graph.yml logs qdrant"
        exit 1
    fi

    echo -n "."
    sleep 2
done

echo ""

# 5. Wait for Redis to be ready
echo "⏳ Waiting for Redis to be ready..."
RETRY_COUNT=0

while [ $RETRY_COUNT -lt $MAX_RETRIES ]; do
    if docker exec archivist-redis redis-cli ping > /dev/null 2>&1; then
        echo -e "${GREEN}✓${NC} Redis is ready!"
        break
    fi

    RETRY_COUNT=$((RETRY_COUNT + 1))
    if [ $RETRY_COUNT -eq $MAX_RETRIES ]; then
        echo -e "${RED}❌ Redis failed to start after ${MAX_RETRIES} attempts${NC}"
        echo "Check logs with: docker-compose -f docker-compose-graph.yml logs redis"
        exit 1
    fi

    echo -n "."
    sleep 2
done

echo ""

# 6. Wait for Graph Service to be ready
echo "⏳ Waiting for Graph Service to be ready..."
RETRY_COUNT=0

while [ $RETRY_COUNT -lt $MAX_RETRIES ]; do
    if curl -s http://localhost:8081/health > /dev/null 2>&1; then
        echo -e "${GREEN}✓${NC} Graph Service is ready!"
        break
    fi

    RETRY_COUNT=$((RETRY_COUNT + 1))
    if [ $RETRY_COUNT -eq $MAX_RETRIES ]; then
        echo -e "${YELLOW}⚠️  Graph Service may not be responding${NC}"
        echo -e "${YELLOW}   Check logs with: docker-compose -f docker-compose-graph.yml logs graph-service${NC}"
        break
    fi

    echo -n "."
    sleep 2
done

echo ""

# 7. Create Kafka topic if it doesn't exist
echo "📡 Creating Kafka topic 'paper.processed'..."
docker exec archivist-kafka kafka-topics.sh \
    --create \
    --if-not-exists \
    --topic paper.processed \
    --bootstrap-server localhost:9092 \
    --partitions 3 \
    --replication-factor 1 \
    > /dev/null 2>&1

echo -e "${GREEN}✓${NC} Kafka topic created"

echo ""

# 8. Print service status
echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${GREEN}✅ Knowledge Graph setup complete!${NC}"
echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""
echo -e "${BLUE}📊 Service Access:${NC}"
echo "   • Neo4j Browser:    http://localhost:7474"
echo "     Username: neo4j | Password: password"
echo "   • Qdrant Dashboard: http://localhost:6333/dashboard"
echo "   • Redis:            localhost:6379"
echo "   • Kafka:            localhost:9092"
echo "   • Graph Service:    http://localhost:8081"
echo ""
echo -e "${BLUE}📝 Next Steps:${NC}"
echo "   1. Enable graph building in config/config.yaml:"
echo "      graph:"
echo "        enabled: true"
echo "        async_building: true"
echo ""
echo "   2. Process papers with graph building:"
echo "      ./archivist index lib/ --enable-graph"
echo ""
echo "   3. Or build graph from existing processed papers:"
echo "      ./archivist graph build"
echo ""
echo -e "${YELLOW}💡 Tip:${NC} The system will ask for permission before enabling concurrent"
echo "   graph building during paper processing"
echo ""
echo -e "${YELLOW}📋 Useful Commands:${NC}"
echo "   • Stop services:  docker-compose -f docker-compose-graph.yml down"
echo "   • View logs:      docker-compose -f docker-compose-graph.yml logs -f"
echo "   • Check status:   docker-compose -f docker-compose-graph.yml ps"
echo ""
