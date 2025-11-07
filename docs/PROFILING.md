# Profiling Guide for Archivist

This guide explains how to profile your Research Paper Helper (Archivist) application to identify performance bottlenecks, optimize resource usage, and improve overall efficiency.

## Table of Contents

1. [What is Profiling?](#what-is-profiling)
2. [Quick Start](#quick-start)
3. [Profiling Options](#profiling-options)
4. [Analyzing Profiles](#analyzing-profiles)
5. [Common Performance Issues](#common-performance-issues)
6. [Best Practices](#best-practices)

---

## What is Profiling?

Profiling is the process of measuring where your program spends its time and how much memory it uses. The Archivist includes built-in profiling capabilities similar to Python's `cProfile`, providing:

- **CPU Profiling**: Identifies which functions consume the most CPU time
- **Memory Profiling**: Shows memory allocation and usage patterns
- **Function Timing**: Tracks execution time for individual functions with call counts
- **Performance Metrics**: Memory statistics, goroutine counts, garbage collection stats

---

## Quick Start

### Enable Profiling

To enable profiling for any command, simply add the `--profile` flag:

```bash
# Profile a single paper processing
./rph process lib/paper.pdf --profile

# Profile batch processing
./rph process lib/ --profile

# Custom profile output directory
./rph process lib/ --profile --profile-dir=./my_profiles
```

### What You Get

When profiling is enabled, Archivist will generate three files:

1. **CPU Profile** (`cpu_<timestamp>.prof`): CPU usage data
2. **Memory Profile** (`mem_<timestamp>.prof`): Memory allocation data
3. **Timing Report** (`timings_<timestamp>.txt`): Human-readable function timing statistics

Example output:
```
📊 Profiling started
   CPU Profile: profiles/cpu_20250113_143022.prof

[... your program runs ...]

📊 Profiling stopped (duration: 2m34s)
   Memory Profile: profiles/mem_20250113_143022.prof
   Timing Report: profiles/timings_20250113_143022.txt

To analyze profiles, run:
   go tool pprof -http=:8080 profiles/cpu_20250113_143022.prof
   go tool pprof -http=:8080 profiles/mem_20250113_143022.prof
```

---

## Profiling Options

### Global Flags

```bash
--profile              # Enable profiling (default: false)
--profile-dir=DIR      # Output directory for profiles (default: ./profiles)
```

### Examples

```bash
# Basic profiling
./rph process lib/attention.pdf --profile

# Quality mode with profiling
./rph process lib/ --mode quality --profile

# Parallel processing with profiling
./rph process lib/ --parallel 4 --profile

# Custom output location
./rph process lib/ --profile --profile-dir=/tmp/perf_analysis
```

---

## Analyzing Profiles

### Using the Helper Script

Archivist includes a convenient shell script for profile analysis:

```bash
# List all available profiles
./scripts/analyze_profile.sh list

# Analyze the latest CPU profile
./scripts/analyze_profile.sh cpu

# Analyze the latest memory profile
./scripts/analyze_profile.sh mem

# Show top CPU-consuming functions
./scripts/analyze_profile.sh top

# View timing report
./scripts/analyze_profile.sh timings

# Open profile in web browser (interactive visualization)
./scripts/analyze_profile.sh web cpu

# Compare two CPU profiles (e.g., before and after optimization)
./scripts/analyze_profile.sh compare profiles/cpu_before.prof profiles/cpu_after.prof

# Analyze specific profile file
./scripts/analyze_profile.sh cpu profiles/cpu_20250113_143022.prof
```

### Manual Analysis with `go tool pprof`

#### CPU Profile Analysis

```bash
# Terminal UI (text-based)
go tool pprof profiles/cpu_20250113_143022.prof

# Interactive commands within pprof:
(pprof) top          # Show top functions by CPU usage
(pprof) top10        # Show top 10 functions
(pprof) list main    # Show line-by-line profile for main package
(pprof) web          # Generate call graph (requires Graphviz)
```

#### Memory Profile Analysis

```bash
# Analyze memory allocations
go tool pprof profiles/mem_20250113_143022.prof

(pprof) top          # Top memory allocators
(pprof) list .       # Show all allocations
```

#### Web UI (Recommended)

The most powerful way to analyze profiles is using the built-in web UI:

```bash
# Open CPU profile in browser
go tool pprof -http=:8080 profiles/cpu_20250113_143022.prof

# Open memory profile in browser
go tool pprof -http=:8080 profiles/mem_20250113_143022.prof
```

Then navigate to `http://localhost:8080` in your browser. You'll see:

- **Graph**: Visual call graph with time/memory annotations
- **Flame Graph**: Interactive flame graph visualization
- **Top**: List of top functions
- **Source**: Line-by-line source code with annotations
- **Disassemble**: Assembly code view

---

## Understanding the Timing Report

The timing report provides a Python `cProfile`-style output:

```
Function Timing Report
Generated: 2025-01-13 14:30:22
Total Duration: 2m34s

==================================================================================================
FUNCTION                                           CALLS          TOTAL            AVG            MIN            MAX
--------------------------------------------------------------------------------------------------
worker.ProcessBatch                                    1        2m33s          2m33s          2m33s          2m33s
analyzer.Analyze                                      15        1m45s          7s             4s             12s
gemini.GenerateContent                                45        1m12s          1.6s           800ms          3.2s
generator.GenerateLatex                               15        28s            1.8s           1.2s           2.4s
compiler.CompilePDF                                   15        19s            1.2s           900ms          1.8s
parser.ExtractContent                                 15        8s             533ms          412ms          687ms
storage.SaveMetadata                                  15        245ms          16ms           12ms           24ms
==================================================================================================
```

**Reading the Report:**
- **CALLS**: How many times the function was called
- **TOTAL**: Total time spent in this function (across all calls)
- **AVG**: Average time per call
- **MIN**: Fastest single call
- **MAX**: Slowest single call

---

## Common Performance Issues

### 1. Gemini API Latency

**Symptom**: `gemini.GenerateContent` takes a long time

**Solutions**:
- Enable Redis caching to avoid repeated API calls
- Use `--mode fast` for quicker but less detailed analysis
- Increase `--parallel` workers to process multiple papers simultaneously

```bash
# Enable caching in config.yaml
cache:
  enabled: true
  type: redis

# Use fast mode
./rph process lib/ --mode fast --parallel 8
```

### 2. LaTeX Compilation Bottleneck

**Symptom**: `compiler.CompilePDF` is slow

**Solutions**:
- Ensure `latexmk` is installed (faster than multiple pdflatex passes)
- Check if LaTeX is doing unnecessary work (large images, complex formulas)
- Consider using `xelatex` or `lualatex` instead of `pdflatex`

```yaml
# config/config.yaml
latex:
  compiler: "pdflatex"  # or "xelatex", "lualatex"
  engine: "latexmk"
  clean_aux: true
```

### 3. Memory Leaks

**Symptom**: Memory usage keeps growing

**Analysis**:
```bash
# Check memory profile
go tool pprof -http=:8080 profiles/mem_20250113_143022.prof

# Look for:
# - Functions with unexpectedly high allocations
# - Growing allocations over time
# - Large temporary buffers
```

**Solutions**:
- Ensure proper cleanup of resources (close files, connections)
- Check for goroutine leaks
- Review caching strategies

### 4. Goroutine Bottlenecks

**Symptom**: CPU usage is low despite high parallelism

**Analysis**:
```bash
# Check goroutine count in timing report
# High goroutine count + low CPU = contention or blocking I/O
```

**Solutions**:
- Adjust `max_workers` in config
- Profile with `GODEBUG=schedtrace=1000` to see scheduler behavior
- Check for excessive locking or channel operations

---

## Best Practices

### 1. Profile Before Optimizing

> "Premature optimization is the root of all evil" - Donald Knuth

Always profile first to identify **actual** bottlenecks, not assumed ones.

### 2. Compare Before and After

```bash
# Create baseline
./rph process lib/test_papers/ --profile --profile-dir=./before

# Make optimizations
# ... edit code ...

# Create comparison
./rph process lib/test_papers/ --profile --profile-dir=./after

# Compare
./scripts/analyze_profile.sh compare before/cpu_*.prof after/cpu_*.prof
```

### 3. Profile Realistic Workloads

- Use actual research papers, not toy examples
- Test with different paper sizes and complexities
- Profile both single and batch processing

### 4. Monitor Over Time

Keep profiles from different versions to track performance trends:

```bash
profiles/
├── v1.0/
│   ├── cpu_baseline.prof
│   └── mem_baseline.prof
├── v1.1/
│   ├── cpu_with_caching.prof
│   └── mem_with_caching.prof
└── v1.2/
    ├── cpu_optimized.prof
    └── mem_optimized.prof
```

### 5. Focus on Impact

Optimize the biggest bottlenecks first. A 50% improvement in a function that takes 1% of runtime only saves 0.5% overall, but a 10% improvement in a function taking 60% of runtime saves 6%.

---

## Advanced Profiling

### Continuous Profiling in Production

For long-running processes or production deployments:

```go
import _ "net/http/pprof"

// In your main.go, add:
go func() {
    log.Println(http.ListenAndServe("localhost:6060", nil))
}()
```

Then access profiles at:
- http://localhost:6060/debug/pprof/
- http://localhost:6060/debug/pprof/goroutine
- http://localhost:6060/debug/pprof/heap

### Benchmarking

For micro-benchmarks of specific functions:

```bash
# Run benchmarks
go test -bench=. -benchmem ./internal/...

# With CPU profiling
go test -bench=. -cpuprofile=cpu.prof ./internal/analyzer

# Analyze
go tool pprof cpu.prof
```

---

## Troubleshooting

### "go: command not found"

Install Go from https://golang.org/dl/

### "Graphviz not installed" (for `web` command)

```bash
sudo apt install graphviz  # Ubuntu/Debian
brew install graphviz      # macOS
```

### Profile files are too large

Use sampling options:

```bash
# Sample less frequently (default is every 10ms)
go tool pprof -seconds=30 -sample_index=cpu profiles/cpu_*.prof
```

### Cannot connect to Redis (for caching)

```bash
# Start Redis
sudo systemctl start redis

# Or install Redis
sudo apt install redis-server
```

---

## Resources

- [Go pprof Documentation](https://pkg.go.dev/runtime/pprof)
- [Profiling Go Programs](https://go.dev/blog/pprof)
- [Flame Graphs](http://www.brendangregg.com/flamegraphs.html)
- [Archivist Configuration Guide](../README.md#configuration)

---

## Example Workflow

Here's a complete profiling workflow:

```bash
# 1. Profile your current code
./rph process lib/test_set/ --profile --profile-dir=./baseline

# 2. Analyze results
./scripts/analyze_profile.sh list
./scripts/analyze_profile.sh top

# 3. Open in web UI for detailed analysis
./scripts/analyze_profile.sh web cpu

# 4. Identify bottleneck (e.g., Gemini API calls taking 70% of time)

# 5. Make optimization (enable Redis caching)
# Edit config/config.yaml: cache.enabled = true

# 6. Profile again
./rph process lib/test_set/ --profile --profile-dir=./optimized

# 7. Compare
./scripts/analyze_profile.sh compare baseline/cpu_*.prof optimized/cpu_*.prof

# 8. Verify improvement
./scripts/analyze_profile.sh timings optimized/timings_*.txt
```

---

**Happy Profiling! 🚀**

For questions or issues, please open an issue on [GitHub](https://github.com/yourusername/archivist/issues).
