# Go pprof Profiling Plan for engine-ci

## Objective
Profile engine-ci using Go's pprof tool to understand actual memory usage patterns and determine if the custom pkg/memory package can be replaced with standard Go memory management.

## Current pkg/memory Usage Analysis

### Usage Locations
1. **pkg/container/container.go**:
   - `memory.WithBufferReturn(memory.HashBuffer, ...)` - SHA256 hashing operations
   - `memory.WithBufferReturn(memory.SmallBuffer, ...)` - Intermediate hash operations
   - `memory.WithBuffer(memory.HashBuffer, ...)` - Concurrent hashing workers
   - `memory.WithBufferReturn(memory.TarBuffer, ...)` - TAR file reading
   - `memory.GetBuffer/PutBuffer(memory.TarBuffer)` - TAR file copying

2. **pkg/container/build.go**:
   - `memory.WithStringBuilder(...)` - String concatenation for image tags

3. **pkg/container/concurrency_bench_test.go**:
   - `memory.GetBufferPoolMetrics()` - Metrics collection (can be removed)
   - `memory.GetPoolMetrics()` - Metrics collection (can be removed)

### Key Patterns
- **Hash operations**: Most buffer usage is for SHA256 hashing (32KB buffers)
- **TAR operations**: Large buffer usage for TAR file operations (1MB buffers)
- **String building**: Only one usage for image tag concatenation
- **Concurrency**: Buffers used in concurrent workers for hashing

## Phase 1: Enable pprof in engine-ci

### 1.1 Add pprof flags to Cobra CLI (cmd/root.go)
```go
// Add to rootCmdArgs struct
type rootCmdArgs struct {
    Progress     string
    Target       string
    Verbose      bool
    // Profiling options
    CPUProfile   string
    MemProfile   string
    PProfHTTP    bool
    PProfPort    int
}

// Add flags in init()
func init() {
    // ... existing flags ...
    rootCmd.PersistentFlags().StringVar(&RootArgs.CPUProfile, "cpuprofile", "", "write cpu profile to file")
    rootCmd.PersistentFlags().StringVar(&RootArgs.MemProfile, "memprofile", "", "write memory profile to file")
    rootCmd.PersistentFlags().BoolVar(&RootArgs.PProfHTTP, "pprof-http", false, "enable HTTP pprof endpoint")
    rootCmd.PersistentFlags().IntVar(&RootArgs.PProfPort, "pprof-port", 6060, "HTTP pprof port")
}
```

### 1.2 Add pprof initialization to PersistentPreRun
```go
// Add to existing PersistentPreRun in root.go after logger setup
PersistentPreRun: func(cmd *cobra.Command, args []string) {
    // ... existing logger setup ...
    
    // Enable CPU profiling if requested
    if RootArgs.CPUProfile != "" {
        f, err := os.Create(RootArgs.CPUProfile)
        if err != nil {
            slog.Error("Could not create CPU profile", "error", err)
            os.Exit(1)
        }
        if err := pprof.StartCPUProfile(f); err != nil {
            slog.Error("Could not start CPU profile", "error", err)
            f.Close()
            os.Exit(1)
        }
        slog.Info("CPU profiling started", "file", RootArgs.CPUProfile)
    }
    
    // Enable HTTP pprof endpoint if requested
    if RootArgs.PProfHTTP {
        go func() {
            addr := fmt.Sprintf("localhost:%d", RootArgs.PProfPort)
            slog.Info("Starting pprof HTTP server", "addr", addr)
            if err := http.ListenAndServe(addr, nil); err != nil {
                slog.Error("pprof server failed", "error", err)
            }
        }()
    }
},
```

### 1.3 Add memory profiling to PersistentPostRun
```go
// Update existing PersistentPostRun in root.go
PersistentPostRun: func(cmd *cobra.Command, args []string) {
    slog.Info("Flushing logs")
    logger.GetLogAggregator().Flush()
    
    // Stop CPU profiling if it was started
    if RootArgs.CPUProfile != "" {
        pprof.StopCPUProfile()
        slog.Info("CPU profiling stopped", "file", RootArgs.CPUProfile)
    }
    
    // Write memory profile if requested
    if RootArgs.MemProfile != "" {
        f, err := os.Create(RootArgs.MemProfile)
        if err != nil {
            slog.Error("Could not create memory profile", "error", err)
            return
        }
        defer f.Close()
        
        runtime.GC() // get up-to-date statistics
        if err := pprof.WriteHeapProfile(f); err != nil {
            slog.Error("Could not write memory profile", "error", err)
        } else {
            slog.Info("Memory profile written", "file", RootArgs.MemProfile)
        }
    }
},
```

### 1.4 Required imports for cmd/root.go
```go
import (
    // ... existing imports ...
    "fmt"
    "net/http"
    _ "net/http/pprof"
    "os"
    "runtime"
    "runtime/pprof"
)
```

## Phase 2: Create Profiling Test Scenarios

### 2.1 Identify Heavy Memory Usage Patterns Based on Current Code
- **Hash Operations**: SHA256 hashing in pkg/container/container.go (uses 32KB HashBuffer)
  - `computeHashFromReader()` - streams large files through hash buffers
  - `combineHashesFromSums()` - processes multiple hash sums
  - `computeHashConcurrently()` - concurrent hashing with multiple workers
- **TAR Operations**: TAR file processing (uses 1MB TarBuffer) 
  - `readFileFromTar()` - reads files from TAR archives
  - `createTarFromDirectory()` - creates TAR archives with large buffers
- **String Building**: Image tag concatenation (uses StringBuilder pool)
  - `BuildKey()` in pkg/container/build.go
- **Real Scenarios to Test**:
  - Building containers with large contexts (many files)
  - Processing large binary files that trigger hash operations
  - Concurrent builds that stress buffer pool contention
  - Multiple TAR operations in parallel

### 2.2 Create Benchmark Scripts
```bash
#!/bin/bash
# profile-memory.sh
# Run engine-ci with memory profiling for different scenarios using actual command structure

# Scenario 1: Single build with memory profiling
go run --tags containers_image_openpgp main.go run -t build --memprofile=build.prof --cpuprofile=build_cpu.prof

# Scenario 2: Multiple concurrent builds (simulate CI load)
for i in {1..4}; do
    go run --tags containers_image_openpgp main.go run -t build --memprofile=build_${i}.prof --pprof-port=$((6060+i)) &
done
wait

# Scenario 3: Build with HTTP pprof endpoint for real-time monitoring
go run --tags containers_image_openpgp main.go run -t build --pprof-http &
BUILD_PID=$!

# Collect profiles during execution
sleep 5  # Let build start
curl http://localhost:6060/debug/pprof/heap > heap_during_build.prof &
curl http://localhost:6060/debug/pprof/allocs > allocs_during_build.prof &
curl http://localhost:6060/debug/pprof/profile?seconds=30 > cpu_during_build.prof &

wait $BUILD_PID

# Scenario 4: Stress test with rapid builds
for i in {1..10}; do
    echo "Build iteration $i"
    go run --tags containers_image_openpgp main.go run -t build
done
```

## Phase 3: Collect and Analyze Profiles

### 3.1 Memory Profile Collection
```bash
# During runtime via HTTP
curl http://localhost:6060/debug/pprof/heap > heap.prof
curl http://localhost:6060/debug/pprof/allocs > allocs.prof

# Analyze allocations
go tool pprof -http=:8080 heap.prof
go tool pprof -http=:8081 allocs.prof
```

### 3.2 Key Metrics to Analyze
1. **Allocation hotspots**: Which functions allocate the most memory?
2. **Object churn**: How many short-lived objects are created?
3. **Memory retention**: What's keeping memory from being GC'd?
4. **String/buffer usage**: Where are strings and buffers most used?

### 3.3 Analysis Commands
```bash
# Top memory allocators
go tool pprof -top heap.prof

# Allocation counts
go tool pprof -top -sample_index=alloc_objects heap.prof

# Cumulative allocations
go tool pprof -cum heap.prof

# Specific function analysis
go tool pprof -list=functionName heap.prof
```

## Phase 4: Compare Custom Pools vs Standard Go

### 4.1 Create A/B Test Branches
1. **Branch A**: Current implementation with pkg/memory pools
2. **Branch B**: Remove pkg/memory, use standard Go

### 4.2 Benchmark Comparisons
```go
// benchmark_test.go
func BenchmarkStringBuildingWithPool(b *testing.B) {
    for i := 0; i < b.N; i++ {
        s := memory.WithStringBuilder(func(sb *strings.Builder) string {
            sb.WriteString("test")
            sb.WriteString("data")
            return sb.String()
        })
        _ = s
    }
}

func BenchmarkStringBuildingStandard(b *testing.B) {
    for i := 0; i < b.N; i++ {
        var sb strings.Builder
        sb.WriteString("test")
        sb.WriteString("data")
        s := sb.String()
        _ = s
    }
}
```

### 4.3 Real-world Performance Tests
```bash
# Time and memory for typical operations using actual command
echo "=== Testing WITH memory pools ==="
/usr/bin/time -v go run --tags containers_image_openpgp main.go run -t build 2>&1 | tee with_pools.log

# Switch to branch without pools
git checkout no-memory-pools

echo "=== Testing WITHOUT memory pools ==="
/usr/bin/time -v go run --tags containers_image_openpgp main.go run -t build 2>&1 | tee without_pools.log

# Compare specific metrics:
echo "=== Memory Usage Comparison ==="
grep "Maximum resident set size" with_pools.log without_pools.log
grep "User time" with_pools.log without_pools.log
grep "System time" with_pools.log without_pools.log
grep "Page faults" with_pools.log without_pools.log

# Benchmark specific operations that use memory pools
echo "=== Benchmarking memory-intensive operations ==="
go test -bench=BenchmarkHashOperations -benchtime=10s -memprofile=hash_with_pools.prof ./pkg/container/
git checkout no-memory-pools
go test -bench=BenchmarkHashOperations -benchtime=10s -memprofile=hash_without_pools.prof ./pkg/container/
```

## Phase 5: Decision Matrix

### 5.1 Criteria for Keeping pkg/memory
- [ ] >20% memory reduction in real workloads
- [ ] >15% performance improvement in benchmarks
- [ ] Significant reduction in GC pressure (>30% fewer GC runs)
- [ ] Clear allocation hotspots that benefit from pooling

### 5.2 Criteria for Removing pkg/memory
- [ ] <10% performance difference
- [ ] Increased code complexity not justified by gains
- [ ] Modern Go GC handles the workload efficiently
- [ ] Maintenance burden outweighs benefits

## Phase 6: Implementation Plan

### 6.1 If Removing pkg/memory
1. **Gradual Migration**:
   - Replace memory.WithStringBuilder with standard strings.Builder
   - Replace buffer pools with byte slices
   - Update all imports and usages
   
2. **Keep Critical Optimizations**:
   - Identify any truly beneficial pools from profiling
   - Consider keeping only those with proven >20% improvement
   
3. **Simplify to Standard Library**:
   ```go
   // Instead of:
   result := memory.WithStringBuilder(func(sb *strings.Builder) string {
       // build string
       return sb.String()
   })
   
   // Use:
   var sb strings.Builder
   // build string
   result := sb.String()
   ```

### 6.2 If Keeping pkg/memory (Optimized)
1. **Remove Unused Features**:
   - Eliminate metrics collection (already done)
   - Remove size-based pools (already done)
   - Keep only pools with proven benefit
   
2. **Optimize Based on Profiling**:
   - Tune pool sizes based on actual usage
   - Add pools for newly identified hotspots
   - Remove pools for rarely reused objects

## Phase 7: Validation and Monitoring

### 7.1 Performance Regression Tests
```go
// Add to CI/CD pipeline
func TestPerformanceRegression(t *testing.T) {
    // Establish baseline metrics
    baseline := runBenchmark("baseline")
    current := runBenchmark("current")
    
    // Allow max 10% regression
    if current.MemoryUsage > baseline.MemoryUsage*1.1 {
        t.Errorf("Memory regression: %v -> %v", baseline.MemoryUsage, current.MemoryUsage)
    }
}
```

### 7.2 Production Monitoring
- Add optional pprof endpoint in production builds
- Collect periodic heap profiles
- Monitor for memory leaks or unexpected growth

## Tools and Resources

### Required Tools
```bash
# Install if needed
go install github.com/google/pprof@latest

# Visualization tools
# graphviz for pprof graphs
apt-get install graphviz  # or brew install graphviz
```

### Profiling Commands Reference
```bash
# CPU profile
go test -cpuprofile cpu.prof -bench .
go tool pprof cpu.prof

# Memory profile
go test -memprofile mem.prof -bench .
go tool pprof mem.prof

# Block profile (contention)
go test -blockprofile block.prof -bench .
go tool pprof block.prof

# All profiles via HTTP
curl http://localhost:6060/debug/pprof/heap
curl http://localhost:6060/debug/pprof/profile?seconds=30
curl http://localhost:6060/debug/pprof/block
curl http://localhost:6060/debug/pprof/mutex
```

## Expected Outcomes

### Best Case: Remove pkg/memory
- Simpler codebase
- Rely on Go's efficient GC
- Less maintenance burden
- Negligible performance impact

### Likely Case: Selective Optimization
- Keep 1-2 critical pools (e.g., hash buffers for image builds)
- Remove everything else
- Document why specific pools are kept
- Add benchmarks to prevent regression

### Worst Case: Keep Current Implementation
- If profiling shows significant benefits
- Document the performance gains
- Add regular profiling to CI/CD
- Consider further optimizations based on data

## Timeline

1. **Week 1**: Implement profiling infrastructure
2. **Week 2**: Run comprehensive profiling scenarios
3. **Week 3**: Analyze results and make decision
4. **Week 4**: Implement changes based on decision
5. **Week 5**: Validate and monitor in staging/production

## Success Metrics

- Memory usage reduction (or acceptable increase if removing pools)
- Build time consistency
- GC pause time improvements
- Code maintainability score improvement
- Zero performance regressions in CI/CD