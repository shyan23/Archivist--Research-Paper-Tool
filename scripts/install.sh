#!/bin/bash

# Archivist Complete Installation Script
# This script sets up everything needed for Archivist including Knowledge Graph

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

echo "╔════════════════════════════════════════════════════════════╗"
echo "║          Archivist Installation & Setup                   ║"
echo "╚════════════════════════════════════════════════════════════╝"
echo ""

# Color codes
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Check if command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Print colored message
print_status() {
    echo -e "${GREEN}✓${NC} $1"
}

print_error() {
    echo -e "${RED}✗${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}⚠${NC} $1"
}

# Step 1: Check prerequisites
echo "📋 Step 1: Checking prerequisites..."
echo ""

# Check Go
if command_exists go; then
    GO_VERSION=$(go version | awk '{print $3}')
    print_status "Go is installed: $GO_VERSION"
else
    print_error "Go is not installed!"
    echo "Please install Go 1.21 or higher from: https://golang.org/dl/"
    exit 1
fi

# Check Docker
if command_exists docker; then
    DOCKER_VERSION=$(docker --version | awk '{print $3}' | sed 's/,//')
    print_status "Docker is installed: $DOCKER_VERSION"
else
    print_warning "Docker is not installed (optional for Knowledge Graph)"
fi

# Check Docker Compose
if command_exists docker-compose; then
    COMPOSE_VERSION=$(docker-compose --version | awk '{print $4}' | sed 's/,//')
    print_status "Docker Compose is installed: $COMPOSE_VERSION"
elif command_exists docker && docker compose version >/dev/null 2>&1; then
    print_status "Docker Compose (plugin) is installed"
else
    print_warning "Docker Compose is not installed (optional for Knowledge Graph)"
fi

echo ""

# Step 2: Navigate to project root
cd "$PROJECT_ROOT"
echo "📁 Step 2: Working directory: $PROJECT_ROOT"
echo ""

# Step 3: Install Go dependencies
echo "📦 Step 3: Installing Go dependencies..."
echo ""

echo "  Installing core dependencies..."
go mod download

echo "  Installing Knowledge Graph dependencies..."
go get github.com/qdrant/go-client
go get google.golang.org/grpc/credentials/insecure

echo "  Tidying modules..."
go mod tidy

echo ""
print_status "All Go dependencies installed!"
echo ""

# Step 4: Verify installation
echo "🔍 Step 4: Verifying installation..."
echo ""

go mod verify
if [ $? -eq 0 ]; then
    print_status "All modules verified successfully!"
else
    print_error "Module verification failed!"
    exit 1
fi

echo ""

# Step 5: Build the binary
echo "🔨 Step 5: Building Archivist..."
echo ""

if [ -f "cmd/main/main.go" ]; then
    go build -o archivist cmd/main/main.go
    print_status "Binary built: ./archivist"
elif [ -f "cmd/rph/main.go" ]; then
    go build -o archivist cmd/rph/main.go
    print_status "Binary built: ./archivist"
else
    print_warning "Main file not found at standard locations"
fi

echo ""

# Step 6: Setup directories
echo "📂 Step 6: Setting up directories..."
echo ""

mkdir -p lib tex_files reports logs .metadata
print_status "Created directories: lib, tex_files, reports, logs, .metadata"

echo ""

# Step 7: Setup Knowledge Graph services (optional)
echo "🧠 Step 7: Knowledge Graph setup (optional)"
echo ""

if command_exists docker && command_exists docker-compose; then
    read -p "Do you want to setup Knowledge Graph services (Neo4j, Qdrant, Redis)? [y/N] " -n 1 -r
    echo

    if [[ $REPLY =~ ^[Yy]$ ]]; then
        echo "  Starting services..."

        if [ -f "docker-compose-graph.yml" ]; then
            docker-compose -f docker-compose-graph.yml up -d

            echo "  Waiting for services to be ready (30s)..."
            sleep 30

            # Check Neo4j
            if docker exec archivist-neo4j cypher-shell -u neo4j -p password "RETURN 1" >/dev/null 2>&1; then
                print_status "Neo4j is running"
            else
                print_warning "Neo4j may still be starting..."
            fi

            # Check Qdrant
            if curl -s http://localhost:6333/healthz >/dev/null 2>&1; then
                print_status "Qdrant is running"
            else
                print_warning "Qdrant may still be starting..."
            fi

            # Check Redis
            if docker exec archivist-redis redis-cli ping >/dev/null 2>&1; then
                print_status "Redis is running"
            else
                print_warning "Redis may still be starting..."
            fi

            echo ""
            echo "  Service URLs:"
            echo "    • Neo4j:  http://localhost:7474 (neo4j / password)"
            echo "    • Qdrant: http://localhost:6333/dashboard"
            echo "    • Redis:  localhost:6379"
        else
            print_error "docker-compose-graph.yml not found!"
        fi
    else
        echo "  Skipping Knowledge Graph setup"
        echo "  You can set it up later with: make setup-graph"
    fi
else
    print_warning "Docker not available, skipping Knowledge Graph setup"
    echo "  Install Docker to use Knowledge Graph features"
fi

echo ""

# Step 8: Configuration check
echo "⚙️  Step 8: Configuration"
echo ""

if [ -f "config/config.yaml" ]; then
    print_status "Configuration file found: config/config.yaml"

    # Check if API key is set
    if grep -q "YOUR_GEMINI_API_KEY\|api_key: \"\"" config/config.yaml 2>/dev/null; then
        print_warning "Gemini API key not set in config/config.yaml"
        echo "  Get your API key from: https://makersuite.google.com/app/apikey"
    else
        print_status "Configuration appears complete"
    fi
else
    print_warning "Configuration file not found"
fi

echo ""

# Final summary
echo "╔════════════════════════════════════════════════════════════╗"
echo "║                  Installation Complete!                   ║"
echo "╚════════════════════════════════════════════════════════════╝"
echo ""
echo "📚 Installed Dependencies:"
echo "  ✓ github.com/qdrant/go-client (Qdrant vector database)"
echo "  ✓ github.com/neo4j/neo4j-go-driver/v5 (Neo4j graph database)"
echo "  ✓ github.com/google/generative-ai-go (Gemini AI)"
echo "  ✓ google.golang.org/grpc (gRPC communication)"
echo "  ✓ All other dependencies from go.mod"
echo ""
echo "🚀 Next Steps:"
echo ""
echo "  1. Configure Gemini API key:"
echo "     Edit config/config.yaml and add your API key"
echo "     Get key from: https://makersuite.google.com/app/apikey"
echo ""
echo "  2. Process your first paper:"
echo "     ./archivist process lib/your_paper.pdf"
echo ""
echo "  3. (Optional) Enable Knowledge Graph:"
echo "     ./archivist process lib/your_paper.pdf --with-graph"
echo ""
echo "  4. Search papers:"
echo "     ./archivist search 'attention mechanisms'"
echo ""
echo "📖 Documentation:"
echo "  • Quick Start:     docs/QUICK_START.md"
echo "  • Knowledge Graph: docs/KNOWLEDGE_GRAPH_GUIDE.md"
echo "  • Dependencies:    docs/DEPENDENCIES.md"
echo ""
echo "💡 Useful Commands:"
echo "  make help          - Show all available commands"
echo "  make build         - Rebuild the binary"
echo "  make start-services - Start Knowledge Graph services"
echo "  ./archivist --help - Show CLI help"
echo ""
echo "Happy researching! 📄🔬"
