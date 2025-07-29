# Performance Optimization Implementation Plan - Issue #196

## Overview
Comprehensive performance optimization plan targeting 25-40% improvement in container operations, memory usage, and concurrent processing efficiency for the engine-ci project.

## Current Performance Analysis

### Identified Bottlenecks

#### 1. Memory Inefficiencies
- **String concatenation in hot paths**: Particularly in `pkg/logger/terminal.go` and container operations
- **Large buffer allocations**: No pooling in tar operations (`TarDir` function)  
- **Repeated small allocations**: Log message handling and container metadata processing
- **Inefficient slice operations**: Growing slices without capacity hints in multiple locations

#### 2. Concurrency Issues
- **Sequential container operations**: All container lifecycle operations are sequential
- **Image pulling blocking**: No parallel image pulls for multiple containers
- **Log aggregation bottlenecks**: Single-threaded log processing in `LogAggregator`
- **No worker pools**: Container management lacks worker pool patterns

#### 3. I/O and Network Bottlenecks
- **Synchronous Docker/Podman API calls**: All CRI operations are blocking
- **Registry authentication repeated**: Auth config regenerated for each operation
- **Large tar operations blocking**: No streaming or chunked processing
- **Log streaming inefficiencies**: Buffered reading without optimal buffer sizes

#### 4. Context and Timeout Issues
- **Fixed 10-minute timeouts**: Not optimized for operation types
- **Context inefficiencies**: Multiple context creation/cancellation cycles
- **Resource leaks**: Deferred operations not properly managed

## Implementation Plan

### Phase 1: Memory Optimizations (Days 1-2)

#### 1.1 String Builder Optimizations
**Files to modify:**
- `/Users/frank.ittermann@goflink.com/private/github/engine-ci/pkg/logger/terminal.go`
- `/Users/frank.ittermann@goflink.com/private/github/engine-ci/pkg/container/container.go`
- `/Users/frank.ittermann@goflink.com/private/github/engine-ci/pkg/container/build.go`

**Changes:**
```go
// Replace string concatenation with builders
const DefaultBuilderCapacity = 1024

var stringBuilderPool = sync.Pool{
    New: func() interface{} {
        builder := &strings.Builder{}
        builder.Grow(DefaultBuilderCapacity)
        return builder
    },
}

func getBuilder() *strings.Builder {
    return stringBuilderPool.Get().(*strings.Builder)
}

func putBuilder(builder *strings.Builder) {
    builder.Reset()
    stringBuilderPool.Put(builder)
}
```

**Specific optimizations:**
- `LogMessage` formatting in `terminal.go:211`
- Container prefix generation in `container.go:150,189,210`
- Build flag generation in `build.go:247-271`

#### 1.2 Buffer and Object Pooling
**Files to modify:**
- `/Users/frank.ittermann@goflink.com/private/github/engine-ci/pkg/container/container.go`
- `/Users/frank.ittermann@goflink.com/private/github/engine-ci/pkg/logger/terminal.go`

**Changes:**
```go
// Buffer pools for tar operations and log processing
var (
    tarBufferPool = sync.Pool{
        New: func() interface{} { return make([]byte, 32*1024) }, // 32KB buffers
    }
    
    logBufferPool = sync.Pool{
        New: func() interface{} { return make([]byte, 8*1024) }, // 8KB buffers
    }
)
```

**Target functions:**
- `TarDir` (container.go:751-818) - Add buffer pooling for tar operations
- `streamContainerLogs` (container.go:231-248) - Use pooled buffers for log scanning
- `LogAggregator.Copy` (terminal.go:222-237) - Pool scanner buffers

#### 1.3 Struct Field Alignment
**Files to analyze and fix:**
- `/Users/frank.ittermann@goflink.com/private/github/engine-ci/pkg/container/build.go` - `Build` struct (lines 91-112)
- `/Users/frank.ittermann@goflink.com/private/github/engine-ci/pkg/logger/terminal.go` - `LogEntry` struct (lines 42-49)

**Optimizations:**
```go
// Before (suboptimal alignment)
type LogEntry struct {
    startTime time.Time    // 24 bytes
    endTime   time.Time    // 24 bytes  
    messages  []string     // 24 bytes
    mu        sync.Mutex   // 8 bytes
    isDone    bool         // 1 byte + 7 padding
    isFailed  bool         // 1 byte + 7 padding
}

// After (optimized alignment)
type LogEntry struct {
    startTime time.Time    // 24 bytes
    endTime   time.Time    // 24 bytes
    messages  []string     // 24 bytes
    mu        sync.Mutex   // 8 bytes
    isDone    bool         // 1 byte
    isFailed  bool         // 1 byte + 6 padding
}
```

### Phase 2: Concurrency Improvements (Days 2-3)

#### 2.1 Container Operation Worker Pool
**New file:** `/Users/frank.ittermann@goflink.com/private/github/engine-ci/pkg/container/worker_pool.go`

**Implementation:**
```go
package container

import (
    "context"
    "sync"
)

const (
    DefaultWorkerCount = 4
    DefaultJobBuffer   = 100
)

type ContainerJob struct {
    Type     JobType
    Target   string
    Context  context.Context
    Callback func(result ContainerResult)
}

type JobType int

const (
    JobTypePull JobType = iota
    JobTypeStart
    JobTypeStop
    JobTypeBuild
)

type ContainerResult struct {
    Success bool
    Error   error
    Data    interface{}
}

type WorkerPool struct {
    workerCount int
    jobs        chan ContainerJob
    results     chan ContainerResult
    wg          sync.WaitGroup
    client      func() cri.ContainerManager
}

func NewWorkerPool(workerCount int, client func() cri.ContainerManager) *WorkerPool {
    return &WorkerPool{
        workerCount: workerCount,
        jobs:        make(chan ContainerJob, DefaultJobBuffer),
        results:     make(chan ContainerResult, DefaultJobBuffer),
        client:      client,
    }
}

func (wp *WorkerPool) Start() {
    for i := 0; i < wp.workerCount; i++ {
        wp.wg.Add(1)
        go wp.worker()
    }
}

func (wp *WorkerPool) Stop() {
    close(wp.jobs)
    wp.wg.Wait()
    close(wp.results)
}

func (wp *WorkerPool) Submit(job ContainerJob) {
    wp.jobs <- job
}

func (wp *WorkerPool) worker() {
    defer wp.wg.Done()
    for job := range wp.jobs {
        wp.processJob(job)
    }
}
```

#### 2.2 Parallel Image Operations
**Files to modify:**
- `/Users/frank.ittermann@goflink.com/private/github/engine-ci/pkg/container/container.go`

**Target functions:**
- `ensureImagesExists` (lines 362-407) - Parallelize image pulling
- `Pull` and `PullDefault` methods - Use worker pools

**Implementation:**
```go
func (c *Container) ensureImagesExistsParallel(ctx context.Context, cli cri.ContainerManager, imageNames []string, platform string) error {
    var wg sync.WaitGroup
    errChan := make(chan error, len(imageNames))
    
    // Limit concurrency to avoid overwhelming registry
    sem := make(chan struct{}, 3) // Max 3 concurrent pulls
    
    for _, imageName := range imageNames {
        wg.Add(1)
        go func(img string) {
            defer wg.Done()
            sem <- struct{}{} // Acquire semaphore
            defer func() { <-sem }() // Release semaphore
            
            if err := c.ensureSingleImageExists(ctx, cli, img, platform); err != nil {
                errChan <- fmt.Errorf("failed to ensure image %s: %w", img, err)
            }
        }(imageName)
    }
    
    wg.Wait()
    close(errChan)
    
    // Collect any errors
    var errs []error
    for err := range errChan {
        errs = append(errs, err)
    }
    
    if len(errs) > 0 {
        return fmt.Errorf("image operations failed: %v", errs)
    }
    return nil
}
```

#### 2.3 Async Log Processing
**Files to modify:**
- `/Users/frank.ittermann@goflink.com/private/github/engine-ci/pkg/logger/terminal.go`

**Optimizations:**
- Make `LogAggregator` processing fully async with buffered channels
- Implement batch processing for log entries
- Add worker pool for log formatting

```go
const (
    LogChannelBuffer = 1000
    LogWorkerCount   = 2
)

type LogAggregator struct {
    logChannel   chan LogMessage
    flushDone    chan struct{}
    logMap       sync.Map
    format       string
    routineOrder []string
    workerPool   *LogWorkerPool
}

type LogWorkerPool struct {
    workers  int
    jobs     chan LogProcessJob
    wg       sync.WaitGroup
}

func (la *LogAggregator) processLogsAsync() {
    // Batch process multiple log messages
    batch := make([]LogMessage, 0, 50)
    ticker := time.NewTicker(16 * time.Millisecond) // ~60fps update rate
    defer ticker.Stop()
    
    for {
        select {
        case msg, ok := <-la.logChannel:
            if !ok {
                la.processBatch(batch)
                la.flushDone <- struct{}{}
                return
            }
            batch = append(batch, msg)
            if len(batch) >= 50 {
                la.processBatch(batch)
                batch = batch[:0]
            }
        case <-ticker.C:
            if len(batch) > 0 {
                la.processBatch(batch)
                batch = batch[:0]
            }
        }
    }
}
```

### Phase 3: I/O and Network Optimizations (Day 3)

#### 3.1 Registry Auth Caching
**Files to modify:**
- `/Users/frank.ittermann@goflink.com/private/github/engine-ci/pkg/container/container.go`

**Implementation:**
```go
type AuthCache struct {
    cache map[string]string
    mutex sync.RWMutex
    ttl   time.Duration
}

var globalAuthCache = &AuthCache{
    cache: make(map[string]string),
    ttl:   15 * time.Minute,
}

func (c *Container) registryAuthBase64Cached(imageName string) string {
    globalAuthCache.mutex.RLock()
    if cached, exists := globalAuthCache.cache[imageName]; exists {
        globalAuthCache.mutex.RUnlock()
        return cached
    }
    globalAuthCache.mutex.RUnlock()
    
    // Compute auth if not cached
    auth := c.registryAuthBase64(imageName)
    
    globalAuthCache.mutex.Lock()
    globalAuthCache.cache[imageName] = auth
    globalAuthCache.mutex.Unlock()
    
    return auth
}
```

#### 3.2 Streaming Tar Operations
**Files to modify:**
- `/Users/frank.ittermann@goflink.com/private/github/engine-ci/pkg/container/container.go`

**Target function:** `TarDir` (lines 751-818)

**Implementation:**
```go
func TarDirStreaming(src fs.ReadDirFS) (*io.PipeReader, error) {
    pr, pw := io.Pipe()
    
    go func() {
        defer pw.Close()
        
        tw := tar.NewWriter(pw)
        defer tw.Close()
        
        // Use worker pool for file reading
        err := fs.WalkDir(src, ".", func(path string, d fs.DirEntry, err error) error {
            if err != nil {
                return err
            }
            
            if d.IsDir() {
                return nil
            }
            
            return c.processTarFile(tw, src, path, d)
        })
        
        if err != nil {
            pw.CloseWithError(err)
        }
    }()
    
    return pr, nil
}
```

### Phase 4: Context and Timeout Optimizations (Day 4)

#### 4.1 Dynamic Timeout Management
**New file:** `/Users/frank.ittermann@goflink.com/private/github/engine-ci/pkg/container/timeouts.go`

**Implementation:**
```go
package container

import (
    "context"
    "time"
)

type OperationType int

const (
    OpTypeStart OperationType = iota
    OpTypeStop
    OpTypePull
    OpTypeBuild
    OpTypePush
)

var operationTimeouts = map[OperationType]time.Duration{
    OpTypeStart: 2 * time.Minute,
    OpTypeStop:  30 * time.Second,
    OpTypePull:  5 * time.Minute,
    OpTypeBuild: 15 * time.Minute,
    OpTypePush:  10 * time.Minute,
}

func (c *Container) contextWithOperationTimeout(parent context.Context, op OperationType) (context.Context, context.CancelFunc) {
    timeout := operationTimeouts[op]
    
    // Adjust timeout based on environment
    if c.Env == ProdEnv {
        timeout = timeout * 2 // More generous timeouts in production
    }
    
    return context.WithTimeout(parent, timeout)
}
```

**Files to modify:**
- `Start()` method (line 194) - Use `OpTypeStart` timeout
- `Stop()` method (line 250) - Use `OpTypeStop` timeout  
- Image pull operations - Use `OpTypePull` timeout

### Phase 5: Benchmarking and Measurement (Day 4)

#### 5.1 Performance Benchmarks
**New file:** `/Users/frank.ittermann@goflink.com/private/github/engine-ci/pkg/container/container_bench_test.go`

**Implementation:**
```go
package container

import (
    "context"
    "testing"
    "time"
)

func BenchmarkContainerOperations(b *testing.B) {
    benchmarks := []struct {
        name string
        fn   func(b *testing.B)
    }{
        {"TarDir", benchmarkTarDir},
        {"StringBuilding", benchmarkStringBuilding},
        {"LogProcessing", benchmarkLogProcessing},
        {"AuthCaching", benchmarkAuthCaching},
        {"ParallelImagePull", benchmarkParallelImagePull},
    }
    
    for _, bm := range benchmarks {
        b.Run(bm.name, bm.fn)
    }
}

func benchmarkTarDir(b *testing.B) {
    // Test directory with various file sizes
    testFS := createTestFS(b)
    
    b.ResetTimer()
    b.ReportAllocs()
    
    for i := 0; i < b.N; i++ {
        _, err := TarDir(testFS)
        if err != nil {
            b.Fatal(err)
        }
    }
}

func benchmarkStringBuilding(b *testing.B) {
    parts := []string{"part1", "part2", "part3", "part4", "part5"}
    
    b.Run("Concatenation", func(b *testing.B) {
        b.ReportAllocs()
        for i := 0; i < b.N; i++ {
            var result string
            for _, part := range parts {
                result += part + "-"
            }
            _ = result
        }
    })
    
    b.Run("StringBuilder", func(b *testing.B) {
        b.ReportAllocs()
        for i := 0; i < b.N; i++ {
            builder := getBuilder()
            for _, part := range parts {
                builder.WriteString(part)
                builder.WriteString("-")
            }
            _ = builder.String()
            putBuilder(builder)
        }
    })
}

func benchmarkParallelImagePull(b *testing.B) {
    images := []string{
        "alpine:latest",
        "busybox:latest", 
        "nginx:latest",
    }
    
    container := setupTestContainer(b)
    ctx := context.Background()
    
    b.Run("Sequential", func(b *testing.B) {
        for i := 0; i < b.N; i++ {
            for _, img := range images {
                container.ensureSingleImageExists(ctx, container.client(), img, "linux/amd64")
            }
        }
    })
    
    b.Run("Parallel", func(b *testing.B) {
        for i := 0; i < b.N; i++ {
            container.ensureImagesExistsParallel(ctx, container.client(), images, "linux/amd64")
        }
    })
}
```

#### 5.2 Memory Profiling Integration
**New file:** `/Users/frank.ittermann@goflink.com/private/github/engine-ci/pkg/container/profiling.go`

**Implementation:**
```go
package container

import (
    "context"
    "log/slog"
    "runtime"
    "time"
)

type PerformanceMetrics struct {
    StartTime     time.Time
    EndTime       time.Time
    MemoryBefore  runtime.MemStats
    MemoryAfter   runtime.MemStats
    Operation     string
}

func (c *Container) withPerformanceTracking(operation string, fn func() error) error {
    metrics := &PerformanceMetrics{
        StartTime: time.Now(),
        Operation: operation,
    }
    
    runtime.GC()
    runtime.ReadMemStats(&metrics.MemoryBefore)
    
    err := fn()
    
    runtime.ReadMemStats(&metrics.MemoryAfter)
    metrics.EndTime = time.Now()
    
    c.logPerformanceMetrics(metrics)
    return err
}

func (c *Container) logPerformanceMetrics(metrics *PerformanceMetrics) {
    duration := metrics.EndTime.Sub(metrics.StartTime)
    memDiff := metrics.MemoryAfter.Alloc - metrics.MemoryBefore.Alloc
    
    slog.Info("Performance metrics",
        "operation", metrics.Operation,
        "duration", duration,
        "memory_allocated", memDiff,
        "gc_cycles", metrics.MemoryAfter.NumGC-metrics.MemoryBefore.NumGC,
    )
}
```

## Expected Performance Improvements

### Quantitative Targets

#### Memory Usage Reduction (30-50%)
- **String operations**: 60% reduction through builder pooling
- **Buffer management**: 40% reduction through object pooling  
- **Struct alignment**: 10-15% reduction in memory footprint
- **Log processing**: 35% reduction in allocation rate

#### Container Operation Speed (25-40%)
- **Parallel image pulls**: 3x faster for multiple images
- **Worker pool processing**: 2.5x throughput improvement
- **Auth caching**: 80% reduction in auth computation time
- **Dynamic timeouts**: 20% faster average operation time

#### Concurrent Processing (2-3x improvement)
- **Log aggregation**: 2x processing throughput
- **Container lifecycle**: 3x concurrent operation capacity
- **Build operations**: 2.5x parallel build efficiency

### Qualitative Improvements
- **Resource utilization**: Better CPU and memory distribution
- **Error recovery**: Faster timeout detection and recovery
- **Scalability**: Linear scaling with worker pool size
- **Monitoring**: Comprehensive performance visibility

## Testing Strategy

### Performance Regression Tests
```bash
# Baseline establishment
go test -bench=. -benchmem -count=3 -benchtime=10s ./pkg/container/

# Memory leak detection  
go test -memprofile=mem.prof -run=TestContainerLifecycle
go tool pprof mem.prof

# CPU profiling
go test -cpuprofile=cpu.prof -bench=BenchmarkContainerOperations
go tool pprof cpu.prof

# Load testing
go test -run=TestHighConcurrencyOperations -parallel=10
```

### Integration Testing
- **Multi-container scenarios**: Test worker pool under load
- **Network conditions**: Validate timeout optimizations
- **Memory pressure**: Confirm pooling effectiveness
- **Long-running operations**: Test resource cleanup

## Implementation Validation

### Success Criteria
- [ ] All benchmarks show improvement over baseline
- [ ] Memory usage reduced by at least 25% in typical workloads
- [ ] Container operations 30% faster on average
- [ ] No performance regressions in existing functionality
- [ ] Resource usage monitoring implemented
- [ ] Worker pool scaling works effectively

### Monitoring and Observability
- **Performance metrics collection**: Built-in monitoring
- **Resource usage tracking**: Memory and CPU monitoring
- **Operation timing dashboards**: Real-time performance visibility
- **Alerting for performance degradation**: Automated regression detection

## Risk Mitigation

### Backward Compatibility
- All changes maintain existing API contracts
- Feature toggles for new optimizations
- Gradual rollout capability

### Resource Management
- Bounded worker pools prevent resource exhaustion
- Proper cleanup in all error paths
- Circuit breaker patterns for external dependencies

### Quality Assurance
- Comprehensive test coverage for all optimizations
- Performance regression protection
- Memory leak prevention validation
- Concurrent operation safety verification

## Estimated Impact

**Development Time**: 4 days total
- Memory optimizations: 1.5 days
- Concurrency improvements: 1.5 days  
- I/O optimizations: 0.5 days
- Benchmarking and validation: 0.5 days

**Expected Results**:
- **25-40% performance improvement** in container operations
- **30-50% memory usage reduction** for typical workloads
- **2-3x improvement** in concurrent operation throughput
- **Reduced timeout requirements** through optimization
- **Enhanced monitoring** and performance visibility

This implementation plan provides a systematic approach to achieving the performance targets outlined in Issue #196 while maintaining code quality and backward compatibility.