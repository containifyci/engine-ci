#!/bin/bash
# compare-memory-pools.sh
# Compare performance with and without memory pools

set -e

CURRENT_BRANCH=$(git branch --show-current)
echo "=== Memory Pool Performance Comparison ==="
echo "Current branch: $CURRENT_BRANCH"
echo "Creating comparison results directory..."
mkdir -p comparison

echo ""
echo "=== Phase 1: Testing WITH memory pools (current implementation) ==="
echo "Building and profiling current implementation..."
/usr/bin/time -v go run --tags containers_image_openpgp main.go run -t build --memprofile=comparison/with_pools_mem.prof --cpuprofile=comparison/with_pools_cpu.prof 2>&1 | tee comparison/with_pools.log

echo ""
echo "=== Phase 2: Benchmarking memory-intensive operations WITH pools ==="
go test -bench=. -benchtime=5s -memprofile=comparison/bench_with_pools.prof -cpuprofile=comparison/bench_with_pools_cpu.prof ./pkg/container/ 2>&1 | tee comparison/bench_with_pools.log

echo ""
echo "=== Phase 3: Memory usage analysis WITH pools ==="
echo "Memory statistics from /usr/bin/time:"
grep -E "(Maximum resident set size|User time|System time|Page faults)" comparison/with_pools.log > comparison/with_pools_stats.txt
cat comparison/with_pools_stats.txt

echo ""
echo "=== Memory Pool Usage Analysis ==="
if [ -f "comparison/with_pools_mem.prof" ]; then
    echo "Top memory allocators WITH pools:"
    go tool pprof -top -sample_index=inuse_space comparison/with_pools_mem.prof | head -20 > comparison/with_pools_top_allocators.txt
    cat comparison/with_pools_top_allocators.txt
fi

echo ""
echo "=== Next Steps ==="
echo "1. Create a branch without memory pools:"
echo "   git checkout -b no-memory-pools"
echo "   # Remove pkg/memory usage and replace with standard Go"
echo ""
echo "2. Run this comparison script again to compare:"
echo "   ./scripts/compare-memory-pools.sh"
echo ""
echo "3. Compare results:"
echo "   diff comparison/with_pools_stats.txt comparison/without_pools_stats.txt"
echo ""
echo "Current results saved in comparison/ directory:"
ls -la comparison/

echo ""
echo "=== Analysis Commands ==="
echo "Interactive profiling:"
echo "  go tool pprof -http=:8080 comparison/with_pools_mem.prof"
echo ""
echo "Compare CPU usage:"
echo "  go tool pprof comparison/with_pools_cpu.prof"
echo ""
echo "Benchmark comparison:"
echo "  go tool pprof comparison/bench_with_pools.prof"