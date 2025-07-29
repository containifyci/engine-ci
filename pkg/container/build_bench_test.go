// +build integration_test

package container

import (
	"os"
	"testing"

	"github.com/containifyci/engine-ci/pkg/cri/types"
	"github.com/containifyci/engine-ci/pkg/cri/utils"
	"github.com/containifyci/engine-ci/protos2"
)

// BenchmarkBuildOperations benchmarks build configuration operations
func BenchmarkBuildOperations(b *testing.B) {
	b.Run("Build Creation", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			build := Build{
				App:       "test-app",
				Image:     "test-image",
				ImageTag:  "v1.0.0",
				BuildType: GoLang,
				Runtime:   utils.Docker,
				Env:       BuildEnv,
			}
			_ = build
		}
	})

	b.Run("Build Defaults Setup", func(b *testing.B) {
		baseBuild := Build{
			App:       "test-app",
			Image:     "test-image",
			ImageTag:  "v1.0.0",
			BuildType: GoLang,
		}

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			build := baseBuild // Copy for each iteration
			build.Defaults()
		}
	})

	b.Run("New Service Build Creation", func(b *testing.B) {
		appName := "benchmark-service"

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			build := NewServiceBuild(appName, GoLang)
			_ = build
		}
	})

	b.Run("Specialized Build Creation", func(b *testing.B) {
		appName := "specialized-service"

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			// Test all specialized build types
			goBuild := NewGoServiceBuild(appName)
			mavenBuild := NewMavenServiceBuild(appName)
			pythonBuild := NewPythonServiceBuild(appName)

			_ = goBuild
			_ = mavenBuild
			_ = pythonBuild
		}
	})
}

// BenchmarkBuildStringOperations benchmarks string operations in build configuration
func BenchmarkBuildStringOperations(b *testing.B) {
	b.Run("AsFlags String Building", func(b *testing.B) {
		build := &Build{
			App:       "benchmark-app",
			Image:     "benchmark-image",
			ImageTag:  "v2.1.0",
			Registry:  "registry.example.com",
			File:      "/src/main.go",
			Folder:    "./build/output",
			BuildType: GoLang,
			Env:       BuildEnv,
			Verbose:   true,
			SourcePackages: []string{
				"github.com/user/repo/pkg/service",
				"github.com/user/repo/pkg/database",
				"github.com/user/repo/pkg/auth",
				"github.com/user/repo/pkg/monitoring",
				"github.com/user/repo/pkg/config",
			},
			SourceFiles: []string{
				"service.proto",
				"database.proto",
				"auth.proto",
				"monitoring.proto",
				"config.proto",
			},
		}

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			flags := build.AsFlags()
			_ = flags
		}
	})

	b.Run("ImageURI Construction", func(b *testing.B) {
		builds := []*Build{
			{Image: "nginx", ImageTag: "latest"},
			{Image: "postgres", ImageTag: "13.4-alpine"},
			{Image: "gcr.io/project/service", ImageTag: "v1.2.3-beta"},
			{Image: "registry.example.com:5000/app", ImageTag: "production-2023-12-01"},
		}

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			for _, build := range builds {
				uri := build.ImageURI()
				_ = uri
			}
		}
	})

	b.Run("CustomString Operations", func(b *testing.B) {
		build := &Build{
			Custom: Custom{
				"single":   []string{"value"},
				"multiple": []string{"value1", "value2", "value3"},
				"empty":    []string{},
				"complex":  []string{"very-long-configuration-value-for-testing"},
			},
		}

		keys := []string{"single", "multiple", "empty", "complex", "nonexistent"}

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			for _, key := range keys {
				result := build.CustomString(key)
				_ = result
			}
		}
	})

	b.Run("BuildType String Operations", func(b *testing.B) {
		buildTypes := []BuildType{GoLang, Maven, Python, Generic}

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			for _, bt := range buildTypes {
				str := bt.String()
				_ = str
			}
		}
	})

	b.Run("EnvType String Operations", func(b *testing.B) {
		envTypes := []EnvType{LocalEnv, BuildEnv, ProdEnv}

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			for _, et := range envTypes {
				str := et.String()
				_ = str
			}
		}
	})
}

// BenchmarkCustomOperations benchmarks Custom map operations
func BenchmarkCustomOperations(b *testing.B) {
	b.Run("Custom Map Access", func(b *testing.B) {
		custom := Custom{
			"string_key": []string{"string_value"},
			"bool_key":   []string{"true"},
			"uint_key":   []string{"42"},
			"multi_key":  []string{"val1", "val2", "val3", "val4", "val5"},
			"empty_key":  []string{},
		}

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			// Test all access methods
			_ = custom.String("string_key")
			_ = custom.Strings("multi_key")
			_ = custom.Bool("bool_key")
			_ = custom.UInt("uint_key")

			// Test non-existent keys
			_ = custom.String("nonexistent")
			_ = custom.Strings("nonexistent")
			_ = custom.Bool("nonexistent")
			_ = custom.UInt("nonexistent")
		}
	})

	b.Run("Large Custom Map", func(b *testing.B) {
		// Create a large custom map
		custom := make(Custom)
		for i := 0; i < 100; i++ {
			key := "key_" + string(rune('0'+i%10))
			values := make([]string, 5)
			for j := range values {
				values[j] = "value_" + string(rune('0'+j))
			}
			custom[key] = values
		}

		keys := make([]string, 10)
		for i := range keys {
			keys[i] = "key_" + string(rune('0'+i))
		}

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			for _, key := range keys {
				_ = custom.String(key)
				_ = custom.Strings(key)
			}
		}
	})
}

// BenchmarkTypeConversions benchmarks type conversion operations
func BenchmarkTypeConversions(b *testing.B) {
	b.Run("BuildType Set Operation", func(b *testing.B) {
		var buildType BuildType
		values := []string{"GoLang", "Maven", "Python", "Generic", "invalid"}

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			for _, value := range values {
				err := buildType.Set(value)
				_ = err
			}
		}
	})

	b.Run("EnvType Set Operation", func(b *testing.B) {
		var envType EnvType
		values := []string{"local", "build", "production", "invalid"}

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			for _, value := range values {
				err := envType.Set(value)
				_ = err
			}
		}
	})

	b.Run("String to UInt Conversion", func(b *testing.B) {
		custom := Custom{
			"small":   []string{"42"},
			"large":   []string{"123456789"},
			"zero":    []string{"0"},
			"invalid": []string{"not_a_number"},
		}

		keys := []string{"small", "large", "zero", "invalid"}

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			for _, key := range keys {
				// This will trigger strconv.Atoi internally
				_ = custom.UInt(key)
			}
		}
	})
}

// BenchmarkComplexBuildOperations benchmarks complex build scenarios
func BenchmarkComplexBuildOperations(b *testing.B) {
	b.Run("Full Build Configuration", func(b *testing.B) {
		// Set a reasonable limit for complex operations
		if b.N > 1000 {
			b.Skip("Skipping complex benchmark for large N to prevent timeout")
		}

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			// Create a fully configured build
			build := &Build{
				App:                "complex-service",
				Image:              "registry.example.com/org/complex-service",
				ImageTag:           "v3.2.1-rc.1",
				BuildType:          GoLang,
				Runtime:            utils.Docker,
				Organization:       "example-org",
				Registry:           "registry.example.com",
				ContainifyRegistry: "containifyci-registry",
				Env:                BuildEnv,
				File:               "/src/cmd/main.go",
				Repository:         "complex-service",
				Folder:             "./dist",
				Verbose:            true,
				Platform:           *types.GetPlatformSpec(),
				Custom: Custom{
					"envs":     []string{"DATABASE_URL", "API_KEY", "SECRET_TOKEN"},
					"secrets":  []string{"DB_PASSWORD", "JWT_SECRET"},
					"volumes":  []string{"/data", "/logs", "/config"},
					"ports":    []string{"8080", "9090", "3000"},
					"timeout":  []string{"300"},
					"replicas": []string{"3"},
				},
				Registries: map[string]*protos2.ContainerRegistry{
					"docker.io": {
						Username: "docker_user",
						Password: "docker_password",
					},
					"gcr.io": {
						Username: "_json_key",
						Password: "service_account_json",
					},
					"registry.example.com": {
						Username: "internal_user",
						Password: "internal_password",
					},
				},
				SourcePackages: []string{
					"github.com/org/service/pkg/api",
					"github.com/org/service/pkg/database",
					"github.com/org/service/pkg/auth",
					"github.com/org/service/pkg/monitoring",
					"github.com/org/service/pkg/config",
					"github.com/org/service/pkg/utils",
					"github.com/org/service/pkg/handlers",
					"github.com/org/service/pkg/middleware",
				},
				SourceFiles: []string{
					"api/service.proto",
					"api/auth.proto",
					"api/monitoring.proto",
					"database/models.proto",
					"config/settings.proto",
				},
			}

			// Apply defaults and get flags
			build.Defaults()
			flags := build.AsFlags()
			uri := build.ImageURI()

			_ = flags
			_ = uri
		}
	})

	b.Run("Build Group Operations", func(b *testing.B) {
		// Create fewer builds for benchmark performance
		builds := make([]*Build, 5)
		for i := range builds {
			builds[i] = &Build{
				App:       "service-" + string(rune('0'+i)),
				Image:     "image-" + string(rune('0'+i)),
				ImageTag:  "v1.0." + string(rune('0'+i)),
				BuildType: GoLang,
				Env:       BuildEnv,
			}
		}

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			group := &BuildGroup{Builds: builds}
			groups := BuildGroups{group}

			_ = groups
		}
	})
}

// BenchmarkEnvironmentOperations benchmarks environment-related operations
func BenchmarkEnvironmentOperations(b *testing.B) {
	// Set up test environment variables
	os.Setenv("TEST_ENV_VAR", "test_value")
	os.Setenv("ENV", "build")
	defer func() {
		os.Unsetenv("TEST_ENV_VAR")
		os.Unsetenv("ENV")
	}()

	b.Run("Environment Variable Access", func(b *testing.B) {
		keys := []string{"TEST_ENV_VAR", "ENV", "NONEXISTENT_VAR"}

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			for _, key := range keys {
				value := Getenv(key)
				_ = value
			}
		}
	})

	b.Run("GetEnv Function", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			env := getEnv()
			_ = env
		}
	})
}

// BenchmarkConcurrentBuildOperations benchmarks concurrent build operations
func BenchmarkConcurrentBuildOperations(b *testing.B) {
	b.Run("Concurrent Build Creation", func(b *testing.B) {
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			counter := 0
			for pb.Next() {
				build := Build{
					App:       "concurrent-app-" + string(rune('0'+counter%10)),
					Image:     "concurrent-image",
					ImageTag:  "v1.0.0",
					BuildType: GoLang,
					Runtime:   utils.Docker,
					Env:       BuildEnv,
				}
				_ = build
				counter++
			}
		})
	})

	b.Run("Concurrent AsFlags", func(b *testing.B) {
		build := &Build{
			App:            "concurrent-app",
			Image:          "concurrent-image",
			ImageTag:       "v1.0.0",
			Registry:       "docker.io",
			File:           "/src/main.go",
			Folder:         "./build",
			BuildType:      GoLang,
			Env:            BuildEnv,
			Verbose:        true,
			SourcePackages: []string{"pkg1", "pkg2", "pkg3"},
			SourceFiles:    []string{"file1.go", "file2.go", "file3.go"},
		}

		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				flags := build.AsFlags()
				_ = flags
			}
		})
	})

	b.Run("Concurrent Custom Access", func(b *testing.B) {
		custom := Custom{
			"key1": []string{"value1"},
			"key2": []string{"value2", "value3"},
			"key3": []string{"true"},
			"key4": []string{"42"},
		}

		build := &Build{Custom: custom}

		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				_ = build.CustomString("key1")
				_ = custom.String("key2")
				_ = custom.Bool("key3")
				_ = custom.UInt("key4")
			}
		})
	})
}
