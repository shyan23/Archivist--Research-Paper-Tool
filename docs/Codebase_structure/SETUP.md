# 🚀 Archivist Setup Guide

Complete setup guide for new users installing Archivist with Knowledge Graph support.

---

## Quick Install (5 minutes)

### Option 1: Automated Install (Recommended)

```bash
# Clone or navigate to the project
cd /path/to/Archivist

# Run automated install script
chmod +x scripts/install.sh
./scripts/install.sh

# Follow the prompts
```

### Option 2: Using Makefile

```bash
# Install dependencies only
make install-graph-deps

# Install dependencies + build + setup services
make install-graph-deps && make build && make setup-graph
```

---

## Manual Installation

If you prefer to install step-by-step:

### Step 1: Install Prerequisites

#### Required:
- **Go 1.21+**: https://golang.org/dl/
- **Git**: Usually pre-installed on Linux/Mac

#### Optional (for Knowledge Graph):
- **Docker**: https://docs.docker.com/get-docker/
- **Docker Compose**: https://docs.docker.com/compose/install/

### Step 2: Install Go Dependencies

```bash
cd /path/to/Archivist

# Download all dependencies
go mod download

# Install Knowledge Graph specific packages
go get github.com/qdrant/go-client
go get google.golang.org/grpc/credentials/insecure

# Tidy up
go mod tidy

# Verify
go mod verify
```

**Expected output:**
```
all modules verified
```

### Step 3: Build Archivist

```bash
# Build the binary
go build -o archivist cmd/main/main.go

# Or if using cmd/rph structure
go build -o archivist cmd/rph/main.go

# Verify
./archivist --version
```

### Step 4: Setup Directories

```bash
mkdir -p lib tex_files reports logs .metadata
```

### Step 5: Configure API Key

Edit `config/config.yaml`:

```yaml
gemini:
  model: "models/gemini-2.0-flash-exp"
  max_tokens: 8000
  temperature: 0.3
  # Add your API key here ↓
  api_key: "YOUR_GEMINI_API_KEY_HERE"
```

Get API key from: https://makersuite.google.com/app/apikey

### Step 6: Setup Knowledge Graph (Optional)

```bash
# Start services
docker-compose -f docker-compose-graph.yml up -d

# Verify services are running
docker ps

# You should see:
# - archivist-neo4j
# - archivist-qdrant
# - archivist-redis
```

---

## Verify Installation

### Test Basic Functionality

```bash
# Check version
./archivist --version

# Show help
./archivist --help

# List commands
./archivist
```

### Test with Sample Paper

```bash
# Place a PDF in lib/
cp /path/to/paper.pdf lib/

# Process it
./archivist process lib/paper.pdf

# Check output
ls tex_files/
ls reports/
```

### Test Knowledge Graph (if enabled)

```bash
# Process with graph
./archivist process lib/paper.pdf --with-graph

# Check graph stats
./archivist graph stats

# Search
./archivist search "machine learning"
```

---

## Installed Dependencies

After installation, you'll have:

### Core Dependencies
```
✓ github.com/spf13/cobra          (CLI framework)
✓ github.com/spf13/viper          (Configuration)
✓ github.com/charmbracelet/*      (TUI components)
✓ github.com/google/generative-ai-go (Gemini AI)
```

### Knowledge Graph Dependencies
```
✓ github.com/qdrant/go-client              v1.15.2+
✓ github.com/neo4j/neo4j-go-driver/v5      v5.14.0+
✓ google.golang.org/grpc                   v1.76.0+
✓ google.golang.org/grpc/credentials/insecure
```

### Supporting Libraries
```
✓ github.com/redis/go-redis/v9    (Redis cache)
✓ gopkg.in/yaml.v3                (YAML parsing)
✓ github.com/fatih/color          (Terminal colors)
```

View full list: `go list -m all`

---

## Common Issues & Solutions

### Issue: "go: cannot find package"

```bash
# Solution: Re-download dependencies
go clean -modcache
go mod download
go mod tidy
```

### Issue: "docker: command not found"

**For Knowledge Graph only** - Docker is optional for basic Archivist functionality.

**Ubuntu/Debian:**
```bash
sudo apt-get update
sudo apt-get install docker.io docker-compose
sudo systemctl start docker
sudo usermod -aG docker $USER
# Logout and login again
```

**Mac:**
```bash
brew install docker docker-compose
# Or download Docker Desktop
```

### Issue: "Neo4j connection refused"

```bash
# Check if services are running
docker ps | grep neo4j

# Check logs
docker logs archivist-neo4j

# Restart services
docker-compose -f docker-compose-graph.yml restart
```

### Issue: "Qdrant gRPC error"

Edit `config/config.yaml`:
```yaml
qdrant:
  use_grpc: false  # Use HTTP instead
```

### Issue: "Gemini API key invalid"

1. Get a new key: https://makersuite.google.com/app/apikey
2. Update `config/config.yaml`
3. Ensure no extra spaces or quotes

---

## What Gets Installed?

### Go Packages (in go.mod)

The following packages are added to your `go.mod`:

```go
require (
    // Knowledge Graph - Vector Database
    github.com/qdrant/go-client v1.15.2

    // Knowledge Graph - Graph Database (already present)
    github.com/neo4j/neo4j-go-driver/v5 v5.14.0

    // Knowledge Graph - AI Embeddings (already present)
    github.com/google/generative-ai-go v0.20.1

    // gRPC for efficient communication
    google.golang.org/grpc v1.76.0

    // ... other existing dependencies
)
```

### Docker Services (optional)

If you run `make setup-graph`:

```yaml
services:
  neo4j:5.15-community     # Graph database (7474, 7687)
  qdrant:v1.7.4            # Vector database (6333, 6334)
  redis:7.2-alpine         # Cache (6379)
```

### Files Created

```
Archivist/
├── archivist              # Binary (after build)
├── go.mod                 # Updated with new dependencies
├── go.sum                 # Checksums for dependencies
├── lib/                   # Your PDF papers
├── tex_files/             # Generated LaTeX
├── reports/               # Final PDF reports
├── logs/                  # Processing logs
└── .metadata/             # Processing metadata
```

---

## Uninstall

### Remove Go Packages

```bash
# This won't actually remove, but you can clean:
go clean -modcache
```

### Stop Services

```bash
docker-compose -f docker-compose-graph.yml down -v
```

### Remove Binary

```bash
rm ./archivist
```

---

## Next Steps

1. **Read Documentation:**
   - [Quick Start](docs/QUICK_START.md) - Get started in 5 minutes
   - [Knowledge Graph Guide](docs/KNOWLEDGE_GRAPH_GUIDE.md) - Full feature guide
   - [Dependencies](docs/DEPENDENCIES.md) - Detailed dependency info

2. **Process Your First Paper:**
   ```bash
   ./archivist process lib/your_paper.pdf
   ```

3. **Explore Commands:**
   ```bash
   make help
   ./archivist --help
   ```

4. **Join Community:**
   - Report issues on GitHub
   - Contribute improvements
   - Share your experience

---

## Summary

### For Basic Paper Processing:
```bash
go mod download
go mod tidy
go build -o archivist cmd/main/main.go
```

### For Knowledge Graph:
```bash
# Install graph dependencies
go get github.com/qdrant/go-client
go get google.golang.org/grpc/credentials/insecure
go mod tidy

# Start services
docker-compose -f docker-compose-graph.yml up -d

# Build and run
go build -o archivist cmd/main/main.go
./archivist process lib/*.pdf --with-graph
```

---

**Installation complete! Ready to process research papers. 📚✨**
