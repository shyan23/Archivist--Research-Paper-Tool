# Archivist - Quick Reference: Critical Fixes

## Status: 3 CRITICAL BUGS BLOCKING DEPLOYMENT

### 🔴 CRITICAL #1: TUI Build Failure
**File:** `internal/tui/loaders.go:123`  
**Status:** BROKEN - Blocks compilation
```go
// WRONG (line 123):
m.multiPaperList.Title = fmt.Sprintf("📋 Select Papers (Space to toggle, Enter to confirm) - 0 selected", len(items))

// FIX:
m.multiPaperList.Title = fmt.Sprintf("📋 Select Papers (Space to toggle, Enter to confirm) - %d available", len(items))
```
**Time to fix:** 2 minutes  
**Verification:** `go build ./...` should pass

---

### 🔴 CRITICAL #2: API Timeout Vulnerability
**File:** `internal/analyzer/gemini_client.go:75-117` + `internal/worker/pool.go:131`  
**Status:** BROKEN - Workers can hang indefinitely

**Problem:** 
- `config.yaml` specifies `timeout_per_paper: 600` but it's never used
- Gemini API calls have no timeout enforcement
- Workers block forever on slow API responses

**Symptoms:**
- Process hangs with no output
- High memory usage from blocked goroutines
- CPU stuck at 0%

**Quick Fix:**
```go
// In worker/pool.go, before calling analyzer:
ctx, cancel := context.WithTimeout(ctx, time.Duration(wp.config.Processing.TimeoutPerPaper)*time.Second)
defer cancel()

latexContent, err = analyzer.AnalyzePaper(ctx, job.FilePath)
```

**Time to fix:** 10 minutes  
**Verification:** Kill process during analysis - should exit cleanly

---

### 🔴 CRITICAL #3: Log File Resource Leak
**File:** `internal/app/logger.go:28-33`  
**Status:** BROKEN - File descriptor leak every run

**Problem:**
```go
logFile, err := os.OpenFile(config.Logging.File, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
if err != nil {
    return fmt.Errorf("failed to open log file: %w", err)
}
writers = append(writers, logFile)
// ^^^ logFile NEVER CLOSED - fd leak!
```

**Quick Fix:**
```go
// Store logFile for cleanup
type LogCloser struct {
    *os.File
}

// Then in main's defer:
defer func() {
    if logFile != nil {
        logFile.Close()
    }
}()
```

OR use `io.WriteCloser` and track it globally.

**Time to fix:** 15 minutes  
**Verification:** Check `lsof -p $(pgrep archivist) | grep .log` - shouldn't grow each run

---

## High Priority Fixes (Next 48 hours)

### 🟠 HIGH #1: Replace MD5 with SHA-256
**File:** `pkg/fileutil/hash.go:20`
```go
// Change from:
hasher := md5.New()

// To:
hasher := sha256.New()
```
**Impact:** Data integrity  
**Time:** 5 minutes + test update

---

### 🟠 HIGH #2: Add Error Categorization
**File:** `internal/worker/pool.go:132-136`  
Distinguish between:
- Transient errors (network, rate limit) → retry
- Permanent errors (invalid PDF, API key) → fail
- Timeout errors → fail with helpful message

**Time:** 1-2 hours

---

### 🟠 HIGH #3: Implement Interface-Based Architecture
**File:** `internal/worker/pool.go:35-42`

Current (WRONG):
```go
type WorkerPool struct {
    cache      *cache.RedisCache  // Concrete type - hard to test
}
```

Better:
```go
type CacheStore interface {
    Get(ctx context.Context, key string) (*CachedAnalysis, error)
    Set(ctx context.Context, key string, value *CachedAnalysis) error
    Close() error
}

type WorkerPool struct {
    cache      CacheStore  // Interface - easy to mock/test
}
```

**Time:** 2-3 hours  
**Benefit:** Can unit test without Redis

---

## Test Coverage Status

| Module | Tests | Coverage |
|--------|-------|----------|
| `pkg/fileutil` | 11 | HIGH ✓ |
| `internal/worker` | 0 | NONE ✗ |
| `internal/analyzer` | 0 | NONE ✗ |
| `internal/cache` | 0 | NONE ✗ |
| `internal/compiler` | 0 | NONE ✗ |
| **Total** | **11/47** | **23%** |

**Priority:** Write tests for worker pool & analyzer (critical path)

---

## Configuration Issues

### Missing Validation in `app/config.go`

Add to `LoadConfig()`:
```go
// Validate MaxWorkers
if config.Processing.MaxWorkers <= 0 || config.Processing.MaxWorkers > runtime.NumCPU() {
    return nil, fmt.Errorf("MaxWorkers must be > 0 and <= %d", runtime.NumCPU())
}

// Validate TimeoutPerPaper
if config.Processing.TimeoutPerPaper <= 0 {
    return nil, fmt.Errorf("TimeoutPerPaper must be > 0 seconds")
}

// Validate Cache TTL
if config.Cache.Enabled && config.Cache.TTL <= 0 {
    return nil, fmt.Errorf("Cache TTL must be > 0 hours")
}

// Validate Model format
if !strings.HasPrefix(config.Gemini.Model, "models/") {
    return nil, fmt.Errorf("Invalid model format: %s (must start with 'models/')", config.Gemini.Model)
}
```

**Time:** 20 minutes

---

## Architecture Issues Summary

### Worker Pool Design Problems

```
Current: Linear processing per worker
┌─────────────────────────────┐
│ Worker 1: PDF → API → LaTeX → Compile
│ Worker 2: PDF → API → LaTeX → Compile
└─────────────────────────────┘
Issue: Each step blocks the next, inefficient

Better: Pipeline stages
┌──────────┬──────────┬──────────┬──────────┐
│ Parser   │ Analyzer │ Generator│ Compiler │
│ Goroutine│ Goroutine│ Goroutine│ Goroutine│
└──────────┴──────────┴──────────┴──────────┘
Benefit: N papers processed simultaneously across stages
```

---

## File-by-File Risk Assessment

| File | LOC | Issues | Risk |
|------|-----|--------|------|
| `internal/worker/pool.go` | 362 | 4 critical | 🔴 HIGH |
| `internal/analyzer/gemini_client.go` | 195 | 1 critical | 🔴 HIGH |
| `internal/tui/loaders.go` | 129 | 1 critical | 🔴 HIGH |
| `internal/app/logger.go` | 44 | 1 critical | 🔴 HIGH |
| `pkg/fileutil/hash.go` | 84 | 1 high (MD5) | 🟠 MED |
| `internal/compiler/latex_compiler.go` | 158 | 1 medium | 🟠 MED |
| `internal/cache/redis_cache.go` | 203 | 2 medium | 🟠 MED |
| `cmd/main/commands/process.go` | 261 | 2 medium | 🟠 MED |

---

## Quick Command Reference

### Check current status:
```bash
go build ./...                    # Should fail due to TUI error
go test ./...                     # Minimal test coverage
```

### After fixes:
```bash
go test ./... -v -cover           # Check coverage
go build -o archivist ./cmd/main  # Build binary
./archivist process lib/          # Test
```

---

## Resource Leak Detection

```bash
# Monitor file descriptors
watch -n 1 'lsof -p $(pgrep archivist) | wc -l'

# Monitor goroutines (add pprof endpoint)
curl http://localhost:6060/debug/pprof/goroutine

# Memory profiling
go build -o archivist ./cmd/main
./archivist process lib/ &
go tool pprof http://localhost:6060/debug/pprof/heap
```

---

## Deployment Readiness Checklist

- [ ] Fix TUI format string (CRITICAL)
- [ ] Add context timeout enforcement (CRITICAL)
- [ ] Fix log file leak (CRITICAL)
- [ ] Replace MD5 with SHA-256
- [ ] Add error categorization
- [ ] Unit tests for worker pool
- [ ] Unit tests for analyzer
- [ ] Config validation
- [ ] Graceful shutdown handler
- [ ] Remove dead code variables
- [ ] Structured logging (JSON)

---

## Issues by Severity for Prioritization

### Must Fix Before Deployment (Day 1)
1. TUI build error → `go build` fails
2. Timeout vulnerability → process hangs
3. Log leak → fd exhaustion  
4. MD5 → data integrity

**Estimated Time: 1 hour**

### Must Fix Before Production (Day 2-3)
5. Error handling → can't recover from failures
6. Interface design → can't test/maintain
7. Config validation → bad config silently breaks
8. Test coverage → no confidence in changes

**Estimated Time: 8-12 hours**

### Should Fix Before Release (Week 1)
9. Graceful shutdown → clean exit
10. Progress persistence → can resume jobs
11. Observability → can debug issues
12. Pipeline architecture → better performance

**Estimated Time: 16-24 hours**

