# Engine-CI Performance Benchmarks

This directory contains comprehensive performance benchmarks and regression testing infrastructure for the engine-ci project, designed to support Issue #196 performance optimizations.

## 🎯 Performance Targets

Based on the analysis, we're targeting the following improvements:

### Container Operations (`pkg/container/container.go`)
- **Target**: 25-40% reduction in container operation time
- **Key Areas**: 
  - Container creation and initialization
  - Image tag parsing
  - Checksum computation
  - Tar directory operations

### Log Aggregation (`pkg/logger/terminal.go`)
- **Target**: 30-50% reduction in memory usage for large builds
- **Key Areas**:
  - Log message buffering and aggregation
  - Concurrent log processing
  - String manipulation operations

### Build Configuration (`pkg/container/build.go`)
- **Target**: 60% reduction in string operation allocations
- **Key Areas**:
  - Flag string building
  - Configuration serialization
  - Custom map operations

### Container Runtime Management (`pkg/cri/manager.go`)
- **Target**: 2-3x improvement in concurrent operation throughput
- **Key Areas**:
  - Runtime detection
  - Manager initialization
  - Singleton access patterns

## 📁 Directory Structure

```
benchmarks/
├── README.md                 # This file
├── benchmark_runner.go       # Main benchmark execution and analysis tool
├── regression_tests.go       # Performance regression testing framework
├── results/                  # Benchmark results storage
│   ├── baseline.json        # Performance baseline for comparisons
│   ├── benchmark_*.json     # Individual benchmark runs
│   └── regression_*.json    # Regression test results
└── scripts/
    └── run_benchmarks.sh    # Convenience script for running benchmarks
```

## 🚀 Quick Start

### 1. Create Performance Baseline

First, establish a baseline of current performance:

```bash
# Run from project root
./scripts/run_benchmarks.sh --baseline
```

This will:
- Run all benchmark suites
- Generate performance analysis
- Save baseline results for future comparisons

### 2. Run Full Benchmark Suite

```bash
# Run comprehensive benchmarks with analysis
./scripts/run_benchmarks.sh

# Or explicitly
./scripts/run_benchmarks.sh --full
```

### 3. Run Regression Tests

After making performance changes:

```bash
./scripts/run_benchmarks.sh --regression
```

This will compare current performance against the baseline and flag any regressions.

## 📊 Benchmark Categories

### Container Operations Benchmarks
- **New Container Creation**: Tests container initialization overhead
- **Parse Image Tag**: Benchmarks image string parsing efficiency
- **Checksum Computation**: Tests cryptographic operations on various data sizes
- **Tar Operations**: Benchmarks directory archiving with different file sizes/counts
- **String Operations**: Tests frequently-called string manipulation functions
- **Concurrent Operations**: Benchmarks thread-safe operations under load

### Logger Benchmarks
- **LogAggregator Creation**: Tests singleton initialization overhead
- **Message Logging**: Benchmarks single vs. multiple routine logging
- **Progress Format**: Tests real-time log display performance
- **LogEntry Operations**: Benchmarks message buffering and overflow handling
- **Concurrent Logging**: Tests thread-safe logging under high concurrency
- **I/O Operations**: Benchmarks write and copy operations with various data sizes

### Build Configuration Benchmarks
- **Build Creation**: Tests build configuration initialization
- **String Building**: Benchmarks AsFlags() and similar operations
- **Custom Map Operations**: Tests configuration parameter access
- **Type Conversions**: Benchmarks string-to-type conversions
- **Complex Build Operations**: Tests full build configuration scenarios

### Container Runtime Manager Benchmarks
- **Runtime Detection**: Tests container runtime discovery
- **Manager Initialization**: Benchmarks singleton initialization patterns
- **Concurrent Access**: Tests thread-safe manager access
- **Factory Operations**: Benchmarks manager creation patterns

## 🔍 Benchmark Analysis Features

### Performance Metrics
- **Execution Time**: Nanoseconds per operation
- **Memory Allocations**: Allocations per operation
- **Memory Usage**: Bytes allocated per operation
- **Throughput**: Operations per second (where applicable)

### Automated Analysis
- **Performance Flags**: Automatically flags operations exceeding thresholds:
  - Slow operations (>1ms per operation)
  - High allocation operations (>1000 allocs/op)
  - High memory operations (>10KB per operation)
- **Top Performers**: Identifies slowest operations and highest allocators
- **Regression Detection**: Compares against baseline with configurable thresholds

### Reporting
- **JSON Output**: Machine-readable results for CI/CD integration
- **Human-Readable Reports**: Formatted console output with summaries
- **Historical Tracking**: Maintains history of performance changes

## 🧪 Regression Testing

### Critical Performance Tests

The regression testing framework includes predefined tests for critical performance paths:

#### Container Operations (Critical)
- **New Container Creation**: Max 15% performance regression
- **Parse Image Tag**: Max 20% performance regression  
- **Checksum Computation**: No allocation increases allowed

#### String Operations (Critical)
- **Safe Short Operation**: Max 10% performance regression, no new allocations
- **Concurrent Image Tag Parsing**: Max 15% performance regression

#### Logger Operations (Critical)
- **Single Routine Logging**: Max 15% performance regression
- **Multiple Routine Logging**: Max 25% performance regression
- **Singleton Access**: Max 5% performance regression, no new allocations

### Customizable Thresholds

Each regression test can be configured with:
- **Max Performance Drop**: Percentage threshold for acceptable regression
- **Max Allocation Increase**: Absolute increase in allocations per operation
- **Max Memory Increase**: Absolute increase in bytes per operation
- **Critical Flag**: Whether failures should cause CI/CD pipeline failures

## 📈 Performance Optimization Workflow

### 1. Establish Baseline
```bash
./scripts/run_benchmarks.sh --baseline
```

### 2. Make Performance Changes
Implement optimizations targeting specific bottlenecks identified in the analysis.

### 3. Run Benchmarks
```bash
./scripts/run_benchmarks.sh
```

### 4. Analyze Results
Review the automated analysis for:
- Performance improvements vs. targets
- Any unexpected regressions
- Memory allocation changes

### 5. Run Regression Tests
```bash
./scripts/run_benchmarks.sh --regression
```

### 6. Iterate
Repeat the process, using previous results to guide further optimizations.

## 🔧 Integration with CI/CD

### GitHub Actions Integration

Add to your workflow:

```yaml
- name: Run Performance Benchmarks
  run: |
    ./scripts/run_benchmarks.sh --full
    
- name: Check for Performance Regressions
  run: |
    ./scripts/run_benchmarks.sh --regression
```

### Performance Monitoring

The benchmark results can be integrated with monitoring systems:
- **Metrics**: Export performance metrics to Prometheus/Grafana
- **Alerts**: Set up alerts for performance regressions
- **Dashboards**: Create dashboards showing performance trends over time

## 📋 Benchmark Interpretation Guide

### Understanding Results

```
BenchmarkContainerOperations/New_Container_Creation-8    5000000    292 ns/op    48 B/op    2 allocs/op
```

- **5000000**: Number of iterations run
- **292 ns/op**: Average nanoseconds per operation
- **48 B/op**: Average bytes allocated per operation
- **2 allocs/op**: Average number of allocations per operation

### Performance Targets
- **Sub-millisecond**: Critical path operations should complete in <1ms
- **Low Allocation**: Frequently called functions should minimize allocations
- **Memory Efficiency**: Large operations should maintain reasonable memory usage
- **Scalability**: Concurrent operations should scale with available cores

### Red Flags
- **High Allocation Count**: >1000 allocations per operation
- **Large Memory Usage**: >10KB per operation for simple functions
- **Slow Operations**: >1ms for frequently called functions
- **Poor Concurrency**: Performance doesn't improve with parallel execution

## 🛠️ Extending Benchmarks

### Adding New Benchmarks

1. Create benchmark functions in `*_bench_test.go` files
2. Follow Go benchmark naming conventions: `BenchmarkXxx(*testing.B)`
3. Use `b.ResetTimer()` and `b.ReportAllocs()` appropriately
4. Add regression tests to `regression_tests.go` if critical

### Benchmark Best Practices

- **Realistic Data**: Use representative data sizes and patterns
- **Warm-up**: Allow for JIT compilation and cache warming
- **Isolation**: Test one thing at a time
- **Repeatability**: Ensure consistent results across runs
- **Memory Profiling**: Use `b.ReportAllocs()` for memory-sensitive operations

## 📚 Additional Resources

- [Go Benchmark Documentation](https://golang.org/pkg/testing/#hdr-Benchmarks)
- [Performance Optimization Guide](https://github.com/golang/go/wiki/Performance)
- [Memory Profiling](https://blog.golang.org/pprof)

## 🤝 Contributing

When contributing performance optimizations:

1. **Benchmark First**: Always establish current performance with benchmarks
2. **Measure Changes**: Use benchmarks to validate improvements
3. **Regression Test**: Ensure changes don't break existing performance
4. **Document**: Update benchmarks and documentation for new critical paths

## 📞 Support

For questions about the benchmark infrastructure or performance optimization:

1. Check existing benchmark results in `benchmarks/results/`
2. Review regression test definitions in `regression_tests.go`
3. Examine benchmark implementations in `*_bench_test.go` files
4. Create an issue with benchmark results and specific questions