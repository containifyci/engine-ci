package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// RegressionTest defines performance regression thresholds
type RegressionTest struct {
	Name                  string  `json:"name"`
	MaxPerformanceDrop    float64 `json:"max_performance_drop"`    // Maximum allowed performance regression (%)
	MaxAllocationIncrease int64   `json:"max_allocation_increase"` // Maximum allowed allocation increase
	MaxMemoryIncrease     int64   `json:"max_memory_increase"`     // Maximum allowed memory increase (bytes)
	Critical              bool    `json:"critical"`                // Is this a critical performance test?
}

// RegressionTestSuite contains all regression tests
type RegressionTestSuite struct {
	Tests []RegressionTest `json:"tests"`
}

// RegressionResult represents the result of a regression test
type RegressionResult struct {
	Test     RegressionTest     `json:"test"`
	Passed   bool               `json:"passed"`
	Current  BenchmarkResult    `json:"current"`
	Baseline BenchmarkResult    `json:"baseline"`
	Changes  PerformanceChanges `json:"changes"`
	Message  string             `json:"message"`
}

// PerformanceChanges tracks specific performance metrics changes
type PerformanceChanges struct {
	PerformanceChange float64 `json:"performance_change"` // % change in ns/op
	AllocationChange  int64   `json:"allocation_change"`  // absolute change in allocs/op
	MemoryChange      int64   `json:"memory_change"`      // absolute change in bytes/op
}

func runRegressionTests() {
	fmt.Println("\n🔍 Running Performance Regression Tests")
	fmt.Println("=======================================")

	// Load baseline
	baseline, err := loadResults("benchmarks/results/baseline.json")
	if err != nil {
		fmt.Printf("❌ Cannot load baseline for regression testing: %v\n", err)
		return
	}

	// Load latest results
	latestFile := findLatestResultsFile()
	if latestFile == "" {
		fmt.Println("❌ No recent benchmark results found")
		return
	}

	current, err := loadResults(latestFile)
	if err != nil {
		fmt.Printf("❌ Cannot load current results: %v\n", err)
		return
	}

	// Load regression test configuration
	suite := loadRegressionTestSuite()

	// Run regression tests
	results := performRegressionTests(suite, baseline, current)

	// Report results
	reportRegressionResults(results)

	// Save results
	saveRegressionResults(results)
}

func loadRegressionTestSuite() RegressionTestSuite {
	// Define critical performance regression tests
	return RegressionTestSuite{
		Tests: []RegressionTest{
			// Container operations - critical path
			{
				Name:                  "ContainerOperations/New_Container_Creation",
				MaxPerformanceDrop:    15.0, // 15% max regression
				MaxAllocationIncrease: 2,
				MaxMemoryIncrease:     1024, // 1KB
				Critical:              true,
			},
			{
				Name:                  "ContainerOperations/Parse_Image_Tag",
				MaxPerformanceDrop:    20.0,
				MaxAllocationIncrease: 1,
				MaxMemoryIncrease:     512,
				Critical:              true,
			},
			{
				Name:                  "ContainerOperations/Checksum_Computation",
				MaxPerformanceDrop:    25.0,
				MaxAllocationIncrease: 0, // No allocation increase allowed
				MaxMemoryIncrease:     0,
				Critical:              true,
			},

			// Tar operations - I/O intensive
			{
				Name:                  "TarOperations/Small_Files",
				MaxPerformanceDrop:    30.0,
				MaxAllocationIncrease: 10,
				MaxMemoryIncrease:     10240, // 10KB
				Critical:              false,
			},
			{
				Name:                  "TarOperations/Large_Files",
				MaxPerformanceDrop:    40.0,
				MaxAllocationIncrease: 50,
				MaxMemoryIncrease:     102400, // 100KB
				Critical:              true,
			},

			// String operations - frequently called
			{
				Name:                  "StringOperations/Safe_Short_Operation",
				MaxPerformanceDrop:    10.0,
				MaxAllocationIncrease: 0,
				MaxMemoryIncrease:     0,
				Critical:              true,
			},
			{
				Name:                  "StringOperations/Build_AsFlags_Operation",
				MaxPerformanceDrop:    20.0,
				MaxAllocationIncrease: 5,
				MaxMemoryIncrease:     2048, // 2KB
				Critical:              false,
			},

			// Concurrent operations
			{
				Name:                  "ConcurrentOperations/Concurrent_Image_Tag_Parsing",
				MaxPerformanceDrop:    15.0,
				MaxAllocationIncrease: 1,
				MaxMemoryIncrease:     512,
				Critical:              true,
			},
			{
				Name:                  "ConcurrentOperations/Concurrent_Checksum_Computation",
				MaxPerformanceDrop:    20.0,
				MaxAllocationIncrease: 0,
				MaxMemoryIncrease:     0,
				Critical:              true,
			},

			// Logger operations - high frequency
			{
				Name:                  "LogAggregation/Single_Routine_Logging",
				MaxPerformanceDrop:    15.0,
				MaxAllocationIncrease: 2,
				MaxMemoryIncrease:     1024,
				Critical:              true,
			},
			{
				Name:                  "LogAggregation/Multiple_Routine_Logging",
				MaxPerformanceDrop:    25.0,
				MaxAllocationIncrease: 5,
				MaxMemoryIncrease:     2048,
				Critical:              true,
			},

			// Build operations
			{
				Name:                  "BuildOperations/Build_Defaults_Setup",
				MaxPerformanceDrop:    20.0,
				MaxAllocationIncrease: 3,
				MaxMemoryIncrease:     1536,
				Critical:              false,
			},
			{
				Name:                  "StringOperations/AsFlags_String_Building",
				MaxPerformanceDrop:    25.0,
				MaxAllocationIncrease: 10,
				MaxMemoryIncrease:     4096, // 4KB
				Critical:              false,
			},

			// Container runtime manager
			{
				Name:                  "ContainerManager/Runtime_Detection",
				MaxPerformanceDrop:    10.0,
				MaxAllocationIncrease: 1,
				MaxMemoryIncrease:     256,
				Critical:              false,
			},
			{
				Name:                  "ContainerManager/Singleton_Access",
				MaxPerformanceDrop:    5.0,
				MaxAllocationIncrease: 0,
				MaxMemoryIncrease:     0,
				Critical:              true,
			},
		},
	}
}

func performRegressionTests(suite RegressionTestSuite, baseline, current BenchmarkSuite) []RegressionResult {
	var results []RegressionResult

	// Create lookup maps
	baselineMap := make(map[string]BenchmarkResult)
	currentMap := make(map[string]BenchmarkResult)

	for _, result := range baseline.Results {
		baselineMap[result.Name] = result
	}

	for _, result := range current.Results {
		currentMap[result.Name] = result
	}

	// Run each regression test
	for _, test := range suite.Tests {
		result := RegressionResult{
			Test:   test,
			Passed: false,
		}

		baselineResult, hasBaseline := baselineMap[test.Name]
		currentResult, hasCurrent := currentMap[test.Name]

		if !hasBaseline || !hasCurrent {
			result.Message = "Benchmark not found in baseline or current results"
			results = append(results, result)
			continue
		}

		result.Baseline = baselineResult
		result.Current = currentResult

		// Calculate changes
		result.Changes = PerformanceChanges{
			PerformanceChange: (currentResult.NsPerOp - baselineResult.NsPerOp) / baselineResult.NsPerOp * 100,
			AllocationChange:  currentResult.AllocsPerOp - baselineResult.AllocsPerOp,
			MemoryChange:      currentResult.BytesPerOp - baselineResult.BytesPerOp,
		}

		// Check for regressions
		passed := true
		var issues []string

		// Performance regression check
		if result.Changes.PerformanceChange > test.MaxPerformanceDrop {
			passed = false
			issues = append(issues, fmt.Sprintf("Performance regression: %.1f%% (max: %.1f%%)",
				result.Changes.PerformanceChange, test.MaxPerformanceDrop))
		}

		// Allocation increase check
		if result.Changes.AllocationChange > test.MaxAllocationIncrease {
			passed = false
			issues = append(issues, fmt.Sprintf("Allocation increase: %d (max: %d)",
				result.Changes.AllocationChange, test.MaxAllocationIncrease))
		}

		// Memory increase check
		if result.Changes.MemoryChange > test.MaxMemoryIncrease {
			passed = false
			issues = append(issues, fmt.Sprintf("Memory increase: %d bytes (max: %d)",
				result.Changes.MemoryChange, test.MaxMemoryIncrease))
		}

		result.Passed = passed
		if !passed {
			result.Message = fmt.Sprintf("Regression detected: %v", issues)
		} else {
			result.Message = "No regression detected"
		}

		results = append(results, result)
	}

	return results
}

func reportRegressionResults(results []RegressionResult) {
	totalTests := len(results)
	passedTests := 0
	criticalFailures := 0
	nonCriticalFailures := 0

	fmt.Printf("\nRegression Test Results (%d tests)\n", totalTests)
	fmt.Println("-----------------------------------")

	for _, result := range results {
		status := "✅ PASS"
		if !result.Passed {
			if result.Test.Critical {
				status = "❌ CRITICAL FAIL"
				criticalFailures++
			} else {
				status = "⚠️  FAIL"
				nonCriticalFailures++
			}
		} else {
			passedTests++
		}

		fmt.Printf("%s %s\n", status, result.Test.Name)

		if !result.Passed {
			fmt.Printf("   %s\n", result.Message)
			fmt.Printf("   Performance: %.1f%% change (%.2f -> %.2f ms)\n",
				result.Changes.PerformanceChange,
				result.Baseline.NsPerOp/1000000,
				result.Current.NsPerOp/1000000)

			if result.Changes.AllocationChange != 0 {
				fmt.Printf("   Allocations: %+d change (%d -> %d)\n",
					result.Changes.AllocationChange,
					result.Baseline.AllocsPerOp,
					result.Current.AllocsPerOp)
			}

			if result.Changes.MemoryChange != 0 {
				fmt.Printf("   Memory: %+d bytes change (%d -> %d)\n",
					result.Changes.MemoryChange,
					result.Baseline.BytesPerOp,
					result.Current.BytesPerOp)
			}
			fmt.Println()
		}
	}

	fmt.Printf("\nSummary: %d/%d tests passed\n", passedTests, totalTests)
	if criticalFailures > 0 {
		fmt.Printf("❌ %d critical failures detected\n", criticalFailures)
	}
	if nonCriticalFailures > 0 {
		fmt.Printf("⚠️  %d non-critical failures detected\n", nonCriticalFailures)
	}

	// Exit with error code if critical failures
	if criticalFailures > 0 {
		fmt.Println("\n🚨 Critical performance regressions detected!")
		fmt.Println("Please review and optimize the affected code paths.")
		os.Exit(1)
	}
}

func saveRegressionResults(results []RegressionResult) {
	timestamp := time.Now().Format("2006-01-02_15-04-05")
	filename := filepath.Join("benchmarks/results", fmt.Sprintf("regression_%s.json", timestamp))

	file, err := os.Create(filename)
	if err != nil {
		fmt.Printf("⚠️  Failed to save regression results: %v\n", err)
		return
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(results); err != nil {
		fmt.Printf("⚠️  Failed to encode regression results: %v\n", err)
		return
	}

	fmt.Printf("💾 Regression test results saved to: %s\n", filename)
}

func findLatestResultsFile() string {
	resultsDir := "benchmarks/results"

	files, err := filepath.Glob(filepath.Join(resultsDir, "benchmark_*.json"))
	if err != nil || len(files) == 0 {
		return ""
	}

	// Find the most recent file
	var latestFile string
	var latestTime time.Time

	for _, file := range files {
		info, err := os.Stat(file)
		if err != nil {
			continue
		}

		if info.ModTime().After(latestTime) {
			latestTime = info.ModTime()
			latestFile = file
		}
	}

	return latestFile
}

// Entry point for regression testing
func main() {
	runRegressionTests()
}
