package benchmarks

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// BenchmarkResult represents a single benchmark result
type BenchmarkResult struct {
	Name        string  `json:"name"`
	Iterations  int64   `json:"iterations"`
	NsPerOp     float64 `json:"ns_per_op"`
	MBPerSec    float64 `json:"mb_per_sec,omitempty"`
	AllocsPerOp int64   `json:"allocs_per_op,omitempty"`
	BytesPerOp  int64   `json:"bytes_per_op,omitempty"`
	Package     string  `json:"package"`
	Timestamp   string  `json:"timestamp"`
}

// BenchmarkSuite represents a collection of benchmark results
type BenchmarkSuite struct {
	Timestamp string            `json:"timestamp"`
	GoVersion string            `json:"go_version"`
	GitCommit string            `json:"git_commit"`
	Results   []BenchmarkResult `json:"results"`
	Summary   BenchmarkSummary  `json:"summary"`
}

// BenchmarkSummary provides high-level statistics
type BenchmarkSummary struct {
	TotalBenchmarks   int               `json:"total_benchmarks"`
	PackageBreakdown  map[string]int    `json:"package_breakdown"`
	PerformanceFlags  []string          `json:"performance_flags"`
	TopSlowestOps     []BenchmarkResult `json:"top_slowest_ops"`
	HighestAllocators []BenchmarkResult `json:"highest_allocators"`
}

const (
	// Performance thresholds for flagging
	SlowOperationThresholdNs = 1000000 // 1ms
	HighAllocationThreshold  = 1000    // 1000 allocs per op
	HighMemoryThreshold      = 10000   // 10KB per op
)

// RunBenchmarkSuite runs the complete benchmark suite
func RunBenchmarkSuite() {
	fmt.Println("🚀 Engine-CI Performance Benchmark Suite")
	fmt.Println("==========================================")

	// Get current timestamp
	timestamp := time.Now().Format("2006-01-02_15-04-05")

	// Create benchmarks directory if it doesn't exist
	if err := os.MkdirAll("benchmarks/results", 0755); err != nil {
		fmt.Printf("❌ Failed to create benchmarks directory: %v\n", err)
		os.Exit(1)
	}

	// Run benchmarks
	suite := BenchmarkSuite{
		Timestamp: timestamp,
		Results:   []BenchmarkResult{},
	}

	// Get Go version
	suite.GoVersion = getGoVersion()

	// Get Git commit
	suite.GitCommit = getGitCommit()

	fmt.Printf("📋 Go Version: %s\n", suite.GoVersion)
	fmt.Printf("📋 Git Commit: %s\n", suite.GitCommit)
	fmt.Printf("📋 Timestamp: %s\n", suite.Timestamp)
	fmt.Println()

	// Define benchmark packages
	packages := []string{
		"./pkg/container",
		"./pkg/logger",
		"./pkg/cri",
	}

	// Run benchmarks for each package
	for _, pkg := range packages {
		fmt.Printf("🔍 Running benchmarks for %s...\n", pkg)
		results := runPackageBenchmarks(pkg)

		for _, result := range results {
			result.Package = pkg
			result.Timestamp = timestamp
			suite.Results = append(suite.Results, result)
		}

		fmt.Printf("✅ Completed %d benchmarks for %s\n", len(results), pkg)
	}

	// Generate summary
	suite.Summary = generateSummary(suite.Results)

	// Save results
	resultsFile := filepath.Join("benchmarks/results", fmt.Sprintf("benchmark_%s.json", timestamp))
	if err := saveResults(suite, resultsFile); err != nil {
		fmt.Printf("❌ Failed to save results: %v\n", err)
		os.Exit(1)
	}

	// Save baseline if this is the first run
	baselineFile := "benchmarks/results/baseline.json"
	if _, err := os.Stat(baselineFile); os.IsNotExist(err) {
		if err := saveResults(suite, baselineFile); err != nil {
			fmt.Printf("⚠️  Failed to save baseline: %v\n", err)
		} else {
			fmt.Printf("📊 Saved baseline results to %s\n", baselineFile)
		}
	}

	// Print summary
	printSummary(suite)

	// Compare with baseline if it exists
	if baseline, err := loadResults(baselineFile); err == nil {
		fmt.Println("\n📈 Comparison with Baseline")
		fmt.Println("===========================")
		compareWithBaseline(suite, baseline)
	}

	fmt.Printf("\n💾 Results saved to: %s\n", resultsFile)
	fmt.Println("🎉 Benchmark suite completed successfully!")
}

func runPackageBenchmarks(pkg string) []BenchmarkResult {
	// Run go test -bench with detailed output
	cmd := exec.Command("go", "test", "-bench=.", "-benchmem", "-benchtime=1s", pkg)

	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("⚠️  Warning: Failed to run benchmarks for %s: %v\n", pkg, err)
		fmt.Printf("Output: %s\n", string(output))
		return []BenchmarkResult{}
	}

	return parseBenchmarkOutput(string(output))
}

func parseBenchmarkOutput(output string) []BenchmarkResult {
	var results []BenchmarkResult

	// Regular expression to parse benchmark lines
	// Example: BenchmarkContainerOperations/New_Container_Creation-8         5000000       292 ns/op      48 B/op       2 allocs/op
	re := regexp.MustCompile(`Benchmark(\S+)\s+(\d+)\s+(\d+(?:\.\d+)?)\s+ns/op(?:\s+(\d+(?:\.\d+)?)\s+MB/s)?(?:\s+(\d+)\s+B/op)?(?:\s+(\d+)\s+allocs/op)?`)

	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()

		if matches := re.FindStringSubmatch(line); matches != nil {
			name := matches[1]
			iterations, _ := strconv.ParseInt(matches[2], 10, 64)
			nsPerOp, _ := strconv.ParseFloat(matches[3], 64)

			result := BenchmarkResult{
				Name:       name,
				Iterations: iterations,
				NsPerOp:    nsPerOp,
			}

			// Parse optional MB/s
			if matches[4] != "" {
				result.MBPerSec, _ = strconv.ParseFloat(matches[4], 64)
			}

			// Parse optional bytes per op
			if matches[5] != "" {
				result.BytesPerOp, _ = strconv.ParseInt(matches[5], 10, 64)
			}

			// Parse optional allocs per op
			if matches[6] != "" {
				result.AllocsPerOp, _ = strconv.ParseInt(matches[6], 10, 64)
			}

			results = append(results, result)
		}
	}

	return results
}

func generateSummary(results []BenchmarkResult) BenchmarkSummary {
	summary := BenchmarkSummary{
		TotalBenchmarks:   len(results),
		PackageBreakdown:  make(map[string]int),
		PerformanceFlags:  []string{},
		TopSlowestOps:     []BenchmarkResult{},
		HighestAllocators: []BenchmarkResult{},
	}

	// Package breakdown
	for _, result := range results {
		summary.PackageBreakdown[result.Package]++
	}

	// Find slow operations
	var slowOps []BenchmarkResult
	var highAllocOps []BenchmarkResult

	for _, result := range results {
		// Flag slow operations
		if result.NsPerOp > SlowOperationThresholdNs {
			summary.PerformanceFlags = append(summary.PerformanceFlags,
				fmt.Sprintf("SLOW: %s (%.2fms)", result.Name, result.NsPerOp/1000000))
			slowOps = append(slowOps, result)
		}

		// Flag high allocation operations
		if result.AllocsPerOp > HighAllocationThreshold {
			summary.PerformanceFlags = append(summary.PerformanceFlags,
				fmt.Sprintf("HIGH_ALLOC: %s (%d allocs/op)", result.Name, result.AllocsPerOp))
			highAllocOps = append(highAllocOps, result)
		}

		// Flag high memory operations
		if result.BytesPerOp > HighMemoryThreshold {
			summary.PerformanceFlags = append(summary.PerformanceFlags,
				fmt.Sprintf("HIGH_MEM: %s (%.2fKB/op)", result.Name, float64(result.BytesPerOp)/1024))
		}
	}

	// Sort and get top 5 slowest
	sort.Slice(slowOps, func(i, j int) bool {
		return slowOps[i].NsPerOp > slowOps[j].NsPerOp
	})
	if len(slowOps) > 5 {
		summary.TopSlowestOps = slowOps[:5]
	} else {
		summary.TopSlowestOps = slowOps
	}

	// Sort and get top 5 highest allocators
	sort.Slice(highAllocOps, func(i, j int) bool {
		return highAllocOps[i].AllocsPerOp > highAllocOps[j].AllocsPerOp
	})
	if len(highAllocOps) > 5 {
		summary.HighestAllocators = highAllocOps[:5]
	} else {
		summary.HighestAllocators = highAllocOps
	}

	return summary
}

func saveResults(suite BenchmarkSuite, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(suite)
}

func loadResults(filename string) (BenchmarkSuite, error) {
	var suite BenchmarkSuite

	file, err := os.Open(filename)
	if err != nil {
		return suite, err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	err = decoder.Decode(&suite)
	return suite, err
}

func printSummary(suite BenchmarkSuite) {
	fmt.Println("\n📊 Benchmark Summary")
	fmt.Println("====================")
	fmt.Printf("Total Benchmarks: %d\n", suite.Summary.TotalBenchmarks)

	fmt.Println("\nPackage Breakdown:")
	for pkg, count := range suite.Summary.PackageBreakdown {
		fmt.Printf("  %s: %d benchmarks\n", pkg, count)
	}

	if len(suite.Summary.PerformanceFlags) > 0 {
		fmt.Println("\n⚠️  Performance Flags:")
		for _, flag := range suite.Summary.PerformanceFlags {
			fmt.Printf("  %s\n", flag)
		}
	}

	if len(suite.Summary.TopSlowestOps) > 0 {
		fmt.Println("\n🐌 Top Slowest Operations:")
		for i, op := range suite.Summary.TopSlowestOps {
			fmt.Printf("  %d. %s: %.2fms\n", i+1, op.Name, op.NsPerOp/1000000)
		}
	}

	if len(suite.Summary.HighestAllocators) > 0 {
		fmt.Println("\n🧠 Highest Memory Allocators:")
		for i, op := range suite.Summary.HighestAllocators {
			fmt.Printf("  %d. %s: %d allocs/op, %.2fKB/op\n",
				i+1, op.Name, op.AllocsPerOp, float64(op.BytesPerOp)/1024)
		}
	}
}

func compareWithBaseline(current, baseline BenchmarkSuite) {
	// Create a map of baseline results for quick lookup
	baselineMap := make(map[string]BenchmarkResult)
	for _, result := range baseline.Results {
		key := result.Package + "::" + result.Name
		baselineMap[key] = result
	}

	improvements := 0
	regressions := 0

	fmt.Println("\nSignificant Changes (>10% difference):")

	for _, current := range current.Results {
		key := current.Package + "::" + current.Name
		if baseline, exists := baselineMap[key]; exists {
			// Compare performance
			perfChange := (current.NsPerOp - baseline.NsPerOp) / baseline.NsPerOp * 100

			if perfChange > 10 {
				fmt.Printf("  📈 REGRESSION: %s: %.1f%% slower (%.2fms -> %.2fms)\n",
					current.Name, perfChange, baseline.NsPerOp/1000000, current.NsPerOp/1000000)
				regressions++
			} else if perfChange < -10 {
				fmt.Printf("  📉 IMPROVEMENT: %s: %.1f%% faster (%.2fms -> %.2fms)\n",
					current.Name, -perfChange, baseline.NsPerOp/1000000, current.NsPerOp/1000000)
				improvements++
			}

			// Compare allocations if available
			if current.AllocsPerOp > 0 && baseline.AllocsPerOp > 0 {
				allocChange := float64(current.AllocsPerOp-baseline.AllocsPerOp) / float64(baseline.AllocsPerOp) * 100
				if allocChange > 20 {
					fmt.Printf("  🧠 ALLOC REGRESSION: %s: %.1f%% more allocations (%d -> %d)\n",
						current.Name, allocChange, baseline.AllocsPerOp, current.AllocsPerOp)
				} else if allocChange < -20 {
					fmt.Printf("  🧠 ALLOC IMPROVEMENT: %s: %.1f%% fewer allocations (%d -> %d)\n",
						current.Name, -allocChange, baseline.AllocsPerOp, current.AllocsPerOp)
				}
			}
		}
	}

	fmt.Printf("\nSummary: %d improvements, %d regressions\n", improvements, regressions)
}

func getGoVersion() string {
	cmd := exec.Command("go", "version")
	output, err := cmd.Output()
	if err != nil {
		return "unknown"
	}
	return strings.TrimSpace(string(output))
}

func getGitCommit() string {
	cmd := exec.Command("git", "rev-parse", "--short", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return "unknown"
	}
	return strings.TrimSpace(string(output))
}
