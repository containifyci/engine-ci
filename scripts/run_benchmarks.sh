#!/bin/bash

# Engine-CI Performance Benchmark Runner
# =====================================

set -e  # Exit on error

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
BENCHMARK_DIR="benchmarks"
RESULTS_DIR="$BENCHMARK_DIR/results"
BASELINE_FILE="$RESULTS_DIR/baseline.json"

echo -e "${BLUE}🚀 Engine-CI Performance Benchmark Suite${NC}"
echo "==========================================="

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo -e "${RED}❌ Go is not installed or not in PATH${NC}"
    exit 1
fi

# Create directories if they don't exist
mkdir -p "$RESULTS_DIR"

# Function to run benchmarks for a specific package
run_package_benchmarks() {
    local package=$1
    local package_name=$(basename "$package")
    
    echo -e "${BLUE}🔍 Running benchmarks for $package_name...${NC}"
    
    # Check if benchmark files exist
    if ! ls "$package"/*_bench_test.go 1> /dev/null 2>&1; then
        echo -e "${YELLOW}⚠️  No benchmark files found in $package${NC}"
        return 0
    fi
    
    # Run benchmarks with detailed output
    echo "   Running go test -bench=. -benchmem -benchtime=3s $package"
    
    if go test -bench=. -benchmem -benchtime=3s "$package" > /tmp/bench_output.txt 2>&1; then
        echo -e "${GREEN}   ✅ Benchmarks completed successfully${NC}"
        
        # Show a summary of the benchmarks
        local bench_count=$(grep -c "^Benchmark" /tmp/bench_output.txt || echo "0")
        echo "   📊 $bench_count benchmarks executed"
        
        # Show top 3 slowest operations
        echo "   🐌 Slowest operations:"
        grep "^Benchmark" /tmp/bench_output.txt | \
            sort -k3 -nr | \
            head -3 | \
            awk '{printf "      %s: %.2f ms\n", $1, $3/1000000}' || echo "      No benchmark data found"
    else
        echo -e "${RED}   ❌ Benchmarks failed${NC}"
        echo "   Error output:"
        cat /tmp/bench_output.txt | head -10
        return 1
    fi
    
    echo ""
}

# Function to run comprehensive benchmark suite
run_full_suite() {
    echo -e "${BLUE}📋 Running comprehensive benchmark suite...${NC}"
    echo ""
    
    # Define packages to benchmark
    local packages=(
        "./pkg/container"
        "./pkg/logger"
        "./pkg/cri"
    )
    
    local failed_packages=0
    local total_packages=${#packages[@]}
    
    # Run benchmarks for each package
    for package in "${packages[@]}"; do
        if ! run_package_benchmarks "$package"; then
            ((failed_packages++))
        fi
    done
    
    echo -e "${BLUE}📊 Package Summary:${NC}"
    echo "   Total packages: $total_packages"
    echo "   Successful: $((total_packages - failed_packages))"
    if [ $failed_packages -gt 0 ]; then
        echo -e "   ${RED}Failed: $failed_packages${NC}"
    else
        echo -e "   ${GREEN}Failed: $failed_packages${NC}"
    fi
    echo ""
}

# Function to run the benchmark runner tool
run_benchmark_tool() {
    echo -e "${BLUE}🔧 Running benchmark analysis tool...${NC}"
    
    if [ -f "$BENCHMARK_DIR/benchmark_runner.go" ]; then
        cd "$BENCHMARK_DIR"
        go run benchmark_runner.go
        cd - > /dev/null
    else
        echo -e "${RED}❌ Benchmark runner tool not found${NC}"
        return 1
    fi
}

# Function to run regression tests
run_regression_tests() {
    echo -e "${BLUE}🔍 Running performance regression tests...${NC}"
    
    if [ ! -f "$BASELINE_FILE" ]; then
        echo -e "${YELLOW}⚠️  No baseline found, skipping regression tests${NC}"
        echo "   Run with --baseline first to establish a baseline"
        return 0
    fi
    
    if [ -f "$BENCHMARK_DIR/regression_tests.go" ]; then
        cd "$BENCHMARK_DIR"
        go run regression_tests.go
        cd - > /dev/null
    else
        echo -e "${RED}❌ Regression test tool not found${NC}"
        return 1
    fi
}

# Function to create or update baseline
create_baseline() {
    echo -e "${BLUE}📊 Creating performance baseline...${NC}"
    
    if [ -f "$BASELINE_FILE" ]; then
        echo -e "${YELLOW}⚠️  Baseline already exists${NC}"
        read -p "Do you want to update it? (y/N): " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            echo "Baseline creation cancelled"
            return 0
        fi
        
        # Backup existing baseline
        local backup_file="$RESULTS_DIR/baseline_backup_$(date +%Y%m%d_%H%M%S).json"
        cp "$BASELINE_FILE" "$backup_file"
        echo "   💾 Existing baseline backed up to: $backup_file"
    fi
    
    # Run the benchmark tool which will create baseline if it doesn't exist
    run_benchmark_tool
    
    if [ -f "$BASELINE_FILE" ]; then
        echo -e "${GREEN}✅ Baseline created successfully${NC}"
        echo "   📍 Baseline saved to: $BASELINE_FILE"
    else
        echo -e "${RED}❌ Failed to create baseline${NC}"
        return 1
    fi
}

# Function to show system information
show_system_info() {
    echo -e "${BLUE}💻 System Information:${NC}"
    echo "   Go Version: $(go version | cut -d' ' -f3-)"
    echo "   OS: $(uname -s) $(uname -r)"
    echo "   Architecture: $(uname -m)"
    echo "   CPU Cores: $(nproc 2>/dev/null || sysctl -n hw.ncpu 2>/dev/null || echo 'unknown')"
    
    if command -v git &> /dev/null; then
        echo "   Git Commit: $(git rev-parse --short HEAD 2>/dev/null || echo 'unknown')"
        echo "   Git Branch: $(git branch --show-current 2>/dev/null || echo 'unknown')"
    fi
    echo ""
}

# Function to clean old results
clean_results() {
    echo -e "${BLUE}🧹 Cleaning old benchmark results...${NC}"
    
    if [ -d "$RESULTS_DIR" ]; then
        local file_count=$(find "$RESULTS_DIR" -name "*.json" | wc -l)
        if [ "$file_count" -gt 0 ]; then
            echo "   Found $file_count result files"
            read -p "Delete all result files except baseline? (y/N): " -n 1 -r
            echo
            if [[ $REPLY =~ ^[Yy]$ ]]; then
                find "$RESULTS_DIR" -name "*.json" ! -name "baseline.json" -delete
                echo -e "${GREEN}   ✅ Old results cleaned${NC}"
            else
                echo "   Cleaning cancelled"
            fi
        else
            echo "   No result files to clean"
        fi
    else
        echo "   Results directory doesn't exist"
    fi
}

# Function to show usage
show_usage() {
    echo "Usage: $0 [OPTIONS]"
    echo ""
    echo "Options:"
    echo "  --baseline, -b     Create or update performance baseline"
    echo "  --full, -f         Run full benchmark suite (default)"
    echo "  --regression, -r   Run regression tests against baseline"
    echo "  --clean, -c        Clean old benchmark results"
    echo "  --info, -i         Show system information"
    echo "  --help, -h         Show this help message"
    echo ""
    echo "Examples:"
    echo "  $0                 # Run full benchmark suite with analysis"
    echo "  $0 --baseline      # Create performance baseline"
    echo "  $0 --regression    # Run regression tests"
    echo "  $0 --clean         # Clean old results"
}

# Parse command line arguments
case "${1:-}" in
    --baseline|-b)
        show_system_info
        create_baseline
        ;;
    --regression|-r)
        show_system_info
        run_regression_tests
        ;;
    --clean|-c)
        clean_results
        ;;
    --info|-i)
        show_system_info
        ;;
    --help|-h)
        show_usage
        ;;
    --full|-f|"")
        show_system_info
        run_full_suite
        run_benchmark_tool
        
        # Run regression tests if baseline exists
        if [ -f "$BASELINE_FILE" ]; then
            echo ""
            run_regression_tests
        fi
        ;;
    *)
        echo -e "${RED}❌ Unknown option: $1${NC}"
        echo ""
        show_usage
        exit 1
        ;;
esac

echo -e "${GREEN}🎉 Benchmark operation completed!${NC}"