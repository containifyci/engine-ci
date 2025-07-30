package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Config represents the centralized configuration for engine-ci.
// This eliminates scattered configuration and magic numbers throughout the codebase.
type Config struct {
	// Language-specific settings
	Languages map[string]*LanguageConfig `yaml:"languages" validate:"required"`

	// Container settings
	Container *ContainerConfig `yaml:"container" validate:"required"`

	// Cache settings
	Cache *CacheConfig `yaml:"cache" validate:"required"`

	// Build settings
	Build *BuildConfig `yaml:"build" validate:"required"`

	// Registry settings
	Registry *RegistryConfig `yaml:"registry" validate:"required"`

	// Feature flags
	Features *FeatureFlags `yaml:"features"`

	// Logging settings
	Logging *LoggingConfig `yaml:"logging"`

	// Internal fields
	configPath string // Path to the configuration file
}

// LanguageConfig contains language-specific configuration
type LanguageConfig struct {
	Environment   map[string]string `yaml:"environment"`
	BaseImage     string            `yaml:"base_image" validate:"required"`
	CacheLocation string            `yaml:"cache_location" validate:"required"`
	ProdImage     string            `yaml:"prod_image,omitempty"`
	CustomArgs    []string          `yaml:"custom_args"`
	BuildTimeout  time.Duration     `yaml:"build_timeout" validate:"min=1m,max=2h"`
	Enabled       bool              `yaml:"enabled"`
}

// ContainerConfig contains container runtime configuration
type ContainerConfig struct {
	PlatformConfig *PlatformConfig          `yaml:"platform"`
	RegistryAuth   map[string]*RegistryAuth `yaml:"registry_auth"`
	Runtime        string                   `yaml:"runtime" validate:"oneof=docker podman"`
	DefaultUser    string                   `yaml:"default_user"`
	NetworkMode    string                   `yaml:"network_mode"`
	PullTimeout    time.Duration            `yaml:"pull_timeout" validate:"min=30s,max=10m"`
	BuildTimeout   time.Duration            `yaml:"build_timeout" validate:"min=1m,max=4h"`
}

// PlatformConfig contains platform-specific settings
type PlatformConfig struct {
	Host      string   `yaml:"host"`
	Container string   `yaml:"container"`
	Supported []string `yaml:"supported"`
}

// RegistryAuth contains authentication information for container registries
type RegistryAuth struct {
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	Token    string `yaml:"token,omitempty"`
}

// CacheConfig contains cache management configuration
type CacheConfig struct {
	Strategies map[string]string `yaml:"strategies"`
	BaseDir    string            `yaml:"base_dir" validate:"required"`
	MaxSize    string            `yaml:"max_size" validate:"required"`
	TTL        time.Duration     `yaml:"ttl" validate:"min=1h"`
	Enabled    bool              `yaml:"enabled"`
}

// BuildConfig contains build execution configuration
type BuildConfig struct {
	DefaultTarget string        `yaml:"default_target"`
	Parallel      int           `yaml:"parallel" validate:"min=1,max=20"`
	Timeout       time.Duration `yaml:"timeout" validate:"min=1m,max=8h"`
	RetryCount    int           `yaml:"retry_count" validate:"min=0,max=5"`
	RetryDelay    time.Duration `yaml:"retry_delay" validate:"min=1s,max=5m"`
	FailFast      bool          `yaml:"fail_fast"`
}

// RegistryConfig contains container registry configuration
type RegistryConfig struct {
	TLS        *TLSConfig    `yaml:"tls"`
	PushConfig *PushConfig   `yaml:"push"`
	Default    string        `yaml:"default" validate:"required"`
	Mirrors    []string      `yaml:"mirrors"`
	Insecure   []string      `yaml:"insecure"`
	Timeout    time.Duration `yaml:"timeout" validate:"min=30s,max=10m"`
}

// TLSConfig contains TLS configuration for registries
type TLSConfig struct {
	CertFile           string `yaml:"cert_file"`
	KeyFile            string `yaml:"key_file"`
	CAFile             string `yaml:"ca_file"`
	InsecureSkipVerify bool   `yaml:"insecure_skip_verify"`
}

// PushConfig contains image push configuration
type PushConfig struct {
	RetryCount      int           `yaml:"retry_count" validate:"min=0,max=10"`
	RetryDelay      time.Duration `yaml:"retry_delay" validate:"min=1s,max=30s"`
	Enabled         bool          `yaml:"enabled"`
	RemoveAfterPush bool          `yaml:"remove_after_push"`
}

// FeatureFlags contains feature flag configuration
type FeatureFlags struct {
	NewLanguageBuilders   bool `yaml:"new_language_builders"`
	CentralizedConfig     bool `yaml:"centralized_config"`
	EnhancedErrorHandling bool `yaml:"enhanced_error_handling"`
	ParallelOptimization  bool `yaml:"parallel_optimization"`
	AdvancedCaching       bool `yaml:"advanced_caching"`
	DetailedLogging       bool `yaml:"detailed_logging"`
}

// LoggingConfig contains logging configuration
type LoggingConfig struct {
	Level      string `yaml:"level" validate:"oneof=debug info warn error"`
	Format     string `yaml:"format" validate:"oneof=text json"`
	Output     string `yaml:"output"`
	TimeFormat string `yaml:"time_format"`
	AddSource  bool   `yaml:"add_source"`
}

// DefaultConfig returns a configuration with sensible defaults for all settings
func DefaultConfig() *Config {
	homeDir, _ := os.UserHomeDir()
	defaultCacheDir := filepath.Join(homeDir, ".containifyci", "cache")

	return &Config{
		Languages: map[string]*LanguageConfig{
			"golang": {
				BaseImage:     "golang:1.24.2-alpine",
				CacheLocation: "/go/pkg/mod",
				BuildTimeout:  30 * time.Minute,
				Environment: map[string]string{
					"CGO_ENABLED": "0",
					"GOOS":        "linux",
					"GOARCH":      "amd64",
				},
				Enabled: true,
			},
			"python": {
				BaseImage:     "python:3.11-slim-bookworm",
				CacheLocation: "/root/.cache/pip",
				BuildTimeout:  20 * time.Minute,
				Environment: map[string]string{
					"_PIP_USE_IMPORTLIB_METADATA": "0",
					"UV_CACHE_DIR":                "/root/.cache/pip",
				},
				Enabled: true,
			},
			"maven": {
				BaseImage:     "registry.access.redhat.com/ubi8/openjdk-17:latest",
				CacheLocation: "/root/.m2",
				BuildTimeout:  45 * time.Minute,
				ProdImage:     "registry.access.redhat.com/ubi8/openjdk-17:latest",
				Enabled:       true,
			},
			"protobuf": {
				BaseImage:     "protobuf/protobuf:latest",
				CacheLocation: "/tmp/protobuf",
				BuildTimeout:  10 * time.Minute,
				Enabled:       true,
			},
		},
		Container: &ContainerConfig{
			Runtime:      "docker",
			PullTimeout:  5 * time.Minute,
			BuildTimeout: 1 * time.Hour,
			DefaultUser:  "root",
			NetworkMode:  "default",
			PlatformConfig: &PlatformConfig{
				Host:      "auto",
				Container: "auto",
				Supported: []string{"linux/amd64", "linux/arm64", "darwin/arm64"},
			},
			RegistryAuth: make(map[string]*RegistryAuth),
		},
		Cache: &CacheConfig{
			BaseDir: defaultCacheDir,
			MaxSize: "10GB",
			TTL:     24 * time.Hour,
			Strategies: map[string]string{
				"golang":   "modules",
				"python":   "pip",
				"maven":    "dependencies",
				"protobuf": "none",
			},
			Enabled: true,
		},
		Build: &BuildConfig{
			Parallel:      3,
			DefaultTarget: "all",
			Timeout:       2 * time.Hour,
			FailFast:      true,
			RetryCount:    2,
			RetryDelay:    5 * time.Second,
		},
		Registry: &RegistryConfig{
			Default: "docker.io",
			Mirrors: []string{},
			Timeout: 2 * time.Minute,
			TLS: &TLSConfig{
				InsecureSkipVerify: false,
			},
			PushConfig: &PushConfig{
				Enabled:         true,
				RetryCount:      3,
				RetryDelay:      2 * time.Second,
				RemoveAfterPush: false,
			},
		},
		Features: &FeatureFlags{
			NewLanguageBuilders:   false,
			CentralizedConfig:     true,
			EnhancedErrorHandling: false,
			ParallelOptimization:  false,
			AdvancedCaching:       false,
			DetailedLogging:       false,
		},
		Logging: &LoggingConfig{
			Level:      "info",
			Format:     "text",
			Output:     "stdout",
			AddSource:  false,
			TimeFormat: "2006-01-02T15:04:05Z07:00",
		},
	}
}

// LoadConfig loads configuration from a file, falling back to defaults if not found
func LoadConfig(path string) (*Config, error) {
	config := DefaultConfig()
	config.configPath = path

	// If no config file exists, return defaults
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return config, nil
	}

	// Read configuration file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", path, err)
	}

	// Parse YAML
	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("failed to parse config file %s: %w", path, err)
	}

	// Apply environment variable overrides
	config.applyEnvironmentOverrides()

	return config, nil
}

// applyEnvironmentOverrides applies environment variable overrides to configuration
func (c *Config) applyEnvironmentOverrides() {
	// Container runtime override
	if runtime := os.Getenv("CONTAINIFYCI_RUNTIME"); runtime != "" {
		c.Container.Runtime = runtime
	}

	// Build timeout override
	if timeout := os.Getenv("CONTAINIFYCI_BUILD_TIMEOUT"); timeout != "" {
		if d, err := time.ParseDuration(timeout); err == nil {
			c.Build.Timeout = d
		}
	}

	// Cache directory override
	if cacheDir := os.Getenv("CONTAINIFYCI_CACHE_DIR"); cacheDir != "" {
		c.Cache.BaseDir = cacheDir
	}

	// Parallel execution override
	if parallel := os.Getenv("CONTAINIFYCI_PARALLEL"); parallel != "" {
		if p := parseIntEnv(parallel, c.Build.Parallel); p > 0 && p <= 20 {
			c.Build.Parallel = p
		}
	}

	// Feature flag overrides
	c.Features.NewLanguageBuilders = parseBoolEnv("CONTAINIFYCI_NEW_BUILDERS", c.Features.NewLanguageBuilders)
	c.Features.EnhancedErrorHandling = parseBoolEnv("CONTAINIFYCI_ENHANCED_ERRORS", c.Features.EnhancedErrorHandling)
	c.Features.ParallelOptimization = parseBoolEnv("CONTAINIFYCI_PARALLEL_OPT", c.Features.ParallelOptimization)
	c.Features.DetailedLogging = parseBoolEnv("CONTAINIFYCI_DEBUG", c.Features.DetailedLogging)

	// Logging level override
	if level := os.Getenv("CONTAINIFYCI_LOG_LEVEL"); level != "" {
		level = strings.ToLower(level)
		switch level {
		case "debug", "info", "warn", "error":
			c.Logging.Level = level
		}
	}
}

// GetConfigPath returns the path to the configuration file
func (c *Config) GetConfigPath() string {
	return c.configPath
}

// IsFeatureEnabled checks if a feature flag is enabled
func (c *Config) IsFeatureEnabled(feature string) bool {
	if c.Features == nil {
		return false
	}

	switch feature {
	case "new_language_builders":
		return c.Features.NewLanguageBuilders
	case "centralized_config":
		return c.Features.CentralizedConfig
	case "enhanced_error_handling":
		return c.Features.EnhancedErrorHandling
	case "parallel_optimization":
		return c.Features.ParallelOptimization
	case "advanced_caching":
		return c.Features.AdvancedCaching
	case "detailed_logging":
		return c.Features.DetailedLogging
	default:
		return false
	}
}

// GetLanguageConfig returns the configuration for a specific language
func (c *Config) GetLanguageConfig(language string) (*LanguageConfig, bool) {
	config, exists := c.Languages[language]
	return config, exists
}

// IsLanguageEnabled checks if a language is enabled
func (c *Config) IsLanguageEnabled(language string) bool {
	if config, exists := c.Languages[language]; exists {
		return config.Enabled
	}
	return false
}

// GetEnabledLanguages returns a list of enabled languages
func (c *Config) GetEnabledLanguages() []string {
	var enabled []string
	for lang, config := range c.Languages {
		if config.Enabled {
			enabled = append(enabled, lang)
		}
	}
	return enabled
}

// SaveConfig saves the current configuration to the specified file
func (c *Config) SaveConfig(path string) error {
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Write file
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	c.configPath = path
	return nil
}
