#!/bin/bash
# profile-memory.sh
# Run engine-ci with memory profiling for different scenarios using actual command structure

set -e

echo "=== Engine-CI Memory Profiling Scenarios ==="
echo "Creating profiles directory..."
mkdir -p profiles

echo ""
echo "=== Scenario 1: Single build with memory profiling ==="
go run --tags containers_image_openpgp main.go run -t build --memprofile=profiles/build.prof --cpuprofile=profiles/build_cpu.prof
echo "✓ Single build profiles created"

echo ""
echo "=== Scenario 2: Build with HTTP pprof endpoint for real-time monitoring ==="
go run --tags containers_image_openpgp main.go run -t build --pprof-http &
BUILD_PID=$!

# Wait for pprof server to start
sleep 2

echo "Collecting profiles during execution..."
# Collect profiles during execution
curl -s http://localhost:6060/debug/pprof/heap > profiles/heap_during_build.prof &
curl -s http://localhost:6060/debug/pprof/allocs > profiles/allocs_during_build.prof &
curl -s "http://localhost:6060/debug/pprof/profile?seconds=30" > profiles/cpu_during_build.prof &

# Wait for build to complete
wait $BUILD_PID
wait  # Wait for all curl commands to complete

echo "✓ Runtime profiles collected"

echo ""
echo "=== Scenario 3: Multiple sequential builds (stress test) ==="
for i in {1..3}; do
    echo "Build iteration $i..."
    go run --tags containers_image_openpgp main.go run -t build --memprofile=profiles/stress_${i}.prof
done
echo "✓ Stress test profiles created"

echo ""
echo "=== Profile Analysis ==="
echo "Generated profiles:"
ls -la profiles/

echo ""
echo "=== Quick heap analysis ==="
if [ -f "profiles/build.prof" ]; then
    echo "Top memory allocators from single build:"
    go tool pprof -top -sample_index=inuse_space profiles/build.prof
fi

echo ""
echo "=== Usage Examples ==="
echo "To analyze profiles interactively:"
echo "  go tool pprof profiles/build.prof"
echo "  go tool pprof -http=:8080 profiles/heap_during_build.prof"
echo ""
echo "To compare allocations:"
echo "  go tool pprof -top -sample_index=alloc_objects profiles/allocs_during_build.prof"
echo ""
echo "Profiling complete! Check the profiles/ directory for results."