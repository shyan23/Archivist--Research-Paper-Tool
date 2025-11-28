# 📦 Knowledge Graph Dependencies - Installation Guide

Quick reference for installing all Go packages needed for the Knowledge Graph feature.

---

## One-Line Install

```bash
make install-graph-deps
```

This will install:
- ✓ Qdrant Go client
- ✓ gRPC with insecure credentials
- ✓ Tidy all modules

---

## Manual Install

### Step 1: Install Qdrant Client

```bash
go get github.com/qdrant/go-client
```

**What it does:**
- Adds Qdrant vector database client (v1.15.2+)
- Provides gRPC and HTTP API access
- Enables vector search functionality

### Step 2: Install gRPC Insecure Credentials

```bash
go get google.golang.org/grpc/credentials/insecure
```

**What it does:**
- Adds gRPC credentials package
- Required for Qdrant gRPC connections
- Enables faster vector operations

### Step 3: Tidy Modules

```bash
go mod tidy
```

**What it does:**
- Removes unused dependencies
- Adds missing dependencies
- Updates go.sum with checksums

### Step 4: Verify

```bash
go mod verify
```

**Expected output:**
```
all modules verified
```

---

## What Gets Added to go.mod?

```go
require (
    github.com/qdrant/go-client v1.15.2
    google.golang.org/grpc v1.76.0
    // ... existing dependencies
)
```

---

## For Fresh Clone

If someone clones your repository:

```bash
# They just need to run:
go mod download

# Or with make:
make deps

# This downloads ALL dependencies including:
# - Qdrant client
# - Neo4j driver
# - Gemini AI SDK
# - All other packages
```

---

## Verification

Check if packages are installed:

```bash
go list -m github.com/qdrant/go-client
go list -m github.com/neo4j/neo4j-go-driver/v5
go list -m github.com/google/generative-ai-go
```

**Expected output:**
```
github.com/qdrant/go-client v1.15.2
github.com/neo4j/neo4j-go-driver/v5 v5.14.0
github.com/google/generative-ai-go v0.20.1
```

---

## Troubleshooting

### "cannot find package"

```bash
go clean -modcache
go mod download
go mod tidy
```

### "checksum mismatch"

```bash
rm go.sum
go mod tidy
```

### "version conflict"

```bash
go get -u github.com/qdrant/go-client
go mod tidy
```

---

## Complete Setup (From Scratch)

For a brand new user:

```bash
# 1. Clone the repo
git clone <repo-url>
cd Archivist

# 2. Install all dependencies (automatic)
go mod download

# 3. Build
make build

# 4. Done!
./archivist --help
```

**The go.mod file already contains all dependencies**, so `go mod download` installs everything automatically!

---

## Summary

| Command | What it installs | When to use |
|---------|-----------------|-------------|
| `go mod download` | Everything in go.mod | **Fresh clone** |
| `make install-graph-deps` | Only graph packages | **Adding graph feature** |
| `go get <package>` | Specific package | **Adding new dependency** |
| `go mod tidy` | Clean up | **After changes** |

---

**✅ Your go.mod is already configured!**

New users just need: `go mod download` 🎉
