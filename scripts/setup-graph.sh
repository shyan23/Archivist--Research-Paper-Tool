#!/bin/bash

# Archivist Knowledge Graph Setup Script
# This script sets up Neo4j, Qdrant, and Redis for the knowledge graph feature

set -e

echo "🧠 Archivist Knowledge Graph Setup"
echo "===================================="
echo ""

# Check if Docker is installed
if ! command -v docker &> /dev/null; then
    echo "❌ Docker is not installed. Please install Docker first."
    exit 1
fi

if ! command -v docker-compose &> /dev/null; then
    echo "❌ Docker Compose is not installed. Please install Docker Compose first."
    exit 1
fi

echo "✓ Docker and Docker Compose are installed"
echo ""

# Navigate to project root
cd "$(dirname "$0")/.."

# Check if docker-compose-graph.yml exists
if [ ! -f "docker-compose-graph.yml" ]; then
    echo "❌ docker-compose-graph.yml not found!"
    exit 1
fi

echo "📦 Starting services..."
docker-compose -f docker-compose-graph.yml up -d

echo ""
echo "⏳ Waiting for services to be ready..."
sleep 5

# Wait for Neo4j
echo -n "  Waiting for Neo4j..."
for i in {1..30}; do
    if docker exec archivist-neo4j cypher-shell -u neo4j -p password "RETURN 1" &> /dev/null; then
        echo " ✓"
        break
    fi
    echo -n "."
    sleep 2
done

# Wait for Qdrant
echo -n "  Waiting for Qdrant..."
for i in {1..30}; do
    if curl -s http://localhost:6333/healthz &> /dev/null; then
        echo " ✓"
        break
    fi
    echo -n "."
    sleep 2
done

# Wait for Redis
echo -n "  Waiting for Redis..."
for i in {1..30}; do
    if docker exec archivist-redis redis-cli ping &> /dev/null; then
        echo " ✓"
        break
    fi
    echo -n "."
    sleep 2
done

echo ""
echo "✅ All services are running!"
echo ""
echo "Service URLs:"
echo "  • Neo4j Browser: http://localhost:7474 (neo4j / password)"
echo "  • Qdrant Dashboard: http://localhost:6333/dashboard"
echo "  • Redis: localhost:6379"
echo ""
echo "Next steps:"
echo "  1. Update your Gemini API key in config/config.yaml"
echo "  2. Process papers: ./archivist process lib/*.pdf --with-graph"
echo "  3. Search papers: ./archivist search 'attention mechanisms'"
echo ""
echo "For more information, see: docs/KNOWLEDGE_GRAPH_GUIDE.md"
