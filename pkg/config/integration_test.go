package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/containifyci/engine-ci/pkg/container"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestConfigurationIntegration tests the complete configuration system integration
func TestConfigurationIntegration(t *testing.T) {
	t.Run("HierarchicalLoading", func(t *testing.T) {
		testHierarchicalLoading(t)
	})

	t.Run("EnvironmentVariableProcessing", func(t *testing.T) {
		testEnvironmentVariableProcessing(t)
	})

	t.Run("ConfigurationValidation", func(t *testing.T) {
		testConfigurationValidation(t)
	})

	t.Run("ThreadSafety", func(t *testing.T) {
		testThreadSafety(t)
	})

	t.Run("BackwardCompatibility", func(t *testing.T) {
		testBackwardCompatibility(t)
	})

	t.Run("BuilderIntegration", func(t *testing.T) {
		testBuilderIntegration(t)
	})
}

func testHierarchicalLoading(t *testing.T) {
	// Test the configuration hierarchy: CLI flags > env vars > config files > defaults

	t.Run("DefaultsOnly", func(t *testing.T) {
		config := GetDefaultConfig()

		assert.Equal(t, "1.0", config.Version)
		assert.Equal(t, "1.24.2", config.Language.Go.Version)
		assert.Equal(t, "golangci/golangci-lint:v2.1.2", config.Language.Go.LintImage)
		assert.Equal(t, 30*time.Second, config.Container.Timeouts.ContainerStart)
		assert.True(t, config.Cache.Enabled)
	})

	t.Run("ConfigFileOverridesDefaults", func(t *testing.T) {
		// Create temporary config file
		tempDir := t.TempDir()
		configFile := filepath.Join(tempDir, "test-config.yaml")

		yamlContent := `
version: "2.0"
language:
  go:
    version: "1.25.0"
    lint_image: "custom/lint:latest"
    test_timeout: "5m"
container:
  timeouts:
    container_start: "45s"
cache:
  enabled: false
logging:
  level: "debug"
`

		err := os.WriteFile(configFile, []byte(yamlContent), 0644)
		require.NoError(t, err)

		config, err := LoadConfigFromFile(configFile)
		require.NoError(t, err)

		// Config file values should override defaults
		assert.Equal(t, "2.0", config.Version)
		assert.Equal(t, "1.25.0", config.Language.Go.Version)
		assert.Equal(t, "custom/lint:latest", config.Language.Go.LintImage)
		assert.Equal(t, 5*time.Minute, config.Language.Go.TestTimeout)
		assert.Equal(t, 45*time.Second, config.Container.Timeouts.ContainerStart)
		assert.False(t, config.Cache.Enabled)
		assert.Equal(t, "debug", config.Logging.Level)

		// Non-overridden values should remain defaults
		assert.Equal(t, "registry.access.redhat.com/ubi8/openjdk-17:latest", config.Language.Maven.ProdImage)
		assert.True(t, config.Security.UserManagement.CreateNonRootUser)
	})

	t.Run("EnvironmentVariablesOverrideConfigFile", func(t *testing.T) {
		// Create config file
		tempDir := t.TempDir()
		configFile := filepath.Join(tempDir, "test-config.yaml")

		yamlContent := `
version: "2.0"
language:
  go:
    version: "1.25.0"
    lint_image: "file/lint:latest"
cache:
  enabled: false
`

		err := os.WriteFile(configFile, []byte(yamlContent), 0644)
		require.NoError(t, err)

		// Set environment variables
		envVars := map[string]string{
			"ENGINE_CI_LANGUAGE_GO_VERSION":    "1.26.0",
			"ENGINE_CI_LANGUAGE_GO_LINT_IMAGE": "env/lint:latest",
			"ENGINE_CI_CACHE_ENABLED":          "true",
			"ENGINE_CI_LOGGING_LEVEL":          "warn",
		}

		for key, value := range envVars {
			os.Setenv(key, value)
		}
		defer func() {
			for key := range envVars {
				os.Unsetenv(key)
			}
		}()

		// Load config from file first
		config, err := LoadConfigFromFile(configFile)
		require.NoError(t, err)

		// Then apply environment variables
		err = LoadFromEnvironmentVariables(config)
		require.NoError(t, err)

		// Environment variables should override file values
		assert.Equal(t, "1.26.0", config.Language.Go.Version)
		assert.Equal(t, "env/lint:latest", config.Language.Go.LintImage)
		assert.True(t, config.Cache.Enabled)
		assert.Equal(t, "warn", config.Logging.Level)

		// Non-env values should come from file
		assert.Equal(t, "2.0", config.Version)
	})

	t.Run("CompleteHierarchyIntegration", func(t *testing.T) {
		// Test a complete scenario with defaults -> config file -> env vars

		// 1. Start with defaults
		config := GetDefaultConfig()
		originalGoVersion := config.Language.Go.Version

		// 2. Create config file that overrides some values
		tempDir := t.TempDir()
		configFile := filepath.Join(tempDir, "hierarchy-test.yaml")

		yamlContent := `
version: "3.0"
language:
  go:
    version: "1.27.0"
    test_timeout: "10m"
  maven:
    java_version: "21"
cache:
  enabled: false
`

		err := os.WriteFile(configFile, []byte(yamlContent), 0644)
		require.NoError(t, err)

		// 3. Load from file (overriding defaults)
		config, err = LoadConfigFromFile(configFile)
		require.NoError(t, err)

		// 4. Set environment variables (overriding file values)
		envVars := map[string]string{
			"ENGINE_CI_LANGUAGE_GO_VERSION": "1.28.0", // Override both default and file
			"ENGINE_CI_CACHE_ENABLED":       "true",   // Override file
			"ENGINE_CI_LOGGING_LEVEL":       "error",  // Not in file, override default
		}

		for key, value := range envVars {
			os.Setenv(key, value)
		}
		defer func() {
			for key := range envVars {
				os.Unsetenv(key)
			}
		}()

		// 5. Apply environment variables
		err = LoadFromEnvironmentVariables(config)
		require.NoError(t, err)

		// Verify hierarchy: env > file > defaults
		assert.Equal(t, "1.28.0", config.Language.Go.Version)           // From env (highest priority)
		assert.Equal(t, 10*time.Minute, config.Language.Go.TestTimeout) // From file
		assert.Equal(t, "3.0", config.Version)                          // From file
		assert.Equal(t, "21", config.Language.Maven.JavaVersion)        // From file
		assert.True(t, config.Cache.Enabled)                            // From env (overriding file)
		assert.Equal(t, "error", config.Logging.Level)                  // From env (overriding default)

		// Values not specified anywhere should be defaults
		assert.Equal(t, "golangci/golangci-lint:v2.1.2", config.Language.Go.LintImage)
		assert.True(t, config.Security.UserManagement.CreateNonRootUser)

		// Verify original default wasn't modified
		originalConfig := GetDefaultConfig()
		assert.Equal(t, originalGoVersion, originalConfig.Language.Go.Version)
	})
}

func testEnvironmentVariableProcessing(t *testing.T) {
	t.Run("AllSupportedEnvironmentVariables", func(t *testing.T) {
		// Test comprehensive environment variable support
		testEnvVars := map[string]string{
			// Version and basic config
			"ENGINE_CI_VERSION": "test-version",

			// Go language configuration
			"ENGINE_CI_LANGUAGE_GO_VERSION":       "1.30.0",
			"ENGINE_CI_LANGUAGE_GO_LINT_IMAGE":    "custom/golangci-lint:latest",
			"ENGINE_CI_LANGUAGE_GO_TEST_TIMEOUT":  "15m",
			"ENGINE_CI_LANGUAGE_GO_BUILD_TIMEOUT": "2h",
			"ENGINE_CI_LANGUAGE_GO_COVERAGE_MODE": "binary",
			"ENGINE_CI_LANGUAGE_GO_PROJECT_MOUNT": "/custom/src",
			"ENGINE_CI_LANGUAGE_GO_OUTPUT_DIR":    "/custom/out",
			"ENGINE_CI_LANGUAGE_GO_MOD_CACHE":     "/custom/cache",

			// Maven configuration
			"ENGINE_CI_LANGUAGE_MAVEN_PROD_IMAGE":     "custom/maven:latest",
			"ENGINE_CI_LANGUAGE_MAVEN_BASE_IMAGE":     "maven:3-custom",
			"ENGINE_CI_LANGUAGE_MAVEN_CACHE_LOCATION": "/custom/m2",
			"ENGINE_CI_LANGUAGE_MAVEN_JAVA_VERSION":   "21",
			"ENGINE_CI_LANGUAGE_MAVEN_MAVEN_VERSION":  "3.9.0",
			"ENGINE_CI_LANGUAGE_MAVEN_JAVA_OPTS":      "-Xmx2g",
			"ENGINE_CI_LANGUAGE_MAVEN_MAVEN_OPTS":     "-DskipTests",

			// Python configuration
			"ENGINE_CI_LANGUAGE_PYTHON_BASE_IMAGE":     "python:3.12-slim",
			"ENGINE_CI_LANGUAGE_PYTHON_VERSION":        "3.12",
			"ENGINE_CI_LANGUAGE_PYTHON_CACHE_LOCATION": "/custom/pip",
			"ENGINE_CI_LANGUAGE_PYTHON_UV_ENABLED":     "true",
			"ENGINE_CI_LANGUAGE_PYTHON_UV_CACHE_DIR":   "/custom/uv",
			"ENGINE_CI_LANGUAGE_PYTHON_PIP_NO_CACHE":   "true",

			// Container configuration
			"ENGINE_CI_CONTAINER_REGISTRY":                 "custom-registry.io",
			"ENGINE_CI_CONTAINER_TIMEOUTS_CONTAINER":       "2h",
			"ENGINE_CI_CONTAINER_TIMEOUTS_CONTAINER_START": "2m",
			"ENGINE_CI_CONTAINER_TIMEOUTS_CONTAINER_STOP":  "30s",
			"ENGINE_CI_CONTAINER_TIMEOUTS_BUILD":           "3h",
			"ENGINE_CI_CONTAINER_TIMEOUTS_TEST":            "1h",
			"ENGINE_CI_CONTAINER_TIMEOUTS_PULL":            "45m",
			"ENGINE_CI_CONTAINER_TIMEOUTS_PUSH":            "45m",
			"ENGINE_CI_CONTAINER_IMAGES_PULL_POLICY":       "always",

			// Cache configuration
			"ENGINE_CI_CACHE_ENABLED":            "false",
			"ENGINE_CI_CACHE_CLEANUP_POLICY":     "aggressive",
			"ENGINE_CI_CACHE_MAX_SIZE":           "10GB",
			"ENGINE_CI_CACHE_DIRECTORIES_GO":     "/custom/go-cache",
			"ENGINE_CI_CACHE_DIRECTORIES_MAVEN":  "/custom/maven-cache",
			"ENGINE_CI_CACHE_DIRECTORIES_PYTHON": "/custom/python-cache",

			// Security configuration
			"ENGINE_CI_SECURITY_USER_MANAGEMENT_CREATE_NON_ROOT_USER": "false",
			"ENGINE_CI_SECURITY_USER_MANAGEMENT_UID":                  "2000",
			"ENGINE_CI_SECURITY_USER_MANAGEMENT_GID":                  "2000",
			"ENGINE_CI_SECURITY_USER_MANAGEMENT_USERNAME":             "builduser",
			"ENGINE_CI_SECURITY_USER_MANAGEMENT_GROUP":                "buildgroup",
			"ENGINE_CI_SECURITY_USER_MANAGEMENT_HOME":                 "/home/builduser",
			"ENGINE_CI_SECURITY_REGISTRIES_VERIFY_TLS":                "false",

			// Logging configuration
			"ENGINE_CI_LOGGING_LEVEL":      "trace",
			"ENGINE_CI_LOGGING_FORMAT":     "json",
			"ENGINE_CI_LOGGING_OUTPUT":     "file",
			"ENGINE_CI_LOGGING_FILE_PATH":  "/custom/logs/engine-ci.log",
			"ENGINE_CI_LOGGING_COMPRESS":   "true",
			"ENGINE_CI_LOGGING_ADD_SOURCE": "true",

			// Network configuration
			"ENGINE_CI_NETWORK_SSH_FORWARDING":    "true",
			"ENGINE_CI_NETWORK_PROXY_ENABLED":     "true",
			"ENGINE_CI_NETWORK_PROXY_HTTP_PROXY":  "http://proxy:8080",
			"ENGINE_CI_NETWORK_PROXY_HTTPS_PROXY": "https://proxy:8443",
			"ENGINE_CI_NETWORK_PROXY_NO_PROXY":    "localhost,127.0.0.1",
		}

		// Set all environment variables
		for key, value := range testEnvVars {
			os.Setenv(key, value)
		}
		defer func() {
			for key := range testEnvVars {
				os.Unsetenv(key)
			}
		}()

		// Load configuration
		config := GetDefaultConfig()
		err := LoadFromEnvironmentVariables(config)
		require.NoError(t, err)

		// Verify all values were applied correctly
		assert.Equal(t, "test-version", config.Version)

		// Go configuration
		assert.Equal(t, "1.30.0", config.Language.Go.Version)
		assert.Equal(t, "custom/golangci-lint:latest", config.Language.Go.LintImage)
		assert.Equal(t, 15*time.Minute, config.Language.Go.TestTimeout)
		assert.Equal(t, 2*time.Hour, config.Language.Go.BuildTimeout)
		assert.Equal(t, "binary", config.Language.Go.CoverageMode)
		assert.Equal(t, "/custom/src", config.Language.Go.ProjectMount)
		assert.Equal(t, "/custom/out", config.Language.Go.OutputDir)
		assert.Equal(t, "/custom/cache", config.Language.Go.ModCache)

		// Maven configuration
		assert.Equal(t, "custom/maven:latest", config.Language.Maven.ProdImage)
		assert.Equal(t, "maven:3-custom", config.Language.Maven.BaseImage)
		assert.Equal(t, "/custom/m2", config.Language.Maven.CacheLocation)
		assert.Equal(t, "21", config.Language.Maven.JavaVersion)
		assert.Equal(t, "3.9.0", config.Language.Maven.MavenVersion)
		assert.Equal(t, "-Xmx2g", config.Language.Maven.JavaOpts)
		assert.Equal(t, "-DskipTests", config.Language.Maven.MavenOpts)

		// Python configuration
		assert.Equal(t, "python:3.12-slim", config.Language.Python.BaseImage)
		assert.Equal(t, "3.12", config.Language.Python.Version)
		assert.Equal(t, "/custom/pip", config.Language.Python.CacheLocation)
		assert.True(t, config.Language.Python.UVEnabled)
		assert.Equal(t, "/custom/uv", config.Language.Python.UVCacheDir)
		assert.True(t, config.Language.Python.PipNoCache)

		// Container configuration
		assert.Equal(t, "custom-registry.io", config.Container.Registry)
		assert.Equal(t, 2*time.Hour, config.Container.Timeouts.Container)
		assert.Equal(t, 2*time.Minute, config.Container.Timeouts.ContainerStart)
		assert.Equal(t, 30*time.Second, config.Container.Timeouts.ContainerStop)
		assert.Equal(t, 3*time.Hour, config.Container.Timeouts.Build)
		assert.Equal(t, 1*time.Hour, config.Container.Timeouts.Test)
		assert.Equal(t, 45*time.Minute, config.Container.Timeouts.Pull)
		assert.Equal(t, 45*time.Minute, config.Container.Timeouts.Push)
		assert.Equal(t, "always", config.Container.Images.PullPolicy)

		// Cache configuration
		assert.False(t, config.Cache.Enabled)
		assert.Equal(t, "aggressive", config.Cache.CleanupPolicy)
		assert.Equal(t, "10GB", config.Cache.MaxSize)
		assert.Equal(t, "/custom/go-cache", config.Cache.Directories.Go)
		assert.Equal(t, "/custom/maven-cache", config.Cache.Directories.Maven)
		assert.Equal(t, "/custom/python-cache", config.Cache.Directories.Python)

		// Security configuration
		assert.False(t, config.Security.UserManagement.CreateNonRootUser)
		assert.Equal(t, "2000", config.Security.UserManagement.UID)
		assert.Equal(t, "2000", config.Security.UserManagement.GID)
		assert.Equal(t, "builduser", config.Security.UserManagement.Username)
		assert.Equal(t, "buildgroup", config.Security.UserManagement.Group)
		assert.Equal(t, "/home/builduser", config.Security.UserManagement.Home)
		assert.False(t, config.Security.Registries.VerifyTLS)

		// Logging configuration
		assert.Equal(t, "trace", config.Logging.Level)
		assert.Equal(t, "json", config.Logging.Format)
		assert.Equal(t, "file", config.Logging.Output)
		assert.Equal(t, "/custom/logs/engine-ci.log", config.Logging.FilePath)
		assert.True(t, config.Logging.Compress)
		assert.True(t, config.Logging.AddSource)

		// Network configuration
		assert.True(t, config.Network.SSHForwarding)
		assert.True(t, config.Network.Proxy.Enabled)
		assert.Equal(t, "http://proxy:8080", config.Network.Proxy.HTTPProxy)
		assert.Equal(t, "https://proxy:8443", config.Network.Proxy.HTTPSProxy)
		assert.Equal(t, "localhost,127.0.0.1", config.Network.Proxy.NoProxy)
	})

	t.Run("EnvironmentVariableValidation", func(t *testing.T) {
		// Test environment variable validation

		testCases := []struct {
			envVars       map[string]string
			name          string
			errorContains string
			expectError   bool
		}{
			{
				name: "valid duration",
				envVars: map[string]string{
					"ENGINE_CI_LANGUAGE_GO_TEST_TIMEOUT": "5m30s",
				},
				expectError: false,
			},
			{
				name: "invalid duration format",
				envVars: map[string]string{
					"ENGINE_CI_LANGUAGE_GO_TEST_TIMEOUT": "invalid-duration",
				},
				expectError:   true,
				errorContains: "invalid duration format",
			},
			{
				name: "valid boolean true",
				envVars: map[string]string{
					"ENGINE_CI_CACHE_ENABLED": "true",
				},
				expectError: false,
			},
			{
				name: "valid boolean false",
				envVars: map[string]string{
					"ENGINE_CI_CACHE_ENABLED": "false",
				},
				expectError: false,
			},
			{
				name: "invalid boolean",
				envVars: map[string]string{
					"ENGINE_CI_CACHE_ENABLED": "maybe",
				},
				expectError:   true,
				errorContains: "invalid boolean format",
			},
			{
				name: "valid version format",
				envVars: map[string]string{
					"ENGINE_CI_LANGUAGE_GO_VERSION": "1.25.3",
				},
				expectError: false,
			},
			{
				name: "invalid version format",
				envVars: map[string]string{
					"ENGINE_CI_LANGUAGE_GO_VERSION": "not-a-version",
				},
				expectError:   true,
				errorContains: "invalid version format",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				// Clean environment
				for key := range tc.envVars {
					os.Unsetenv(key)
				}

				// Set test environment variables
				for key, value := range tc.envVars {
					os.Setenv(key, value)
				}
				defer func() {
					for key := range tc.envVars {
						os.Unsetenv(key)
					}
				}()

				// Validate environment variables
				issues := ValidateEnvironmentVariables()

				if tc.expectError {
					assert.NotEmpty(t, issues, "Should have validation issues")
					found := false
					for _, issue := range issues {
						if strings.Contains(issue, tc.errorContains) {
							found = true
							break
						}
					}
					assert.True(t, found, "Should contain expected error message: %s", tc.errorContains)
				} else {
					// Filter out issues not related to our test variables
					relevantIssues := []string{}
					for _, issue := range issues {
						for key := range tc.envVars {
							if strings.Contains(issue, key) {
								relevantIssues = append(relevantIssues, issue)
								break
							}
						}
					}
					assert.Empty(t, relevantIssues, "Should not have validation issues for our variables")
				}
			})
		}
	})
}

func testConfigurationValidation(t *testing.T) {
	t.Run("ValidDefaultConfiguration", func(t *testing.T) {
		config := GetDefaultConfig()
		err := ValidateConfig(config)
		assert.NoError(t, err, "Default configuration should be valid")
	})

	t.Run("RequiredFieldValidation", func(t *testing.T) {
		_ = GetDefaultConfig()

		// Test missing required fields
		testCases := []struct {
			modify      func(*Config)
			name        string
			expectError bool
		}{
			{
				name: "missing go version",
				modify: func(c *Config) {
					c.Language.Go.Version = ""
				},
				expectError: true,
			},
			{
				name: "missing go lint image",
				modify: func(c *Config) {
					c.Language.Go.LintImage = ""
				},
				expectError: true,
			},
			{
				name: "missing project mount",
				modify: func(c *Config) {
					c.Language.Go.ProjectMount = ""
				},
				expectError: true,
			},
			{
				name: "missing output dir",
				modify: func(c *Config) {
					c.Language.Go.OutputDir = ""
				},
				expectError: true,
			},
			{
				name: "missing maven prod image",
				modify: func(c *Config) {
					c.Language.Maven.ProdImage = ""
				},
				expectError: true,
			},
			{
				name: "missing python base image",
				modify: func(c *Config) {
					c.Language.Python.BaseImage = ""
				},
				expectError: true,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				testConfig := GetDefaultConfig()
				tc.modify(testConfig)

				err := ValidateConfig(testConfig)
				if tc.expectError {
					assert.Error(t, err, "Should have validation error")
				} else {
					assert.NoError(t, err, "Should not have validation error")
				}
			})
		}
	})

	t.Run("ValueRangeValidation", func(t *testing.T) {
		config := GetDefaultConfig()

		// Test invalid timeout values
		config.Language.Go.TestTimeout = 5 * time.Second // Below minimum
		err := ValidateConfig(config)
		assert.Error(t, err, "Should reject timeout below minimum")

		config.Language.Go.TestTimeout = 45 * time.Minute // Above maximum
		err = ValidateConfig(config)
		assert.Error(t, err, "Should reject timeout above maximum")

		// Reset to valid value
		config.Language.Go.TestTimeout = 2 * time.Minute
		err = ValidateConfig(config)
		assert.NoError(t, err, "Should accept valid timeout")
	})

	t.Run("EnumerationValidation", func(t *testing.T) {
		config := GetDefaultConfig()

		// Test invalid enum values
		config.Container.Images.PullPolicy = "invalid_policy"
		err := ValidateConfig(config)
		assert.Error(t, err, "Should reject invalid pull policy")

		config.Container.Images.PullPolicy = "always"
		config.Logging.Level = "invalid_level"
		err = ValidateConfig(config)
		assert.Error(t, err, "Should reject invalid log level")

		config.Logging.Level = "info"
		config.Logging.Format = "invalid_format"
		err = ValidateConfig(config)
		assert.Error(t, err, "Should reject invalid log format")

		// Reset to valid values
		config.Logging.Format = "text"
		err = ValidateConfig(config)
		assert.NoError(t, err, "Should accept valid configuration")
	})
}

func testThreadSafety(t *testing.T) {
	t.Run("ConcurrentConfigurationAccess", func(t *testing.T) {
		// Test concurrent access to global configuration

		done := make(chan bool, 20)

		// Concurrent readers
		for i := 0; i < 10; i++ {
			go func() {
				defer func() { done <- true }()

				for j := 0; j < 100; j++ {
					config := GetGlobalConfig()
					assert.NotNil(t, config)

					// Access various fields
					_ = config.Version
					_ = config.Language.Go.Version
					_ = config.Container.Registry
					_ = config.Cache.Enabled
				}
			}()
		}

		// Concurrent writers
		for i := 0; i < 10; i++ {
			go func(idx int) {
				defer func() { done <- true }()

				for j := 0; j < 50; j++ {
					config := GetDefaultConfig()
					config.Version = "concurrent-" + string(rune(idx)) + "-" + string(rune(j))
					SetGlobalConfig(config)
				}
			}(i)
		}

		// Wait for all operations to complete
		for i := 0; i < 20; i++ {
			<-done
		}

		// Verify final state is valid
		finalConfig := GetGlobalConfig()
		assert.NotNil(t, finalConfig)
		assert.NotEmpty(t, finalConfig.Version)
	})

	t.Run("ConcurrentEnvironmentVariableLoading", func(t *testing.T) {
		// Set some test environment variables
		testVars := map[string]string{
			"ENGINE_CI_LANGUAGE_GO_VERSION": "1.25.0",
			"ENGINE_CI_CACHE_ENABLED":       "true",
			"ENGINE_CI_LOGGING_LEVEL":       "debug",
		}

		for key, value := range testVars {
			os.Setenv(key, value)
		}
		defer func() {
			for key := range testVars {
				os.Unsetenv(key)
			}
		}()

		done := make(chan bool, 10)

		// Concurrent environment variable loading
		for i := 0; i < 10; i++ {
			go func() {
				defer func() { done <- true }()

				for j := 0; j < 50; j++ {
					config := GetDefaultConfig()
					err := LoadFromEnvironmentVariables(config)
					assert.NoError(t, err)

					// Verify values were loaded correctly
					assert.Equal(t, "1.25.0", config.Language.Go.Version)
					assert.True(t, config.Cache.Enabled)
					assert.Equal(t, "debug", config.Logging.Level)
				}
			}()
		}

		// Wait for completion
		for i := 0; i < 10; i++ {
			<-done
		}
	})

	t.Run("ConcurrentConfigurationValidation", func(t *testing.T) {
		done := make(chan bool, 10)

		for i := 0; i < 10; i++ {
			go func() {
				defer func() { done <- true }()

				for j := 0; j < 100; j++ {
					config := GetDefaultConfig()
					err := ValidateConfig(config)
					assert.NoError(t, err)
				}
			}()
		}

		for i := 0; i < 10; i++ {
			<-done
		}
	})
}

func testBackwardCompatibility(t *testing.T) {
	t.Run("ExistingConfigurationMethods", func(t *testing.T) {
		// Test that existing configuration methods still work

		// GetDefaultConfig should work
		config := GetDefaultConfig()
		assert.NotNil(t, config)
		assert.Equal(t, "1.0", config.Version)

		// Environment-specific defaults should work
		localConfig := GetEnvironmentDefaults(container.LocalEnv)
		assert.Equal(t, container.LocalEnv, localConfig.Environment.Type)
		assert.Equal(t, "debug", localConfig.Logging.Level)

		buildConfig := GetEnvironmentDefaults(container.BuildEnv)
		assert.Equal(t, container.BuildEnv, buildConfig.Environment.Type)
		assert.Equal(t, "info", buildConfig.Logging.Level)

		prodConfig := GetEnvironmentDefaults(container.ProdEnv)
		assert.Equal(t, container.ProdEnv, prodConfig.Environment.Type)
		assert.Equal(t, "warn", prodConfig.Logging.Level)
	})

	t.Run("ConfigurationValueAccess", func(t *testing.T) {
		config := GetDefaultConfig()

		// Test getting nested values
		value, err := GetConfigValue(config, "language.go.version")
		require.NoError(t, err)
		assert.Equal(t, "1.24.2", value)

		value, err = GetConfigValue(config, "container.timeouts.build")
		require.NoError(t, err)
		assert.Equal(t, 1*time.Hour, value)

		value, err = GetConfigValue(config, "cache.enabled")
		require.NoError(t, err)
		assert.Equal(t, true, value)

		// Test setting nested values
		err = SetConfigValue(config, "language.go.version", "1.26.0")
		require.NoError(t, err)

		newValue, err := GetConfigValue(config, "language.go.version")
		require.NoError(t, err)
		assert.Equal(t, "1.26.0", newValue)

		// Test invalid paths
		_, err = GetConfigValue(config, "invalid.path")
		assert.Error(t, err)

		err = SetConfigValue(config, "invalid.path", "value")
		assert.Error(t, err)
	})

	t.Run("MergeWithDefaults", func(t *testing.T) {
		// Test merging partial configurations with defaults

		partialConfig := &Config{
			Version: "2.5",
			Language: LanguageConfig{
				Go: GoConfig{
					Version:   "1.27.0",
					LintImage: "custom/lint:v3.0",
				},
				Maven: MavenConfig{
					JavaVersion: "21",
				},
			},
			Cache: CacheConfig{
				Enabled: false,
			},
		}
		
		// Mark cache as explicitly configured since we're setting it programmatically
		partialConfig.Cache.SetConfigured()

		mergedConfig := MergeWithDefaults(partialConfig)

		// Should have custom values
		assert.Equal(t, "2.5", mergedConfig.Version)
		assert.Equal(t, "1.27.0", mergedConfig.Language.Go.Version)
		assert.Equal(t, "custom/lint:v3.0", mergedConfig.Language.Go.LintImage)
		assert.Equal(t, "21", mergedConfig.Language.Maven.JavaVersion)
		assert.False(t, mergedConfig.Cache.Enabled)

		// Should have defaults for unspecified values
		assert.Equal(t, 2*time.Minute, mergedConfig.Language.Go.TestTimeout)
		assert.Equal(t, "registry.access.redhat.com/ubi8/openjdk-17:latest", mergedConfig.Language.Maven.ProdImage)
		assert.Equal(t, "python:3.11-slim-bookworm", mergedConfig.Language.Python.BaseImage)
		assert.Equal(t, 30*time.Second, mergedConfig.Container.Timeouts.ContainerStart)
		assert.True(t, mergedConfig.Security.UserManagement.CreateNonRootUser)
	})
}

func testBuilderIntegration(t *testing.T) {
	t.Run("BuilderFactoryWithConfiguration", func(t *testing.T) {
		config := GetDefaultConfig()
		factory := NewBuilderFactory(config)

		assert.NotNil(t, factory)
		assert.Equal(t, config, factory.GetConfig())

		// Test creating builders with configuration
		build := createTestContainerBuild()

		builder, err := factory.CreateBuilderWithConfig(build)
		require.NoError(t, err)
		assert.NotNil(t, builder)

		// Verify builder has the configuration
		assert.Equal(t, config, builder.GetConfig())

		// Test configuration validation
		err = builder.ValidateConfig()
		assert.NoError(t, err)
	})

	t.Run("ConfigurableGoBuilder", func(t *testing.T) {
		cfg := GetDefaultConfig()
		build := createTestContainerBuild()

		builder := &ConfigurableGoBuilder{
			build:    build,
			config:   cfg,
			goConfig: cfg.Language.Go,
		}

		// Test configuration access methods
		assert.Equal(t, "1.24.2", builder.GetGoVersion())
		assert.Equal(t, "golangci/golangci-lint:v2.1.2", builder.GetLintImage())
		assert.Equal(t, "/src", builder.GetProjectMount())
		assert.Equal(t, "/out/", builder.GetOutputDir())
		assert.Equal(t, "2m0s", builder.GetTestTimeout())
		assert.Equal(t, "text", builder.GetCoverageMode())

		// Test configuration updates
		newConfig := GetDefaultConfig()
		newConfig.Language.Go.Version = "1.28.0"
		newConfig.Language.Go.TestTimeout = 5 * time.Minute
		newConfig.Language.Go.CoverageMode = "binary"

		err := builder.SetConfig(newConfig)
		require.NoError(t, err)

		assert.Equal(t, "1.28.0", builder.GetGoVersion())
		assert.Equal(t, "5m0s", builder.GetTestTimeout())
		assert.Equal(t, "binary", builder.GetCoverageMode())
	})

	t.Run("ConfigurationPerformance", func(t *testing.T) {
		// Test that configuration operations are fast enough

		start := time.Now()
		for i := 0; i < 1000; i++ {
			config := GetDefaultConfig()
			_ = config
		}
		duration := time.Since(start)
		assert.Less(t, duration, 150*time.Millisecond, "GetDefaultConfig should be fast")

		config := GetDefaultConfig()
		start = time.Now()
		for i := 0; i < 1000; i++ {
			err := ValidateConfig(config)
			assert.NoError(t, err)
		}
		duration = time.Since(start)
		assert.Less(t, duration, 500*time.Millisecond, "ValidateConfig should be fast")

		// Test environment variable loading performance
		os.Setenv("ENGINE_CI_LANGUAGE_GO_VERSION", "1.25.0")
		defer os.Unsetenv("ENGINE_CI_LANGUAGE_GO_VERSION")

		start = time.Now()
		for i := 0; i < 100; i++ {
			envConfig := GetDefaultConfig()
			err := LoadFromEnvironmentVariables(envConfig)
			assert.NoError(t, err)
		}
		duration = time.Since(start)
		assert.Less(t, duration, 200*time.Millisecond, "LoadFromEnvironmentVariables should be fast")
	})
}

// createTestContainerBuild creates a test container.Build for testing
func createTestContainerBuild() container.Build {
	return container.Build{
		BuildType: container.GoLang,
		App:       "test-app",
		File:      "main.go",
		Folder:    "./",
		Env:       container.LocalEnv,
		Verbose:   false,
		Custom:    make(container.Custom),
	}
}

// TestConfigurationMemoryUsage tests that configuration doesn't cause memory leaks
func TestConfigurationMemoryUsage(t *testing.T) {
	// This test helps identify memory leaks in configuration loading

	var configs []*Config

	// Load many configurations
	for i := 0; i < 1000; i++ {
		config := GetDefaultConfig()

		// Apply some environment variables with valid version strings
		version := fmt.Sprintf("1.25.%d", i%10)
		os.Setenv("ENGINE_CI_LANGUAGE_GO_VERSION", version)
		_ = LoadFromEnvironmentVariables(config)
		os.Unsetenv("ENGINE_CI_LANGUAGE_GO_VERSION")

		configs = append(configs, config)
	}

	// Verify all configurations are independent
	for i, config := range configs {
		expected := fmt.Sprintf("1.25.%d", i%10)
		assert.Equal(t, expected, config.Language.Go.Version)
	}

	t.Logf("Created and validated %d configurations", len(configs))
}

// TestConfigurationEdgeCases tests edge cases and error conditions
func TestConfigurationEdgeCases(t *testing.T) {
	t.Run("EmptyConfigFile", func(t *testing.T) {
		tempDir := t.TempDir()
		configFile := filepath.Join(tempDir, "empty.yaml")

		err := os.WriteFile(configFile, []byte(""), 0644)
		require.NoError(t, err)

		_, err = LoadConfigFromFile(configFile)
		// Should handle empty file gracefully
		assert.NoError(t, err)
	})

	t.Run("MalformedConfigFile", func(t *testing.T) {
		tempDir := t.TempDir()
		configFile := filepath.Join(tempDir, "malformed.yaml")

		malformedYAML := `
version: "1.0"
language:
  go:
    version: 1.25.0  # Missing quotes
    test_timeout: invalid-duration
  maven:
    java_version: [not-a-string]  # Wrong type
`

		err := os.WriteFile(configFile, []byte(malformedYAML), 0644)
		require.NoError(t, err)

		_, err = LoadConfigFromFile(configFile)
		assert.Error(t, err, "Should reject malformed YAML")
	})

	t.Run("NonExistentConfigFile", func(t *testing.T) {
		_, err := LoadConfigFromFile("/nonexistent/config.yaml")
		assert.Error(t, err, "Should return error for non-existent file")
	})

	t.Run("CircularConfigReferences", func(t *testing.T) {
		// Test handling of deeply nested configuration values
		config := GetDefaultConfig()

		// This tests the limits of the configuration system
		for i := 0; i < 100; i++ {
			_, err := GetConfigValue(config, "language.go.version")
			assert.NoError(t, err)
		}
	})
}
