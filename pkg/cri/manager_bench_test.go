package cri

import (
	"context"
	"os"
	"os/exec"
	"sync"
	"testing"

	"github.com/containifyci/engine-ci/pkg/cri/utils"
)

// BenchmarkContainerManager benchmarks container manager operations
func BenchmarkContainerManager(b *testing.B) {
	b.Run("Runtime Detection", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			runtime := DetectContainerRuntime()
			_ = runtime
		}
	})

	b.Run("Container Manager Initialization", func(b *testing.B) {
		// Reset the singleton for each run
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			// Reset singleton to test initialization cost
			lazyValue = nil
			once = sync.Once{}

			manager, err := InitContainerRuntime()
			_ = manager
			_ = err
		}
	})

	b.Run("Singleton Access", func(b *testing.B) {
		// Initialize once
		_, _ = InitContainerRuntime()

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			manager, err := InitContainerRuntime()
			_ = manager
			_ = err
		}
	})
}

// BenchmarkRuntimeDetection benchmarks runtime detection mechanisms
func BenchmarkRuntimeDetection(b *testing.B) {
	b.Run("Environment Variable Detection", func(b *testing.B) {
		// Test different environment variable scenarios
		testCases := []struct {
			name   string
			envVar string
		}{
			{"Docker", "docker"},
			{"Podman", "podman"},
			{"Test", "test"},
			{"Host", "host"},
			{"Invalid", "invalid"},
		}

		originalEnv := os.Getenv("CONTAINER_RUNTIME")
		defer os.Setenv("CONTAINER_RUNTIME", originalEnv)

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			for _, tc := range testCases {
				os.Setenv("CONTAINER_RUNTIME", tc.envVar)

				// Skip the actual detection to avoid os.Exit
				envValue := os.Getenv("CONTAINER_RUNTIME")
				_ = envValue
			}
		}
	})

	b.Run("Executable Path Lookup", func(b *testing.B) {
		executables := []string{"docker", "podman", "nonexistent-runtime"}

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			for _, exe := range executables {
				_, err := exec.LookPath(exe)
				_ = err
			}
		}
	})

	b.Run("Complete Runtime Detection", func(b *testing.B) {
		// Unset environment variable to test full detection
		originalEnv := os.Getenv("CONTAINER_RUNTIME")
		os.Unsetenv("CONTAINER_RUNTIME")
		defer os.Setenv("CONTAINER_RUNTIME", originalEnv)

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			// We need to mock this to avoid os.Exit calls
			// Test the path lookup portion
			dockerAvailable := false
			podmanAvailable := false

			if _, err := exec.LookPath("docker"); err == nil {
				dockerAvailable = true
			}
			if _, err := exec.LookPath("podman"); err == nil {
				podmanAvailable = true
			}

			_ = dockerAvailable
			_ = podmanAvailable
		}
	})
}

// BenchmarkConcurrentManagerAccess benchmarks concurrent access to container manager
func BenchmarkConcurrentManagerAccess(b *testing.B) {
	b.Run("Concurrent Initialization", func(b *testing.B) {
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				// Reset for each iteration to test concurrent init
				lazyValue = nil
				once = sync.Once{}

				manager, err := InitContainerRuntime()
				_ = manager
				_ = err
			}
		})
	})

	b.Run("Concurrent Singleton Access", func(b *testing.B) {
		// Initialize once
		_, _ = InitContainerRuntime()

		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				manager, err := InitContainerRuntime()
				_ = manager
				_ = err
			}
		})
	})

	b.Run("Concurrent Runtime Detection", func(b *testing.B) {
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				runtime := DetectContainerRuntime()
				_ = runtime
			}
		})
	})
}

// BenchmarkRuntimeTypeOperations benchmarks runtime type operations
func BenchmarkRuntimeTypeOperations(b *testing.B) {
	b.Run("Runtime Type String Conversion", func(b *testing.B) {
		runtimes := []utils.RuntimeType{
			utils.Docker,
			utils.Podman,
			utils.Test,
			utils.Host,
		}

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			for _, runtime := range runtimes {
				str := string(runtime)
				_ = str
			}
		}
	})

	b.Run("Runtime Type Comparison", func(b *testing.B) {
		targetRuntime := utils.Docker
		testRuntimes := []utils.RuntimeType{
			utils.Docker,
			utils.Podman,
			utils.Test,
			utils.Host,
		}

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			for _, runtime := range testRuntimes {
				matches := runtime == targetRuntime
				_ = matches
			}
		}
	})
}

// BenchmarkManagerFactory benchmarks manager factory operations
func BenchmarkManagerFactory(b *testing.B) {
	b.Run("getRuntime Function", func(b *testing.B) {
		// Set environment to test runtime for consistent behavior
		originalEnv := os.Getenv("CONTAINER_RUNTIME")
		os.Setenv("CONTAINER_RUNTIME", "test")
		defer os.Setenv("CONTAINER_RUNTIME", originalEnv)

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			manager, err := getRuntime()
			_ = manager
			_ = err
		}
	})

	b.Run("Switch Statement Performance", func(b *testing.B) {
		runtimes := []utils.RuntimeType{
			utils.Docker,
			utils.Podman,
			utils.Test,
			utils.Host,
		}

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			for _, runtime := range runtimes {
				// Simulate the switch logic without actual manager creation
				var managerType string
				switch runtime {
				case utils.Docker:
					managerType = "docker"
				case utils.Podman:
					managerType = "podman"
				case utils.Test:
					managerType = "test"
				case utils.Host:
					managerType = "host"
				default:
					managerType = "unknown"
				}
				_ = managerType
			}
		}
	})
}

// BenchmarkMemoryAllocation benchmarks memory allocation patterns
func BenchmarkMemoryAllocation(b *testing.B) {
	b.Run("Manager Interface Assignment", func(b *testing.B) {
		// Set to test runtime to avoid actual Docker/Podman initialization
		originalEnv := os.Getenv("CONTAINER_RUNTIME")
		os.Setenv("CONTAINER_RUNTIME", "test")
		defer os.Setenv("CONTAINER_RUNTIME", originalEnv)

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			var manager ContainerManager
			testManager, _ := getRuntime()
			manager = testManager
			_ = manager
		}
	})

	b.Run("Context Creation", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			ctx := context.Background()
			_ = ctx
		}
	})

	b.Run("Error Handling Allocation", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			// Simulate error creation patterns
			err := makeTestError()
			_ = err
		}
	})
}

// BenchmarkStringOperations benchmarks string operations in manager
func BenchmarkStringOperations(b *testing.B) {
	b.Run("Environment Variable String Comparison", func(b *testing.B) {
		testValues := []string{"docker", "podman", "test", "host", "invalid", ""}

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			for _, value := range testValues {
				isDocker := value == "docker"
				isPodman := value == "podman"
				isTest := value == "test"
				isHost := value == "host"

				_ = isDocker
				_ = isPodman
				_ = isTest
				_ = isHost
			}
		}
	})

	b.Run("Manager Name Operations", func(b *testing.B) {
		// Set to test runtime for consistent behavior
		originalEnv := os.Getenv("CONTAINER_RUNTIME")
		os.Setenv("CONTAINER_RUNTIME", "test")
		defer os.Setenv("CONTAINER_RUNTIME", originalEnv)

		manager, _ := getRuntime()

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			name := manager.Name()
			_ = name
		}
	})
}

// BenchmarkErrorHandling benchmarks error handling patterns
func BenchmarkErrorHandling(b *testing.B) {
	b.Run("Error Creation", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			err := makeTestError()
			_ = err
		}
	})

	b.Run("Error Type Checking", func(b *testing.B) {
		testError := makeTestError()
		var nilError error

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			// Check various error conditions
			isNil := testError == nil
			isNotNil := testError != nil
			nilCheck := nilError == nil

			_ = isNil
			_ = isNotNil
			_ = nilCheck
		}
	})
}

// BenchmarkSyncOnce benchmarks sync.Once operations
func BenchmarkSyncOnce(b *testing.B) {
	b.Run("Once.Do Performance", func(b *testing.B) {
		var testOnce sync.Once
		counter := 0

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			testOnce.Do(func() {
				counter++
			})
		}

		_ = counter
	})

	b.Run("Multiple Once Instances", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			var testOnce sync.Once
			counter := 0

			testOnce.Do(func() {
				counter++
			})

			_ = counter
		}
	})
}

// Helper function to create test errors
func makeTestError() error {
	return &testError{message: "test error for benchmarking"}
}

type testError struct {
	message string
}

func (e *testError) Error() string {
	return e.message
}
