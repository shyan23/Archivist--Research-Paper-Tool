.PHONY: help build run test test-unit test-integration test-all test-coverage test-verbose clean docker-build docker-run docker-shell docker-clean all install deps lint format bench setup-graph start-services stop-services install-graph-deps

# Default target
help:
	@echo "Archivist - Research Paper Helper"
	@echo ""
	@echo "Build & Run:"
	@echo "  make build          - Build archivist binary (./archivist)"
	@echo "  make build-graph-init - Build graph-init utility (./graph-init)"
	@echo "  make install        - Install to GOPATH/bin"
	@echo "  make run            - Run archivist"
	@echo "  make deps           - Install dependencies"
	@echo ""
	@echo "Testing:"
	@echo "  make test           - Run all tests"
	@echo "  make test-unit      - Run unit tests only"
	@echo "  make test-integration - Run integration tests"
	@echo "  make test-verbose   - Run tests with verbose output"
	@echo "  make test-coverage  - Run tests with coverage report"
	@echo "  make test-quick     - Run quick tests (for development)"
	@echo "  make bench          - Run benchmarks"
	@echo ""
	@echo "Code Quality:"
	@echo "  make lint           - Run linter"
	@echo "  make format         - Format code"
	@echo ""
	@echo "Docker:"
	@echo "  make docker-build   - Build Docker image"
	@echo "  make docker-run     - Run in Docker"
	@echo "  make docker-shell   - Interactive Docker shell"
	@echo "  make docker-test    - Test in Docker"
	@echo "  make docker-clean   - Clean Docker artifacts"
	@echo ""
	@echo "Knowledge Graph:"
	@echo "  make install-graph-deps - Install Qdrant Go client and gRPC"
	@echo "  make setup-graph    - Setup Neo4j, Qdrant, Redis services"
	@echo "  make start-services - Start knowledge graph services"
	@echo "  make stop-services  - Stop knowledge graph services"
	@echo ""
	@echo "Utilities:"
	@echo "  make clean          - Clean build artifacts"
	@echo "  make all            - Build both native and Docker"

# Native build targets
build:
	@echo "Building native binary..."
	go build -o archivist ./cmd/main
	@echo "✅ Build complete: ./archivist"
	@echo ""
	@echo "Run with: ./archivist --help"

build-graph-init:
	@echo "Building graph-init utility..."
	go build -o graph-init ./cmd/graph-init
	@echo "✅ Build complete: ./graph-init"

run: build
	./archivist run

# Test targets
test:
	@echo "Running all tests..."
	go test -race -timeout 5m ./...
	@echo "✅ All tests passed!"

test-unit:
	@echo "Running unit tests..."
	go test -race -short -timeout 2m ./pkg/... ./internal/storage ./internal/parser ./internal/generator ./internal/compiler ./internal/analyzer
	@echo "✅ Unit tests passed!"

test-integration:
	@echo "Running integration tests..."
	go test -race -timeout 5m ./internal -run Test.*Workflow
	@echo "✅ Integration tests passed!"

test-verbose:
	@echo "Running tests (verbose)..."
	go test -v -race -timeout 5m ./...

test-coverage:
	@echo "Running tests with coverage..."
	go test -race -timeout 5m -coverprofile=coverage.out -covermode=atomic ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "✅ Coverage report generated: coverage.html"
	@go tool cover -func=coverage.out | grep total | awk '{print "Total Coverage: " $$3}'

test-quick:
	@echo "Running quick tests..."
	go test -short -timeout 1m ./pkg/... ./internal/storage ./internal/generator
	@echo "✅ Quick tests passed!"

bench:
	@echo "Running benchmarks..."
	go test -bench=. -benchmem -run=^$$ ./...
	@echo "✅ Benchmarks complete!"

# Code quality targets
install:
	@echo "Installing Archivist..."
	go install ./cmd/main
	@echo "✅ Installed to $$(go env GOPATH)/bin/main"
	@echo "Note: Binary name will be 'main'. Consider creating alias: alias archivist='main'"

deps:
	@echo "Installing dependencies..."
	go mod download
	go mod tidy
	@echo "✅ Dependencies installed!"

# Knowledge Graph specific dependencies
install-graph-deps:
	@echo "📦 Installing Knowledge Graph dependencies..."
	@echo "  - Qdrant Go client"
	go get github.com/qdrant/go-client
	@echo "  - gRPC with insecure credentials"
	go get google.golang.org/grpc/credentials/insecure
	@echo "  - Tidying modules"
	go mod tidy
	@echo "✅ Knowledge Graph dependencies installed!"
	@echo ""
	@echo "Dependencies installed:"
	@echo "  ✓ github.com/qdrant/go-client"
	@echo "  ✓ github.com/neo4j/neo4j-go-driver/v5 (already present)"
	@echo "  ✓ github.com/google/generative-ai-go (already present)"
	@echo "  ✓ google.golang.org/grpc"

# Setup knowledge graph services
setup-graph:
	@echo "🧠 Setting up Knowledge Graph services..."
	@chmod +x scripts/setup-graph.sh
	@./scripts/setup-graph.sh

# Start services using Docker Compose
start-services:
	@echo "🚀 Starting services (Neo4j, Qdrant, Redis)..."
	docker-compose -f docker-compose-graph.yml up -d
	@echo "⏳ Waiting for services to be ready..."
	@sleep 10
	@echo "✅ Services started!"
	@echo ""
	@echo "Service URLs:"
	@echo "  • Neo4j Browser:  http://localhost:7474 (neo4j / password)"
	@echo "  • Qdrant Dashboard: http://localhost:6333/dashboard"
	@echo "  • Redis:          localhost:6379"

# Stop services
stop-services:
	@echo "🛑 Stopping services..."
	docker-compose -f docker-compose-graph.yml down
	@echo "✅ Services stopped!"

lint:
	@echo "Running linter..."
	@which golangci-lint > /dev/null || (echo "⚠️  golangci-lint not installed. Install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest" && exit 1)
	golangci-lint run ./...
	@echo "✅ Linting complete!"

format:
	@echo "Formatting code..."
	go fmt ./...
	@echo "✅ Code formatted!"

clean:
	@echo "Cleaning build artifacts..."
	rm -f archivist graph-init rph
	rm -f coverage.out coverage.html
	rm -f tex_files/*.aux tex_files/*.log tex_files/*.out tex_files/*.toc
	rm -f tex_files/*.fdb_latexmk tex_files/*.fls tex_files/*.synctex.gz
	@echo "✅ Clean complete"

# Docker targets
docker-build:
	@echo "Building Docker image..."
	docker-compose build
	@echo "✅ Docker image built"

docker-run: docker-build
	@echo "Running in Docker..."
	docker-compose run --rm archivist check

docker-shell: docker-build
	@echo "Starting interactive shell..."
	docker-compose --profile interactive run --rm archivist-shell

docker-test: docker-build
	@echo "Testing in Docker..."
	docker-compose run --rm archivist process lib/csit140108.pdf

docker-clean:
	@echo "Cleaning Docker artifacts..."
	docker-compose down --rmi local -v
	@echo "✅ Docker cleaned"

# Build both
all: build docker-build
	@echo "✅ All builds complete"

# Quick process
process: build
	./archivist process lib/

docker-process: docker-build
	docker-compose run --rm archivist process lib/

# List processed papers
list: build
	./archivist list

docker-list: docker-build
	docker-compose run --rm archivist list
