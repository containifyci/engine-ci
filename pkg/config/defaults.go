package config

import (
	"time"

	"github.com/containifyci/engine-ci/pkg/container"
)

// GetDefaultConfig returns the default configuration for engine-ci.
// These defaults maintain backward compatibility with existing hardcoded values
// while providing sensible defaults for new installations.
func GetDefaultConfig() *Config {
	return &Config{
		Version:     "1.0",
		Language:    getDefaultLanguageConfig(),
		Container:   getDefaultContainerConfig(),
		Network:     getDefaultNetworkConfig(),
		Cache:       getDefaultCacheConfig(),
		Security:    getDefaultSecurityConfig(),
		Logging:     getDefaultLoggingConfig(),
		Environment: getDefaultEnvironmentConfig(),
	}
}

// getDefaultLanguageConfig returns default language-specific configuration.
func getDefaultLanguageConfig() LanguageConfig {
	return LanguageConfig{
		Go:       getDefaultGoConfig(),
		Maven:    getDefaultMavenConfig(),
		Python:   getDefaultPythonConfig(),
		Protobuf: getDefaultProtobufConfig(),
	}
}

// getDefaultGoConfig returns default Go language configuration.
// Values match the hardcoded constants from pkg/golang/ packages.
func getDefaultGoConfig() GoConfig {
	return GoConfig{
		Version:      "1.24.2", // DEFAULT_GO from golang packages
		LintImage:    "golangci/golangci-lint:v2.1.2", // LINT_IMAGE from golang packages
		TestTimeout:  2 * time.Minute, // 120s from buildscript templates
		BuildTimeout: 10 * time.Minute,
		CoverageMode: "text",
		BuildTags:    []string{},
		ProjectMount: "/src", // PROJ_MOUNT from golang packages
		OutputDir:    "/out/", // OUT_DIR from golang packages
		Variants: GoVariantConfig{
			Alpine: GoVariantSpec{
				BaseImage:   "golang:1.24.2-alpine",
				ImageSuffix: "alpine",
				CGOEnabled:  false,
				Dependencies: []string{"alpine:latest"},
			},
			Debian: GoVariantSpec{
				BaseImage:   "golang:1.24.2",
				ImageSuffix: "",
				CGOEnabled:  false,
				Dependencies: []string{"alpine:latest"},
			},
			DebianCGO: GoVariantSpec{
				BaseImage:   "golang:1.24.2",
				ImageSuffix: "cgo",
				CGOEnabled:  true,
				Dependencies: []string{"alpine:latest"},
			},
		},
		ModCache:    "",
		Environment: make(map[string]string),
	}
}

// getDefaultMavenConfig returns default Maven language configuration.
// Values match the hardcoded constants from pkg/maven/ package.
func getDefaultMavenConfig() MavenConfig {
	return MavenConfig{
		ProdImage:     "registry.access.redhat.com/ubi8/openjdk-17:latest", // ProdImage from maven
		BaseImage:     "maven:3-eclipse-temurin-17-alpine",
		CacheLocation: "/root/.m2/", // CacheLocation from maven
		TestTimeout:   5 * time.Minute,
		BuildTimeout:  30 * time.Minute,
		JavaVersion:   "17",
		MavenVersion:  "3.9",
		JavaOpts:      "-javaagent:/deployments/dd-java-agent.jar -Dquarkus.http.host=0.0.0.0 -Djava.util.logging.manager=org.jboss.logmanager.LogManager",
		MavenOpts:     "-Xmx1024m",
		Environment:   make(map[string]string),
	}
}

// getDefaultPythonConfig returns default Python language configuration.
// Values match the hardcoded constants from pkg/python/ package.
func getDefaultPythonConfig() PythonConfig {
	return PythonConfig{
		BaseImage:     "python:3.11-slim-bookworm", // BaseImage from python
		Version:       "3.11",
		CacheLocation: "/root/.cache/pip", // CacheLocation from python
		TestTimeout:   5 * time.Minute,
		BuildTimeout:  20 * time.Minute,
		UVEnabled:     true,
		UVCacheDir:    "/root/.cache/pip", // UV_CACHE_DIR from python
		PipNoCache:    false,
		Requirements:  []string{"requirements.txt"},
		Environment: map[string]string{
			"_PIP_USE_IMPORTLIB_METADATA": "0",
		},
	}
}

// getDefaultProtobufConfig returns default Protocol Buffers configuration.
func getDefaultProtobufConfig() ProtobufConfig {
	return ProtobufConfig{
		BaseImage:   "grpc/base:latest",
		Version:     "latest",
		ScriptPath:  "/tmp/script.sh", // Script path from protobuf
		OutputDir:   "/src",
		SourceMount: "/src",
		Environment: make(map[string]string),
	}
}

// getDefaultContainerConfig returns default container configuration.
// Values match hardcoded timeouts and paths from throughout the codebase.
func getDefaultContainerConfig() ContainerConfig {
	return ContainerConfig{
		Registry: "docker.io",
		Images: ImageConfig{
			PullPolicy: "if_not_present",
			BaseImages: map[string]string{
				"alpine": "alpine:latest",
				"debian": "debian:bookworm-slim",
				"ubuntu": "ubuntu:22.04",
			},
			TagPolicy: "latest",
		},
		Timeouts: TimeoutConfig{
			Container:      30 * time.Second, // Container start/stop timeouts from container.go
			ContainerStart: 30 * time.Second, // context.WithTimeout(30*time.Second) from container.go
			ContainerStop:  10 * time.Second, // context.WithTimeout(10*time.Second) from container.go
			Build:          1 * time.Hour,
			Test:           2 * time.Minute,   // 120s from buildscript
			Pull:           5 * time.Minute,
			Push:           10 * time.Minute,
			Script:         30 * time.Second,
		},
		Resources: ResourceConfig{
			MemoryLimit:   "2GB",
			MemoryRequest: "512MB",
			CPULimit:      "2",
			CPURequest:    "0.5",
			DiskLimit:     "10GB",
		},
		Volumes: VolumeConfig{
			SourceMount:  "/src",           // PROJ_MOUNT from golang packages
			OutputDir:    "/out",           // OUT_DIR from golang packages
			CacheDir:     "/cache",
			TempDir:      "/tmp",
			ScriptPath:   "/tmp/script.sh", // Script path used throughout codebase
			CustomMounts: make(map[string]string),
		},
		Runtime: RuntimeConfig{
			Type:       "docker",
			SocketPath: "/var/run/docker.sock",
			Options:    make(map[string]string),
		},
	}
}

// getDefaultNetworkConfig returns default network configuration.
func getDefaultNetworkConfig() NetworkConfig {
	return NetworkConfig{
		SSHForwarding: true,
		Proxy: ProxyConfig{
			Enabled:    false,
			HTTPProxy:  "",
			HTTPSProxy: "",
			NoProxy:    "localhost,127.0.0.1,*.local",
		},
		DNS: DNSConfig{
			Servers:       []string{},
			SearchDomains: []string{},
			Options:       []string{},
		},
	}
}

// getDefaultCacheConfig returns default cache configuration.
// Values match hardcoded cache paths from throughout the codebase.
func getDefaultCacheConfig() CacheConfig {
	return CacheConfig{
		Enabled:       true,
		CleanupPolicy: "30d",
		MaxSize:       "10GB",
		Directories: CacheDirectories{
			Go:     "/var/cache/go",
			Maven:  "/var/cache/maven",
			Python: "/var/cache/pip",
			Trivy:  "/var/cache/trivy", // ~/.trivy/cache from trivy.go
			Docker: "/var/cache/docker",
			Custom: make(map[string]string),
		},
		Permissions: CachePermissions{
			Mode:  "0755",
			Owner: "root",
			Group: "root",
		},
	}
}

// getDefaultSecurityConfig returns default security configuration.
// Values match hardcoded user settings from BaseBuilder.
func getDefaultSecurityConfig() SecurityConfig {
	return SecurityConfig{
		UserManagement: UserManagementConfig{
			CreateNonRootUser: true,
			UID:               "11211", // UID from BaseBuilder.SetupUserInContainer
			GID:               "1121",  // GID from BaseBuilder.SetupUserInContainer
			Username:          "app",   // Username from BaseBuilder.SetupUserInContainer
			Group:             "app",   // Group from BaseBuilder.SetupUserInContainer
			Home:              "/app",
			Shell:             "/bin/sh",
		},
		Registries: RegistryConfig{
			VerifyTLS:           true,
			AuthConfigPath:      "~/.docker/config.json",
			DefaultRegistry:     "docker.io",
			InsecureRegistries:  []string{},
		},
		Secrets: SecretsConfig{
			Provider: "env",
			FileConfig: SecretsFileConfig{
				Path:        "/etc/secrets",
				Permissions: "0600",
			},
			VaultConfig: SecretsVaultConfig{
				Address: "",
				Path:    "secret/engine-ci",
				Token:   "",
			},
		},
		Scanning: ScanningConfig{
			Trivy: getTrivyDefaultConfig(),
		},
	}
}

// getTrivyDefaultConfig returns default Trivy configuration.
// Values match hardcoded settings from pkg/trivy/ package.
func getTrivyDefaultConfig() TrivyConfig {
	return TrivyConfig{
		Image:         "aquasec/trivy:latest", // IMAGE from trivy
		CacheDir:      "/root/.cache/trivy",   // TRIVY_CACHE_DIR from trivy
		Severity:      []string{"CRITICAL", "HIGH"}, // Default severity from trivy script
		IgnoreUnfixed: true,  // --ignore-unfixed from trivy script
		Timeout:       5 * time.Minute,
		Scanners:      []string{"vuln"}, // --scanners vuln from trivy script
		Format:        "json", // --format json from trivy script
		OutputPath:    "/usr/src/trivy.json", // --output path from trivy script
		Environment: map[string]string{
			"TRIVY_INSECURE":  "true", // From trivy.go
			"TRIVY_NON_SSL":   "true", // From trivy.go
		},
	}
}

// getDefaultLoggingConfig returns default logging configuration.
func getDefaultLoggingConfig() LoggingConfig {
	return LoggingConfig{
		Level:           "info",
		Format:          "structured",
		Output:          "stdout",
		FilePath:        "",
		MaxSize:         "100MB",
		MaxBackups:      3,
		MaxAge:          "30d",
		Compress:        true,
		AddSource:       false,
		SampleRate:      1.0,
		ProgressFormat:  "auto", // Progress format from root.go
		CustomFields:    make(map[string]string),
	}
}

// getDefaultEnvironmentConfig returns default environment configuration.
func getDefaultEnvironmentConfig() EnvironmentConfig {
	return EnvironmentConfig{
		Type: container.BuildEnv, // Default to build environment
		Profiles: ProfilesConfig{
			Local: EnvironmentProfile{
				Verbose:                true,
				PullPolicy:            "never",
				SecurityHardening:     false,
				ResourceLimitsEnforced: false,
				LogLevel:              "debug",
				CacheEnabled:          true,
				ParallelBuilds:        1,
				CustomSettings:        make(map[string]interface{}),
			},
			Build: EnvironmentProfile{
				Verbose:                false,
				PullPolicy:            "if_not_present",
				SecurityHardening:     false,
				ResourceLimitsEnforced: true,
				LogLevel:              "info",
				CacheEnabled:          true,
				ParallelBuilds:        2,
				CustomSettings:        make(map[string]interface{}),
			},
			Production: EnvironmentProfile{
				Verbose:                false,
				PullPolicy:            "always",
				SecurityHardening:     true,
				ResourceLimitsEnforced: true,
				LogLevel:              "warn",
				CacheEnabled:          true,
				ParallelBuilds:        4,
				CustomSettings:        make(map[string]interface{}),
			},
		},
	}
}

// GetEnvironmentDefaults returns environment-specific default overrides.
// This allows different defaults based on the deployment environment.
func GetEnvironmentDefaults(env container.EnvType) *Config {
	config := GetDefaultConfig()
	
	switch env {
	case container.LocalEnv:
		applyLocalDefaults(config)
	case container.BuildEnv:
		applyBuildDefaults(config)
	case container.ProdEnv:
		applyProductionDefaults(config)
	}
	
	return config
}

// applyLocalDefaults applies local environment-specific defaults.
func applyLocalDefaults(config *Config) {
	config.Environment.Type = container.LocalEnv
	config.Logging.Level = "debug"
	config.Logging.AddSource = true
	config.Container.Images.PullPolicy = "never"
	config.Security.UserManagement.CreateNonRootUser = false
	config.Container.Resources.MemoryLimit = "1GB"
	config.Container.Resources.CPULimit = "1"
}

// applyBuildDefaults applies build environment-specific defaults.
func applyBuildDefaults(config *Config) {
	config.Environment.Type = container.BuildEnv
	config.Logging.Level = "info"
	config.Container.Images.PullPolicy = "if_not_present"
	config.Cache.Enabled = true
	config.Container.Timeouts.Build = 30 * time.Minute
}

// applyProductionDefaults applies production environment-specific defaults.
func applyProductionDefaults(config *Config) {
	config.Environment.Type = container.ProdEnv
	config.Logging.Level = "warn"
	config.Container.Images.PullPolicy = "always"
	config.Security.UserManagement.CreateNonRootUser = true
	config.Security.Registries.VerifyTLS = true
	config.Container.Resources.MemoryLimit = "4GB"
	config.Container.Resources.CPULimit = "4"
	config.Container.Timeouts.Build = 1 * time.Hour
}

// GetLanguageDefaults returns language-specific default configuration.
// This provides a way to get defaults for individual language builders.
func GetLanguageDefaults(language string) interface{} {
	switch language {
	case "go", "golang":
		return getDefaultGoConfig()
	case "maven", "java":
		return getDefaultMavenConfig()
	case "python":
		return getDefaultPythonConfig()
	case "protobuf", "proto":
		return getDefaultProtobufConfig()
	default:
		return nil
	}
}

// GetContainerDefaults returns container-specific default configuration.
func GetContainerDefaults() ContainerConfig {
	return getDefaultContainerConfig()
}

// GetSecurityDefaults returns security-specific default configuration.
func GetSecurityDefaults() SecurityConfig {
	return getDefaultSecurityConfig()
}

// GetCacheDefaults returns cache-specific default configuration.
func GetCacheDefaults() CacheConfig {
	return getDefaultCacheConfig()
}

// MergeWithDefaults merges the provided config with defaults.
// Missing fields in the provided config will be filled with default values.
func MergeWithDefaults(config *Config) *Config {
	defaults := GetDefaultConfig()
	return mergeConfigs(defaults, config)
}

// mergeConfigs merges two configurations, with override taking precedence.
func mergeConfigs(base, override *Config) *Config {
	if override == nil {
		return base
	}
	
	// Create a new config starting with base
	result := *base
	
	// Apply overrides for each section
	if override.Version != "" {
		result.Version = override.Version
	}
	
	// Merge language configs
	result.Language = mergeLanguageConfigs(base.Language, override.Language)
	result.Container = mergeContainerConfigs(base.Container, override.Container)
	result.Network = mergeNetworkConfigs(base.Network, override.Network)
	result.Cache = mergeCacheConfigs(base.Cache, override.Cache)
	result.Security = mergeSecurityConfigs(base.Security, override.Security)
	result.Logging = mergeLoggingConfigs(base.Logging, override.Logging)
	result.Environment = mergeEnvironmentConfigs(base.Environment, override.Environment)
	
	return &result
}

// Helper merge functions for different config sections
func mergeLanguageConfigs(base, override LanguageConfig) LanguageConfig {
	result := base
	// Implement field-by-field merging as needed
	// This is a simplified version - full implementation would merge each nested struct
	return result
}

func mergeContainerConfigs(base, override ContainerConfig) ContainerConfig {
	result := base
	// Implement field-by-field merging as needed
	return result
}

func mergeNetworkConfigs(base, override NetworkConfig) NetworkConfig {
	result := base
	// Implement field-by-field merging as needed
	return result
}

func mergeCacheConfigs(base, override CacheConfig) CacheConfig {
	result := base
	// Implement field-by-field merging as needed
	return result
}

func mergeSecurityConfigs(base, override SecurityConfig) SecurityConfig {
	result := base
	// Implement field-by-field merging as needed
	return result
}

func mergeLoggingConfigs(base, override LoggingConfig) LoggingConfig {
	result := base
	// Implement field-by-field merging as needed
	return result
}

func mergeEnvironmentConfigs(base, override EnvironmentConfig) EnvironmentConfig {
	result := base
	// Implement field-by-field merging as needed
	return result
}