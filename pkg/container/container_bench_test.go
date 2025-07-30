package container

import (
	"context"
	"io"
	"io/fs"
	"log/slog"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/containifyci/engine-ci/pkg/cri/types"
	"github.com/containifyci/engine-ci/pkg/cri/utils"
	"github.com/stretchr/testify/require"
)

// BenchmarkContainerOperations benchmarks the core container operations
func BenchmarkContainerOperations(b *testing.B) {
	// Setup a test build configuration
	build := Build{
		App:       "test-app",
		Image:     "test-image",
		ImageTag:  "test-tag",
		BuildType: GoLang,
		Runtime:   utils.Docker,
		Env:       BuildEnv,
		Registry:  "docker.io",
		Platform:  *types.GetPlatformSpec(),
		Custom:    make(Custom),
	}

	b.Run("New Container Creation", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			container := New(build)
			_ = container // Prevent compiler optimization
		}
	})

	b.Run("Parse Image Tag", func(b *testing.B) {
		testImages := []string{
			"nginx:latest",
			"redis:6.2",
			"postgres:13.4-alpine",
			"gcr.io/project/image:v1.2.3",
			"registry.example.com:5000/app:production",
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			for _, img := range testImages {
				name, tag := ParseImageTag(img)
				_ = name
				_ = tag
			}
		}
	})

	b.Run("Checksum Computation", func(b *testing.B) {
		// Generate test data of various sizes (reduced for benchmark performance)
		testData := [][]byte{
			make([]byte, 512),    // 512B
			make([]byte, 1024),   // 1KB
			make([]byte, 2048),   // 2KB
			make([]byte, 4096),   // 4KB
		}

		// Fill with deterministic data
		for i, data := range testData {
			for j := range data {
				data[j] = byte((i + j) % 256)
			}
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			for _, data := range testData {
				checksum := ComputeChecksum(data)
				_ = checksum
			}
		}
	})

	b.Run("Multiple Checksum Sum", func(b *testing.B) {
		// Create multiple checksum inputs
		checksums := make([][]byte, 10)
		for i := range checksums {
			checksums[i] = make([]byte, 32) // SHA256 size
			for j := range checksums[i] {
				checksums[i][j] = byte((i + j) % 256)
			}
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			sum := SumChecksum(checksums...)
			_ = sum
		}
	})
}

// BenchmarkTarOperations benchmarks tar directory operations
func BenchmarkTarOperations(b *testing.B) {
	// Create test filesystem with varying sizes
	testCases := []struct {
		name     string
		fileSize int
		numFiles int
	}{
		{"Small Files", 512, 5},     // 5 files, 512B each
		{"Medium Files", 1024, 10},  // 10 files, 1KB each
		{"Large Files", 2048, 20},   // 20 files, 2KB each
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			// Create test filesystem
			fsys := createTestFS(tc.numFiles, tc.fileSize)

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				buf, err := TarDir(fsys)
				require.NoError(b, err)

				// Measure the size of generated tar
				b.ReportMetric(float64(buf.Len()), "tar_bytes")

				// Prevent compiler optimization
				_ = buf
			}
		})
	}
}

// BenchmarkContainerStringOperations benchmarks string operations in container module
func BenchmarkContainerStringOperations(b *testing.B) {
	b.Run("Safe Short Operation", func(b *testing.B) {
		testStrings := []string{
			"short",
			"medium-length-string",
			"very-long-string-that-exceeds-normal-limits-and-should-be-truncated",
			strings.Repeat("x", 1000),
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			for _, str := range testStrings {
				result := safeShort(str, 8)
				_ = result
			}
		}
	})

	b.Run("Build AsFlags Operation", func(b *testing.B) {
		build := &Build{
			App:            "test-app",
			Image:          "test-image",
			ImageTag:       "v1.2.3",
			Registry:       "docker.io",
			File:           "/src/main.go",
			Folder:         "./build",
			BuildType:      GoLang,
			Env:            BuildEnv,
			Verbose:        true,
			SourcePackages: []string{"pkg1", "pkg2", "pkg3", "pkg4", "pkg5"},
			SourceFiles:    []string{"file1.go", "file2.go", "file3.go", "file4.go", "file5.go"},
		}

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			flags := build.AsFlags()
			_ = flags
		}
	})
}

// BenchmarkConcurrentOperations benchmarks concurrent container operations
func BenchmarkConcurrentOperations(b *testing.B) {
	b.Run("Concurrent Image Tag Parsing", func(b *testing.B) {
		images := []string{
			"nginx:latest",
			"redis:6.2",
			"postgres:13.4-alpine",
			"gcr.io/project/image:v1.2.3",
			"registry.example.com:5000/app:production",
		}

		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				img := images[i%len(images)]
				name, tag := ParseImageTag(img)
				_ = name
				_ = tag
				i++
			}
		})
	})

	b.Run("Concurrent Checksum Computation", func(b *testing.B) {
		testData := make([]byte, 4096) // 4KB (reduced for benchmark performance)
		for i := range testData {
			testData[i] = byte(i % 256)
		}

		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				checksum := ComputeChecksum(testData)
				_ = checksum
			}
		})
	})

	b.Run("Concurrent Container Creation", func(b *testing.B) {
		build := Build{
			App:       "test-app",
			Image:     "test-image",
			ImageTag:  "test-tag",
			BuildType: GoLang,
			Runtime:   utils.Docker,
			Env:       BuildEnv,
			Registry:  "docker.io",
			Platform:  *types.GetPlatformSpec(),
			Custom:    make(Custom),
		}

		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				container := New(build)
				_ = container
			}
		})
	})
}

// BenchmarkMemoryAllocation benchmarks memory allocation patterns
func BenchmarkMemoryAllocation(b *testing.B) {
	b.Run("Custom Map Operations", func(b *testing.B) {
		custom := make(Custom)
		custom["key1"] = []string{"value1", "value2", "value3"}
		custom["key2"] = []string{"single"}
		custom["key3"] = []string{"true"}
		custom["key4"] = []string{"42"}

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			// Test all Custom methods
			_ = custom.String("key1")
			_ = custom.Strings("key1")
			_ = custom.Bool("key3")
			_ = custom.UInt("key4")
		}
	})

	b.Run("Build Defaults Setup", func(b *testing.B) {
		baseBuild := Build{
			App:       "test-app",
			Image:     "test-image",
			ImageTag:  "test-tag",
			BuildType: GoLang,
		}

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			// Create new build each iteration to test allocation
			build := baseBuild
			build.Defaults()
		}
	})
}

// BenchmarkContextOperations benchmarks context-related operations
func BenchmarkContextOperations(b *testing.B) {
	b.Run("Context Creation and Cancellation", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			// Use shorter timeout for benchmark performance
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			_ = ctx  // Use context to avoid unused variable
			cancel() // Immediate cancellation for benchmark
		}
	})

	b.Run("Build Context Creation", func(b *testing.B) {
		build := Build{
			App:       "test-app",
			Image:     "test-image",
			ImageTag:  "test-tag",
			BuildType: GoLang,
			Runtime:   utils.Docker,
			Env:       BuildEnv,
		}

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			container := New(build)
			_ = container.ctx // Access the context
		}
	})
}

// createTestFS creates a test filesystem for benchmarking
func createTestFS(numFiles, fileSize int) fs.ReadDirFS {
	fsys := make(fstest.MapFS)

	for i := 0; i < numFiles; i++ {
		filename := filepath.Join("dir", "subdir", "file"+string(rune('0'+i%10))+".txt")
		content := make([]byte, fileSize)

		// Fill with deterministic content
		for j := range content {
			content[j] = byte((i + j) % 256)
		}

		fsys[filename] = &fstest.MapFile{
			Data: content,
		}
	}

	return fsys
}

// Helper function to disable logging during benchmarks
func init() {
	// Set a no-op handler to avoid logging overhead in benchmarks
	slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil)))
}
