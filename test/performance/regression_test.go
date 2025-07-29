package performance

import (
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/containifyci/engine-ci/pkg/builder"
	"github.com/containifyci/engine-ci/pkg/config"
	"github.com/containifyci/engine-ci/pkg/container"
	"github.com/containifyci/engine-ci/pkg/cri/types"
	"github.com/containifyci/engine-ci/pkg/golang"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Performance baselines for regression testing
var (
	// Configuration loading thresholds
	DefaultConfigLoadingThreshold    = 10 * time.Millisecond
	EnvironmentConfigLoadingThreshold = 50 * time.Millisecond
	FileConfigLoadingThreshold       = 20 * time.Millisecond
	
	// Builder creation thresholds
	BuilderCreationThreshold = 5 * time.Millisecond
	FactoryCreationThreshold = 1 * time.Millisecond
	
	// Method call thresholds
	MethodCallThreshold = 1 * time.Microsecond
	
	// Memory thresholds (bytes)
	ConfigMemoryThreshold  = 100 * 1024  // 100KB
	BuilderMemoryThreshold = 50 * 1024   // 50KB
)

// TestPerformanceRegression tests that performance has not degraded
func TestPerformanceRegression(t *testing.T) {
	t.Run("ConfigurationPerformance", func(t *testing.T) {
		testConfigurationPerformance(t)
	})

	t.Run("BuilderPerformance", func(t *testing.T) {
		testBuilderPerformance(t)
	})

	t.Run("MemoryUsage", func(t *testing.T) {
		testMemoryUsage(t)
	})

	t.Run("ConcurrencyPerformance", func(t *testing.T) {
		testConcurrencyPerformance(t)
	})

	t.Run("ScalabilityPerformance", func(t *testing.T) {
		testScalabilityPerformance(t)
	})
}

func testConfigurationPerformance(t *testing.T) {
	t.Run("DefaultConfigLoading", func(t *testing.T) {
		// Benchmark default configuration loading
		iterations := 1000
		
		start := time.Now()
		for i := 0; i < iterations; i++ {
			cfg := config.GetDefaultConfig()
			_ = cfg
		}
		duration := time.Since(start)
		avgDuration := duration / time.Duration(iterations)
		
		assert.Less(t, avgDuration, DefaultConfigLoadingThreshold,
			"Default config loading should be fast: %v (threshold: %v)", 
			avgDuration, DefaultConfigLoadingThreshold)
		
		t.Logf("Default config loading: %v/op (threshold: %v)", avgDuration, DefaultConfigLoadingThreshold)
	})

	t.Run("ConfigurationValidation", func(t *testing.T) {
		cfg := config.GetDefaultConfig()
		iterations := 1000
		
		start := time.Now()
		for i := 0; i < iterations; i++ {
			err := config.ValidateConfig(cfg)
			require.NoError(t, err)
		}
		duration := time.Since(start)
		avgDuration := duration / time.Duration(iterations)
		
		// Validation should be fast
		validationThreshold := 5 * time.Millisecond
		assert.Less(t, avgDuration, validationThreshold,
			"Config validation should be fast: %v (threshold: %v)", 
			avgDuration, validationThreshold)
		
		t.Logf("Config validation: %v/op (threshold: %v)", avgDuration, validationThreshold)
	})

	t.Run("ConfigurationMerging", func(t *testing.T) {
		partialConfig := &config.Config{
			Version: "test",
			Language: config.LanguageConfig{
				Go: config.GoConfig{
					Version: "1.25.0",
				},
			},
		}
		
		iterations := 1000
		start := time.Now()
		for i := 0; i < iterations; i++ {
			merged := config.MergeWithDefaults(partialConfig)
			_ = merged
		}
		duration := time.Since(start)
		avgDuration := duration / time.Duration(iterations)
		
		mergeThreshold := 10 * time.Millisecond
		assert.Less(t, avgDuration, mergeThreshold,
			"Config merging should be fast: %v (threshold: %v)", 
			avgDuration, mergeThreshold)
		
		t.Logf("Config merging: %v/op (threshold: %v)", avgDuration, mergeThreshold)
	})

	t.Run("GlobalConfigAccess", func(t *testing.T) {
		iterations := 10000
		
		start := time.Now()
		for i := 0; i < iterations; i++ {
			cfg := config.GetGlobalConfig()
			_ = cfg
		}
		duration := time.Since(start)
		avgDuration := duration / time.Duration(iterations)
		
		globalAccessThreshold := 100 * time.Nanosecond
		assert.Less(t, avgDuration, globalAccessThreshold,
			"Global config access should be very fast: %v (threshold: %v)", 
			avgDuration, globalAccessThreshold)
		
		t.Logf("Global config access: %v/op (threshold: %v)", avgDuration, globalAccessThreshold)
	})
}

func testBuilderPerformance(t *testing.T) {
	build := createPerformanceTestBuild()

	t.Run("GoBuilderCreation", func(t *testing.T) {
		iterations := 100
		
		start := time.Now()
		for i := 0; i < iterations; i++ {
			builder, err := golang.NewGoBuilder(build, golang.VariantAlpine)
			require.NoError(t, err)
			_ = builder
		}
		duration := time.Since(start)
		avgDuration := duration / time.Duration(iterations)
		
		assert.Less(t, avgDuration, BuilderCreationThreshold,
			"Builder creation should be fast: %v (threshold: %v)", 
			avgDuration, BuilderCreationThreshold)
		
		t.Logf("Go builder creation: %v/op (threshold: %v)", avgDuration, BuilderCreationThreshold)
	})

	t.Run("LegacyBuilderCreation", func(t *testing.T) {
		iterations := 100
		
		// Test legacy functions
		legacyFunctions := []func(container.Build) (*golang.GoBuilder, error){
			golang.New,
			golang.NewDebian,
			golang.NewCGO,
		}
		
		for _, fn := range legacyFunctions {
			start := time.Now()
			for i := 0; i < iterations; i++ {
				builder, err := fn(build)
				require.NoError(t, err)
				_ = builder
			}
			duration := time.Since(start)
			avgDuration := duration / time.Duration(iterations)
			
			// Legacy functions should maintain reasonable performance
			legacyThreshold := BuilderCreationThreshold * 2
			assert.Less(t, avgDuration, legacyThreshold,
				"Legacy builder creation should be reasonably fast: %v (threshold: %v)", 
				avgDuration, legacyThreshold)
		}
	})

	t.Run("BuilderMethodCalls", func(t *testing.T) {
		builder, err := golang.NewGoBuilder(build, golang.VariantAlpine)
		require.NoError(t, err)
		
		iterations := 10000
		
		// Test individual method performance
		methods := map[string]func(){
			"Name":             func() { _ = builder.Name() },
			"IsAsync":          func() { _ = builder.IsAsync() },
			"LintImage":        func() { _ = builder.LintImage() },
			"CacheFolder":      func() { _ = builder.CacheFolder() },
			"BuildScript":      func() { _ = builder.BuildScript() },
		}
		
		for methodName, method := range methods {
			start := time.Now()
			for i := 0; i < iterations; i++ {
				method()
			}
			duration := time.Since(start)
			avgDuration := duration / time.Duration(iterations)
			
			assert.Less(t, avgDuration, MethodCallThreshold,
				"Method %s should be fast: %v (threshold: %v)", 
				methodName, avgDuration, MethodCallThreshold)
			
			t.Logf("Method %s: %v/op (threshold: %v)", methodName, avgDuration, MethodCallThreshold)
		}
	})

	t.Run("FactoryOperations", func(t *testing.T) {
		// Test factory creation performance
		iterations := 1000
		
		start := time.Now()
		for i := 0; i < iterations; i++ {
			factory := builder.NewStandardBuildFactory()
			_ = factory
		}
		duration := time.Since(start)
		avgDuration := duration / time.Duration(iterations)
		
		assert.Less(t, avgDuration, FactoryCreationThreshold,
			"Factory creation should be very fast: %v (threshold: %v)", 
			avgDuration, FactoryCreationThreshold)
		
		// Test factory operations
		factory := builder.NewStandardBuildFactory()
		
		start = time.Now()
		for i := 0; i < iterations; i++ {
			types := factory.SupportedTypes()
			_ = types
		}
		duration = time.Since(start)
		avgDuration = duration / time.Duration(iterations)
		
		factoryOpThreshold := 1 * time.Microsecond
		assert.Less(t, avgDuration, factoryOpThreshold,
			"Factory operations should be very fast: %v (threshold: %v)", 
			avgDuration, factoryOpThreshold)
		
		t.Logf("Factory creation: %v/op, SupportedTypes: %v/op", 
			duration/time.Duration(iterations), avgDuration)
	})

	t.Run("IntermediateImageCaching", func(t *testing.T) {
		builder, err := golang.NewGoBuilder(build, golang.VariantAlpine)
		require.NoError(t, err)
		
		// First call (may compute checksum)
		start := time.Now()
		img1 := builder.IntermediateImage()
		firstCallDuration := time.Since(start)
		
		// Subsequent calls (should be cached)
		iterations := 1000
		start = time.Now()
		for i := 0; i < iterations; i++ {
			img := builder.IntermediateImage()
			assert.Equal(t, img1, img)
		}
		duration := time.Since(start)
		avgCachedDuration := duration / time.Duration(iterations)
		
		// Cached calls should be much faster than first call
		assert.Less(t, avgCachedDuration, firstCallDuration/10,
			"Cached intermediate image calls should be much faster: %v vs %v", 
			avgCachedDuration, firstCallDuration)
		
		// And should be very fast in absolute terms
		cachedThreshold := 100 * time.Nanosecond
		assert.Less(t, avgCachedDuration, cachedThreshold,
			"Cached intermediate image should be very fast: %v (threshold: %v)", 
			avgCachedDuration, cachedThreshold)
		
		t.Logf("Intermediate image - First call: %v, Cached: %v/op (threshold: %v)", 
			firstCallDuration, avgCachedDuration, cachedThreshold)
	})
}

func testMemoryUsage(t *testing.T) {
	t.Run("ConfigurationMemoryUsage", func(t *testing.T) {
		runtime.GC()
		var m1, m2, m3 runtime.MemStats
		runtime.ReadMemStats(&m1)
		
		// Create many configurations
		configs := make([]*config.Config, 100)
		for i := 0; i < 100; i++ {
			configs[i] = config.GetDefaultConfig()
		}
		
		runtime.ReadMemStats(&m2)
		configMemory := m2.Alloc - m1.Alloc
		
		// Keep references to prevent GC
		_ = configs
		
		assert.Less(t, configMemory, uint64(ConfigMemoryThreshold),
			"Configuration memory usage should be reasonable: %d bytes (threshold: %d)", 
			configMemory, ConfigMemoryThreshold)
		
		t.Logf("100 configurations use %d bytes (avg: %d bytes/config, threshold: %d)", 
			configMemory, configMemory/100, ConfigMemoryThreshold/100)
		
		// Test for memory leaks
		configs = nil
		runtime.GC()
		runtime.ReadMemStats(&m3)
		
		// Memory should be reclaimed (allowing some variance)
		assert.Less(t, m3.Alloc, m2.Alloc+uint64(ConfigMemoryThreshold/10),
			"Memory should be reclaimed after GC")
	})

	t.Run("BuilderMemoryUsage", func(t *testing.T) {
		runtime.GC()
		var m1, m2, m3 runtime.MemStats
		runtime.ReadMemStats(&m1)
		
		build := createPerformanceTestBuild()
		builders := make([]*golang.GoBuilder, 100)
		
		for i := 0; i < 100; i++ {
			builder, err := golang.NewGoBuilder(build, golang.VariantAlpine)
			require.NoError(t, err)
			builders[i] = builder
		}
		
		runtime.ReadMemStats(&m2)
		builderMemory := m2.Alloc - m1.Alloc
		
		// Keep references to prevent GC
		_ = builders
		
		assert.Less(t, builderMemory, uint64(BuilderMemoryThreshold*100),
			"Builder memory usage should be reasonable: %d bytes (threshold: %d)", 
			builderMemory, BuilderMemoryThreshold*100)
		
		t.Logf("100 builders use %d bytes (avg: %d bytes/builder, threshold: %d)", 
			builderMemory, builderMemory/100, BuilderMemoryThreshold)
		
		// Test for memory leaks
		builders = nil
		runtime.GC()
		runtime.ReadMemStats(&m3)
		
		assert.Less(t, m3.Alloc, m2.Alloc+uint64(BuilderMemoryThreshold*10),
			"Memory should be reclaimed after GC")
	})

	t.Run("MemoryLeakDetection", func(t *testing.T) {
		runtime.GC()
		var m1, m2 runtime.MemStats
		runtime.ReadMemStats(&m1)
		
		// Perform many operations that should not leak memory
		build := createPerformanceTestBuild()
		
		for i := 0; i < 1000; i++ {
			// Create and discard builders
			builder, err := golang.NewGoBuilder(build, golang.VariantAlpine)
			require.NoError(t, err)
			
			// Perform operations
			_ = builder.Name()
			_ = builder.Images()
			_ = builder.IntermediateImage()
			_ = builder.BuildScript()
			_ = builder.CacheFolder()
			
			// Create and discard configurations
			cfg := config.GetDefaultConfig()
			err = config.ValidateConfig(cfg)
			require.NoError(t, err)
			
			// Force GC periodically
			if i%100 == 0 {
				runtime.GC()
			}
		}
		
		runtime.GC()
		runtime.ReadMemStats(&m2)
		
		memoryIncrease := m2.Alloc - m1.Alloc
		
		// Memory increase should be minimal
		leakThreshold := uint64(1024 * 1024) // 1MB
		assert.Less(t, memoryIncrease, leakThreshold,
			"Memory leak detected: %d bytes increase (threshold: %d)", 
			memoryIncrease, leakThreshold)
		
		t.Logf("Memory increase after 1000 operations: %d bytes (threshold: %d)", 
			memoryIncrease, leakThreshold)
	})
}

func testConcurrencyPerformance(t *testing.T) {
	t.Run("ConcurrentConfigurationLoading", func(t *testing.T) {
		concurrency := 10
		iterations := 100
		done := make(chan time.Duration, concurrency)
		
		start := time.Now()
		for i := 0; i < concurrency; i++ {
			go func() {
				localStart := time.Now()
				for j := 0; j < iterations; j++ {
					cfg := config.GetDefaultConfig()
					err := config.ValidateConfig(cfg)
					assert.NoError(t, err)
				}
				done <- time.Since(localStart)
			}()
		}
		
		totalDuration := time.Since(start)
		
		// Collect individual goroutine times
		var maxGoroutineDuration time.Duration
		for i := 0; i < concurrency; i++ {
			duration := <-done
			if duration > maxGoroutineDuration {
				maxGoroutineDuration = duration
			}
		}
		
		// Concurrent execution should be efficient
		sequentialEstimate := time.Duration(concurrency*iterations) * DefaultConfigLoadingThreshold
		concurrencyEfficiency := float64(sequentialEstimate) / float64(totalDuration)
		
		assert.Greater(t, concurrencyEfficiency, 2.0,
			"Concurrent configuration loading should be efficient: %.2fx speedup", 
			concurrencyEfficiency)
		
		t.Logf("Concurrent config loading: %d goroutines x %d ops in %v (efficiency: %.2fx)", 
			concurrency, iterations, totalDuration, concurrencyEfficiency)
	})

	t.Run("ConcurrentBuilderCreation", func(t *testing.T) {
		build := createPerformanceTestBuild()
		concurrency := 10
		iterations := 50
		
		done := make(chan time.Duration, concurrency)
		
		start := time.Now()
		for i := 0; i < concurrency; i++ {
			go func() {
				localStart := time.Now()
				for j := 0; j < iterations; j++ {
					builder, err := golang.NewGoBuilder(build, golang.VariantAlpine)
					assert.NoError(t, err)
					
					// Perform some operations
					_ = builder.Name()
					_ = builder.Images()
				}
				done <- time.Since(localStart)
			}()
		}
		
		totalDuration := time.Since(start)
		
		// Collect results
		for i := 0; i < concurrency; i++ {
			<-done
		}
		
		// Test should complete in reasonable time
		concurrentThreshold := time.Duration(concurrency*iterations) * BuilderCreationThreshold / 2
		assert.Less(t, totalDuration, concurrentThreshold,
			"Concurrent builder creation should be efficient: %v (threshold: %v)", 
			totalDuration, concurrentThreshold)
		
		t.Logf("Concurrent builder creation: %d goroutines x %d ops in %v (threshold: %v)", 
			concurrency, iterations, totalDuration, concurrentThreshold)
	})

	t.Run("RaceConditionFreedom", func(t *testing.T) {
		// Test for race conditions in concurrent access
		build := createPerformanceTestBuild()
		concurrency := 20
		iterations := 100
		
		var wg sync.WaitGroup
		
		for i := 0; i < concurrency; i++ {
			wg.Add(1)
			go func(routineID int) {
				defer wg.Done()
				
				for j := 0; j < iterations; j++ {
					// Mix different operations
					switch j % 4 {
					case 0:
						cfg := config.GetDefaultConfig()
						config.SetGlobalConfig(cfg)
					case 1:
						builder, err := golang.New(build)
						assert.NoError(t, err)
						_ = builder.Name()
					case 2:
						factory := builder.NewStandardBuildFactory()
						_ = factory.SupportedTypes()
					case 3:
						cfg := config.GetGlobalConfig()
						_ = cfg.Version
					}
				}
			}(i)
		}
		
		// Should complete without race conditions
		done := make(chan bool, 1)
		go func() {
			wg.Wait()
			done <- true
		}()
		
		select {
		case <-done:
			t.Log("Race condition test completed successfully")
		case <-time.After(10 * time.Second):
			t.Fatal("Race condition test timed out - possible deadlock")
		}
	})
}

func testScalabilityPerformance(t *testing.T) {
	t.Run("ScalingConfigurationSize", func(t *testing.T) {
		// Test performance as configuration complexity increases
		baseCfg := config.GetDefaultConfig()
		
		// Create configurations with increasing complexity
		scales := []int{1, 10, 100}
		
		for _, scale := range scales {
			cfg := baseCfg
			
			// Add complexity (multiple environment variables)
			for i := 0; i < scale; i++ {
				if cfg.Language.Go.Environment == nil {
					cfg.Language.Go.Environment = make(map[string]string)
				}
				cfg.Language.Go.Environment["VAR_"+string(rune(i))] = "value_" + string(rune(i))
				
				if cfg.Language.Maven.Environment == nil {
					cfg.Language.Maven.Environment = make(map[string]string)
				}
				cfg.Language.Maven.Environment["MAVEN_VAR_"+string(rune(i))] = "maven_value_" + string(rune(i))
			}
			
			// Test validation performance
			iterations := 100
			start := time.Now()
			for i := 0; i < iterations; i++ {
				err := config.ValidateConfig(cfg)
				require.NoError(t, err)
			}
			duration := time.Since(start)
			avgDuration := duration / time.Duration(iterations)
			
			// Performance should scale reasonably
			scaledThreshold := DefaultConfigLoadingThreshold * time.Duration(1+scale/10)
			assert.Less(t, avgDuration, scaledThreshold,
				"Config validation should scale reasonably for scale %d: %v (threshold: %v)", 
				scale, avgDuration, scaledThreshold)
			
			t.Logf("Config validation scale %d: %v/op (threshold: %v)", 
				scale, avgDuration, scaledThreshold)
		}
	})

	t.Run("ScalingBuilderOperations", func(t *testing.T) {
		// Test performance with multiple builders
		build := createPerformanceTestBuild()
		builderCounts := []int{1, 10, 50, 100}
		
		for _, count := range builderCounts {
			start := time.Now()
			
			builders := make([]*golang.GoBuilder, count)
			for i := 0; i < count; i++ {
				builder, err := golang.NewGoBuilder(build, golang.VariantAlpine)
				require.NoError(t, err)
				builders[i] = builder
			}
			
			// Perform operations on all builders
			for _, builder := range builders {
				_ = builder.Name()
				_ = builder.IntermediateImage()
			}
			
			duration := time.Since(start)
			avgDuration := duration / time.Duration(count)
			
			// Should scale roughly linearly
			scaledThreshold := BuilderCreationThreshold * 2
			assert.Less(t, avgDuration, scaledThreshold,
				"Builder operations should scale linearly for %d builders: %v/builder (threshold: %v)", 
				count, avgDuration, scaledThreshold)
			
			t.Logf("Builder operations scale %d: %v total, %v/builder (threshold: %v)", 
				count, duration, avgDuration, scaledThreshold)
		}
	})

	t.Run("MemoryScaling", func(t *testing.T) {
		// Test memory usage scaling
		build := createPerformanceTestBuild()
		counts := []int{10, 50, 100}
		
		for _, count := range counts {
			runtime.GC()
			var m1, m2 runtime.MemStats
			runtime.ReadMemStats(&m1)
			
			builders := make([]*golang.GoBuilder, count)
			for i := 0; i < count; i++ {
				builder, err := golang.NewGoBuilder(build, golang.VariantAlpine)
				require.NoError(t, err)
				builders[i] = builder
			}
			
			runtime.ReadMemStats(&m2)
			memoryUsed := m2.Alloc - m1.Alloc
			memoryPerBuilder := memoryUsed / uint64(count)
			
			// Memory per builder should remain reasonable
			assert.Less(t, memoryPerBuilder, uint64(BuilderMemoryThreshold),
				"Memory per builder should be reasonable for %d builders: %d bytes/builder (threshold: %d)", 
				count, memoryPerBuilder, BuilderMemoryThreshold)
			
			t.Logf("Memory scaling %d builders: %d total bytes, %d bytes/builder (threshold: %d)", 
				count, memoryUsed, memoryPerBuilder, BuilderMemoryThreshold)
			
			// Keep reference to prevent GC during test
			_ = builders
		}
	})
}

// BenchmarkConfigurationOperations provides detailed benchmarks for configuration operations
func BenchmarkConfigurationOperations(b *testing.B) {
	b.Run("GetDefaultConfig", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			cfg := config.GetDefaultConfig()
			_ = cfg
		}
	})

	b.Run("ValidateConfig", func(b *testing.B) {
		cfg := config.GetDefaultConfig()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			err := config.ValidateConfig(cfg)
			_ = err
		}
	})

	b.Run("MergeWithDefaults", func(b *testing.B) {
		partialConfig := &config.Config{
			Version: "benchmark",
			Language: config.LanguageConfig{
				Go: config.GoConfig{Version: "1.25.0"},
			},
		}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			merged := config.MergeWithDefaults(partialConfig)
			_ = merged
		}
	})

	b.Run("GetGlobalConfig", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			cfg := config.GetGlobalConfig()
			_ = cfg
		}
	})
}

// BenchmarkBuilderOperations provides detailed benchmarks for builder operations
func BenchmarkBuilderOperations(b *testing.B) {
	build := createPerformanceTestBuild()

	b.Run("NewGoBuilder", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			builder, err := golang.NewGoBuilder(build, golang.VariantAlpine)
			if err != nil {
				b.Fatal(err)
			}
			_ = builder
		}
	})

	b.Run("LegacyNew", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			builder, err := golang.New(build)
			if err != nil {
				b.Fatal(err)
			}
			_ = builder
		}
	})

	builder, err := golang.NewGoBuilder(build, golang.VariantAlpine)
	if err != nil {
		b.Fatal(err)
	}

	b.Run("BuilderName", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			name := builder.Name()
			_ = name
		}
	})

	b.Run("BuilderImages", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			images := builder.Images()
			_ = images
		}
	})

	b.Run("IntermediateImage", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			img := builder.IntermediateImage()
			_ = img
		}
	})

	b.Run("BuildScript", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			script := builder.BuildScript()
			_ = script
		}
	})
}

// BenchmarkConcurrentOperations provides benchmarks for concurrent operations
func BenchmarkConcurrentOperations(b *testing.B) {
	build := createPerformanceTestBuild()

	b.Run("ConcurrentBuilderCreation", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				builder, err := golang.NewGoBuilder(build, golang.VariantAlpine)
				if err != nil {
					b.Fatal(err)
				}
				_ = builder
			}
		})
	})

	b.Run("ConcurrentConfigLoading", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				cfg := config.GetDefaultConfig()
				err := config.ValidateConfig(cfg)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	})
}

// createPerformanceTestBuild creates a test build for performance testing
func createPerformanceTestBuild() container.Build {
	return container.Build{
		BuildType: container.GoLang,
		App:       "perf-test-app",
		File:      "main.go",
		Folder:    "./",
		Platform: types.Platform{
			Host: &types.PlatformSpec{
				OS:           "darwin",
				Architecture: "arm64",
			},
			Container: &types.PlatformSpec{
				OS:           "linux",
				Architecture: "amd64",
			},
		},
		Env:     container.LocalEnv,
		Verbose: false,
		Custom:  make(container.Custom),
	}
}