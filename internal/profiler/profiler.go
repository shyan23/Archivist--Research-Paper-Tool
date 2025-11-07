package profiler

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"sync"
	"time"
)

// Profiler manages CPU and memory profiling
type Profiler struct {
	cpuFile    *os.File
	memFile    *os.File
	outputDir  string
	enabled    bool
	startTime  time.Time
	mu         sync.Mutex

	// Function-level timing
	timings    map[string]*FunctionTiming
	timingsMu  sync.RWMutex
}

// FunctionTiming tracks execution statistics for a function
type FunctionTiming struct {
	Name          string
	CallCount     int
	TotalDuration time.Duration
	MinDuration   time.Duration
	MaxDuration   time.Duration
	AvgDuration   time.Duration
}

// ProfileConfig holds profiling configuration
type ProfileConfig struct {
	Enabled       bool
	OutputDir     string
	CPUProfile    bool
	MemProfile    bool
	FuncTiming    bool   // Track function-level timings
}

// DefaultConfig returns default profiling configuration
func DefaultConfig() *ProfileConfig {
	return &ProfileConfig{
		Enabled:    false,
		OutputDir:  "./profiles",
		CPUProfile: true,
		MemProfile: true,
		FuncTiming: true,
	}
}

// NewProfiler creates a new profiler instance
func NewProfiler(config *ProfileConfig) (*Profiler, error) {
	if !config.Enabled {
		return &Profiler{enabled: false}, nil
	}

	// Create output directory
	if err := os.MkdirAll(config.OutputDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create profile directory: %w", err)
	}

	return &Profiler{
		enabled:   true,
		outputDir: config.OutputDir,
		timings:   make(map[string]*FunctionTiming),
		startTime: time.Now(),
	}, nil
}

// Start begins CPU profiling
func (p *Profiler) Start() error {
	if !p.enabled {
		return nil
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	// Start CPU profiling
	timestamp := time.Now().Format("20060102_150405")
	cpuPath := filepath.Join(p.outputDir, fmt.Sprintf("cpu_%s.prof", timestamp))

	cpuFile, err := os.Create(cpuPath)
	if err != nil {
		return fmt.Errorf("failed to create CPU profile: %w", err)
	}
	p.cpuFile = cpuFile

	if err := pprof.StartCPUProfile(cpuFile); err != nil {
		cpuFile.Close()
		return fmt.Errorf("failed to start CPU profile: %w", err)
	}

	fmt.Printf("\n📊 Profiling started\n")
	fmt.Printf("   CPU Profile: %s\n", cpuPath)

	p.startTime = time.Now()
	return nil
}

// Stop ends profiling and writes memory profile
func (p *Profiler) Stop() error {
	if !p.enabled {
		return nil
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	// Stop CPU profiling
	if p.cpuFile != nil {
		pprof.StopCPUProfile()
		p.cpuFile.Close()
	}

	// Write memory profile
	timestamp := time.Now().Format("20060102_150405")
	memPath := filepath.Join(p.outputDir, fmt.Sprintf("mem_%s.prof", timestamp))

	memFile, err := os.Create(memPath)
	if err != nil {
		return fmt.Errorf("failed to create memory profile: %w", err)
	}
	defer memFile.Close()

	runtime.GC() // Get up-to-date statistics
	if err := pprof.WriteHeapProfile(memFile); err != nil {
		return fmt.Errorf("failed to write memory profile: %w", err)
	}

	duration := time.Since(p.startTime)

	fmt.Printf("\n📊 Profiling stopped (duration: %s)\n", duration.Round(time.Millisecond))
	fmt.Printf("   Memory Profile: %s\n", memPath)

	// Write function timings if available
	if len(p.timings) > 0 {
		timingPath := filepath.Join(p.outputDir, fmt.Sprintf("timings_%s.txt", timestamp))
		if err := p.writeTimingReport(timingPath); err != nil {
			return fmt.Errorf("failed to write timing report: %w", err)
		}
		fmt.Printf("   Timing Report: %s\n", timingPath)
	}

	fmt.Println("\nTo analyze profiles, run:")
	fmt.Printf("   go tool pprof -http=:8080 %s\n", filepath.Join(p.outputDir, fmt.Sprintf("cpu_%s.prof", timestamp)))
	fmt.Printf("   go tool pprof -http=:8080 %s\n", memPath)
	fmt.Println()

	return nil
}

// TimeFunction tracks execution time for a named function
// Usage: defer profiler.TimeFunction("FunctionName")()
func (p *Profiler) TimeFunction(name string) func() {
	if !p.enabled {
		return func() {}
	}

	start := time.Now()
	return func() {
		duration := time.Since(start)
		p.recordTiming(name, duration)
	}
}

// TimeFunctionWithContext tracks execution time and respects context cancellation
func (p *Profiler) TimeFunctionWithContext(ctx context.Context, name string) func() {
	if !p.enabled {
		return func() {}
	}

	start := time.Now()
	return func() {
		select {
		case <-ctx.Done():
			// Context cancelled, record partial timing
			duration := time.Since(start)
			p.recordTiming(fmt.Sprintf("%s (cancelled)", name), duration)
		default:
			duration := time.Since(start)
			p.recordTiming(name, duration)
		}
	}
}

// recordTiming records a function's execution time
func (p *Profiler) recordTiming(name string, duration time.Duration) {
	p.timingsMu.Lock()
	defer p.timingsMu.Unlock()

	timing, exists := p.timings[name]
	if !exists {
		timing = &FunctionTiming{
			Name:        name,
			MinDuration: duration,
			MaxDuration: duration,
		}
		p.timings[name] = timing
	}

	timing.CallCount++
	timing.TotalDuration += duration

	if duration < timing.MinDuration {
		timing.MinDuration = duration
	}
	if duration > timing.MaxDuration {
		timing.MaxDuration = duration
	}

	timing.AvgDuration = timing.TotalDuration / time.Duration(timing.CallCount)
}

// writeTimingReport writes function timing statistics to a file
func (p *Profiler) writeTimingReport(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	fmt.Fprintf(f, "Function Timing Report\n")
	fmt.Fprintf(f, "Generated: %s\n", time.Now().Format("2006-01-02 15:04:05"))
	fmt.Fprintf(f, "Total Duration: %s\n", time.Since(p.startTime).Round(time.Millisecond))
	fmt.Fprintf(f, "\n%s\n", repeatString("=", 100))
	fmt.Fprintf(f, "%-50s %10s %15s %15s %15s %15s\n",
		"FUNCTION", "CALLS", "TOTAL", "AVG", "MIN", "MAX")
	fmt.Fprintf(f, "%s\n", repeatString("-", 100))

	// Sort by total duration (descending)
	type sortedTiming struct {
		name   string
		timing *FunctionTiming
	}

	var sorted []sortedTiming
	p.timingsMu.RLock()
	for name, timing := range p.timings {
		sorted = append(sorted, sortedTiming{name, timing})
	}
	p.timingsMu.RUnlock()

	// Simple bubble sort by total duration
	for i := 0; i < len(sorted)-1; i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[i].timing.TotalDuration < sorted[j].timing.TotalDuration {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}

	// Write sorted timings
	for _, st := range sorted {
		fmt.Fprintf(f, "%-50s %10d %15s %15s %15s %15s\n",
			truncate(st.name, 50),
			st.timing.CallCount,
			st.timing.TotalDuration.Round(time.Millisecond),
			st.timing.AvgDuration.Round(time.Millisecond),
			st.timing.MinDuration.Round(time.Millisecond),
			st.timing.MaxDuration.Round(time.Millisecond))
	}

	fmt.Fprintf(f, "%s\n", repeatString("=", 100))
	return nil
}

// GetTimings returns current timing statistics
func (p *Profiler) GetTimings() map[string]*FunctionTiming {
	p.timingsMu.RLock()
	defer p.timingsMu.RUnlock()

	// Return a copy
	timings := make(map[string]*FunctionTiming)
	for name, timing := range p.timings {
		timings[name] = &FunctionTiming{
			Name:          timing.Name,
			CallCount:     timing.CallCount,
			TotalDuration: timing.TotalDuration,
			MinDuration:   timing.MinDuration,
			MaxDuration:   timing.MaxDuration,
			AvgDuration:   timing.AvgDuration,
		}
	}
	return timings
}

// PrintTimings prints timing statistics to stdout
func (p *Profiler) PrintTimings() {
	if !p.enabled {
		return
	}

	p.timingsMu.RLock()
	defer p.timingsMu.RUnlock()

	fmt.Println("\n╔════════════════════════════════════════════════════════════════╗")
	fmt.Println("║              FUNCTION TIMING REPORT                            ║")
	fmt.Println("╚════════════════════════════════════════════════════════════════╝")
	fmt.Println()

	// Sort by total duration
	type sortedTiming struct {
		name   string
		timing *FunctionTiming
	}

	var sorted []sortedTiming
	for name, timing := range p.timings {
		sorted = append(sorted, sortedTiming{name, timing})
	}

	// Simple bubble sort
	for i := 0; i < len(sorted)-1; i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[i].timing.TotalDuration < sorted[j].timing.TotalDuration {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}

	fmt.Printf("%-45s %8s %12s %12s %12s %12s\n",
		"FUNCTION", "CALLS", "TOTAL", "AVG", "MIN", "MAX")
	fmt.Println(string(make([]byte, 105)) + "\n" + string(make([]byte, 105)))

	for _, st := range sorted {
		fmt.Printf("%-45s %8d %12s %12s %12s %12s\n",
			truncate(st.name, 45),
			st.timing.CallCount,
			st.timing.TotalDuration.Round(time.Millisecond),
			st.timing.AvgDuration.Round(time.Millisecond),
			st.timing.MinDuration.Round(time.Millisecond),
			st.timing.MaxDuration.Round(time.Millisecond))
	}
	fmt.Println()
}

// Helper functions
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// repeatString repeats a string n times
func repeatString(s string, n int) string {
	result := ""
	for i := 0; i < n; i++ {
		result += s
	}
	return result
}

// MemoryStats returns current memory statistics
func MemoryStats() runtime.MemStats {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return m
}

// PrintMemoryStats prints current memory usage
func PrintMemoryStats() {
	m := MemoryStats()
	fmt.Println("\n╔════════════════════════════════════════════════════════════════╗")
	fmt.Println("║                  MEMORY STATISTICS                             ║")
	fmt.Println("╚════════════════════════════════════════════════════════════════╝")
	fmt.Println()
	fmt.Printf("  Allocated:       %6d MB\n", m.Alloc/1024/1024)
	fmt.Printf("  Total Allocated: %6d MB\n", m.TotalAlloc/1024/1024)
	fmt.Printf("  System Memory:   %6d MB\n", m.Sys/1024/1024)
	fmt.Printf("  GC Runs:         %6d\n", m.NumGC)
	fmt.Printf("  Goroutines:      %6d\n", runtime.NumGoroutine())
	fmt.Println()
}
