package config

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// ValidationError represents a configuration validation error
type ValidationError struct {
	Field   string
	Value   interface{}
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation error for field '%s': %s (got: %v)", e.Field, e.Message, e.Value)
}

// ValidationErrors represents multiple validation errors
type ValidationErrors []ValidationError

func (e ValidationErrors) Error() string {
	if len(e) == 0 {
		return "no validation errors"
	}

	var messages []string
	for _, err := range e {
		messages = append(messages, err.Error())
	}
	return fmt.Sprintf("%d validation errors: %s", len(e), strings.Join(messages, "; "))
}

// Validate validates the entire configuration and returns any validation errors
func (c *Config) Validate() error {
	var errors ValidationErrors

	// Validate language configurations
	errors = append(errors, c.validateLanguages()...)

	// Validate container configuration
	errors = append(errors, c.validateContainer()...)

	// Validate cache configuration
	errors = append(errors, c.validateCache()...)

	// Validate build configuration
	errors = append(errors, c.validateBuild()...)

	// Validate registry configuration
	errors = append(errors, c.validateRegistry()...)

	// Validate logging configuration
	errors = append(errors, c.validateLogging()...)

	if len(errors) > 0 {
		return errors
	}

	return nil
}

// validateLanguages validates all language configurations
func (c *Config) validateLanguages() []ValidationError {
	var errors []ValidationError

	if len(c.Languages) == 0 {
		errors = append(errors, ValidationError{
			Field:   "languages",
			Value:   c.Languages,
			Message: "at least one language must be configured",
		})
		return errors
	}

	for lang, config := range c.Languages {
		errors = append(errors, c.validateLanguageConfig(lang, config)...)
	}

	return errors
}

// validateLanguageConfig validates a single language configuration
func (c *Config) validateLanguageConfig(lang string, config *LanguageConfig) []ValidationError {
	var errors []ValidationError
	prefix := fmt.Sprintf("languages.%s", lang)

	// Validate base image
	if config.BaseImage == "" {
		errors = append(errors, ValidationError{
			Field:   prefix + ".base_image",
			Value:   config.BaseImage,
			Message: "base image cannot be empty",
		})
	} else if !isValidImageName(config.BaseImage) {
		errors = append(errors, ValidationError{
			Field:   prefix + ".base_image",
			Value:   config.BaseImage,
			Message: "invalid image name format",
		})
	}

	// Validate cache location
	if config.CacheLocation == "" {
		errors = append(errors, ValidationError{
			Field:   prefix + ".cache_location",
			Value:   config.CacheLocation,
			Message: "cache location cannot be empty",
		})
	} else if !strings.HasPrefix(config.CacheLocation, "/") {
		errors = append(errors, ValidationError{
			Field:   prefix + ".cache_location",
			Value:   config.CacheLocation,
			Message: "cache location must be an absolute path",
		})
	}

	// Validate build timeout
	if config.BuildTimeout < time.Minute {
		errors = append(errors, ValidationError{
			Field:   prefix + ".build_timeout",
			Value:   config.BuildTimeout,
			Message: "build timeout must be at least 1 minute",
		})
	} else if config.BuildTimeout > 2*time.Hour {
		errors = append(errors, ValidationError{
			Field:   prefix + ".build_timeout",
			Value:   config.BuildTimeout,
			Message: "build timeout cannot exceed 2 hours",
		})
	}

	// Validate environment variables
	for key, value := range config.Environment {
		if key == "" {
			errors = append(errors, ValidationError{
				Field:   prefix + ".environment",
				Value:   fmt.Sprintf("%s=%s", key, value),
				Message: "environment variable key cannot be empty",
			})
		}
		if strings.Contains(key, " ") {
			errors = append(errors, ValidationError{
				Field:   prefix + ".environment",
				Value:   key,
				Message: "environment variable key cannot contain spaces",
			})
		}
	}

	return errors
}

// validateContainer validates container configuration
func (c *Config) validateContainer() []ValidationError {
	var errors []ValidationError

	if c.Container == nil {
		errors = append(errors, ValidationError{
			Field:   "container",
			Value:   nil,
			Message: "container configuration is required",
		})
		return errors
	}

	// Validate runtime
	if c.Container.Runtime != "docker" && c.Container.Runtime != "podman" {
		errors = append(errors, ValidationError{
			Field:   "container.runtime",
			Value:   c.Container.Runtime,
			Message: "runtime must be either 'docker' or 'podman'",
		})
	}

	// Validate timeouts
	if c.Container.PullTimeout < 30*time.Second {
		errors = append(errors, ValidationError{
			Field:   "container.pull_timeout",
			Value:   c.Container.PullTimeout,
			Message: "pull timeout must be at least 30 seconds",
		})
	} else if c.Container.PullTimeout > 10*time.Minute {
		errors = append(errors, ValidationError{
			Field:   "container.pull_timeout",
			Value:   c.Container.PullTimeout,
			Message: "pull timeout cannot exceed 10 minutes",
		})
	}

	if c.Container.BuildTimeout < time.Minute {
		errors = append(errors, ValidationError{
			Field:   "container.build_timeout",
			Value:   c.Container.BuildTimeout,
			Message: "build timeout must be at least 1 minute",
		})
	} else if c.Container.BuildTimeout > 4*time.Hour {
		errors = append(errors, ValidationError{
			Field:   "container.build_timeout",
			Value:   c.Container.BuildTimeout,
			Message: "build timeout cannot exceed 4 hours",
		})
	}

	// Validate platform configuration
	if c.Container.PlatformConfig != nil {
		errors = append(errors, c.validatePlatformConfig()...)
	}

	return errors
}

// validatePlatformConfig validates platform configuration
func (c *Config) validatePlatformConfig() []ValidationError {
	var errors []ValidationError
	pc := c.Container.PlatformConfig

	// Validate supported platforms
	for _, platform := range pc.Supported {
		if !isValidPlatform(platform) {
			errors = append(errors, ValidationError{
				Field:   "container.platform.supported",
				Value:   platform,
				Message: "invalid platform format, expected 'os/arch'",
			})
		}
	}

	return errors
}

// validateCache validates cache configuration
func (c *Config) validateCache() []ValidationError {
	var errors []ValidationError

	if c.Cache == nil {
		errors = append(errors, ValidationError{
			Field:   "cache",
			Value:   nil,
			Message: "cache configuration is required",
		})
		return errors
	}

	// Validate base directory
	if c.Cache.BaseDir == "" {
		errors = append(errors, ValidationError{
			Field:   "cache.base_dir",
			Value:   c.Cache.BaseDir,
			Message: "cache base directory cannot be empty",
		})
	} else {
		// Expand home directory if present
		baseDir := c.Cache.BaseDir
		if strings.HasPrefix(baseDir, "~") {
			homeDir, _ := os.UserHomeDir()
			baseDir = strings.Replace(baseDir, "~", homeDir, 1)
		}
		
		// Check if we can create the directory
		if err := os.MkdirAll(baseDir, 0755); err != nil {
			errors = append(errors, ValidationError{
				Field:   "cache.base_dir",
				Value:   c.Cache.BaseDir,
				Message: fmt.Sprintf("cannot create cache directory: %v", err),
			})
		}
	}

	// Validate max size
	if c.Cache.MaxSize == "" {
		errors = append(errors, ValidationError{
			Field:   "cache.max_size",
			Value:   c.Cache.MaxSize,
			Message: "cache max size cannot be empty",
		})
	} else if !isValidSizeString(c.Cache.MaxSize) {
		errors = append(errors, ValidationError{
			Field:   "cache.max_size",
			Value:   c.Cache.MaxSize,
			Message: "invalid size format, expected format like '10GB', '512MB'",
		})
	}

	// Validate TTL
	if c.Cache.TTL < time.Hour {
		errors = append(errors, ValidationError{
			Field:   "cache.ttl",
			Value:   c.Cache.TTL,
			Message: "cache TTL must be at least 1 hour",
		})
	}

	return errors
}

// validateBuild validates build configuration
func (c *Config) validateBuild() []ValidationError {
	var errors []ValidationError

	if c.Build == nil {
		errors = append(errors, ValidationError{
			Field:   "build",
			Value:   nil,
			Message: "build configuration is required",
		})
		return errors
	}

	// Validate parallel count
	if c.Build.Parallel < 1 {
		errors = append(errors, ValidationError{
			Field:   "build.parallel",
			Value:   c.Build.Parallel,
			Message: "parallel count must be at least 1",
		})
	} else if c.Build.Parallel > 20 {
		errors = append(errors, ValidationError{
			Field:   "build.parallel",
			Value:   c.Build.Parallel,
			Message: "parallel count cannot exceed 20",
		})
	}

	// Validate timeout
	if c.Build.Timeout < time.Minute {
		errors = append(errors, ValidationError{
			Field:   "build.timeout",
			Value:   c.Build.Timeout,
			Message: "build timeout must be at least 1 minute",
		})
	} else if c.Build.Timeout > 8*time.Hour {
		errors = append(errors, ValidationError{
			Field:   "build.timeout",
			Value:   c.Build.Timeout,
			Message: "build timeout cannot exceed 8 hours",
		})
	}

	// Validate retry count
	if c.Build.RetryCount < 0 {
		errors = append(errors, ValidationError{
			Field:   "build.retry_count",
			Value:   c.Build.RetryCount,
			Message: "retry count cannot be negative",
		})
	} else if c.Build.RetryCount > 5 {
		errors = append(errors, ValidationError{
			Field:   "build.retry_count",
			Value:   c.Build.RetryCount,
			Message: "retry count cannot exceed 5",
		})
	}

	// Validate retry delay
	if c.Build.RetryDelay < time.Second {
		errors = append(errors, ValidationError{
			Field:   "build.retry_delay",
			Value:   c.Build.RetryDelay,
			Message: "retry delay must be at least 1 second",
		})
	} else if c.Build.RetryDelay > 5*time.Minute {
		errors = append(errors, ValidationError{
			Field:   "build.retry_delay",
			Value:   c.Build.RetryDelay,
			Message: "retry delay cannot exceed 5 minutes",
		})
	}

	return errors
}

// validateRegistry validates registry configuration
func (c *Config) validateRegistry() []ValidationError {
	var errors []ValidationError

	if c.Registry == nil {
		errors = append(errors, ValidationError{
			Field:   "registry",
			Value:   nil,
			Message: "registry configuration is required",
		})
		return errors
	}

	// Validate default registry
	if c.Registry.Default == "" {
		errors = append(errors, ValidationError{
			Field:   "registry.default",
			Value:   c.Registry.Default,
			Message: "default registry cannot be empty",
		})
	}

	// Validate timeout
	if c.Registry.Timeout < 30*time.Second {
		errors = append(errors, ValidationError{
			Field:   "registry.timeout",
			Value:   c.Registry.Timeout,
			Message: "registry timeout must be at least 30 seconds",
		})
	} else if c.Registry.Timeout > 10*time.Minute {
		errors = append(errors, ValidationError{
			Field:   "registry.timeout",
			Value:   c.Registry.Timeout,
			Message: "registry timeout cannot exceed 10 minutes",
		})
	}

	// Validate push configuration
	if c.Registry.PushConfig != nil {
		pc := c.Registry.PushConfig
		if pc.RetryCount < 0 || pc.RetryCount > 10 {
			errors = append(errors, ValidationError{
				Field:   "registry.push.retry_count",
				Value:   pc.RetryCount,
				Message: "push retry count must be between 0 and 10",
			})
		}
		if pc.RetryDelay < time.Second || pc.RetryDelay > 30*time.Second {
			errors = append(errors, ValidationError{
				Field:   "registry.push.retry_delay",
				Value:   pc.RetryDelay,
				Message: "push retry delay must be between 1 second and 30 seconds",
			})
		}
	}

	return errors
}

// validateLogging validates logging configuration
func (c *Config) validateLogging() []ValidationError {
	var errors []ValidationError

	if c.Logging == nil {
		// Logging configuration is optional, use defaults
		return errors
	}

	// Validate log level
	validLevels := []string{"debug", "info", "warn", "error"}
	if !contains(validLevels, c.Logging.Level) {
		errors = append(errors, ValidationError{
			Field:   "logging.level",
			Value:   c.Logging.Level,
			Message: "log level must be one of: debug, info, warn, error",
		})
	}

	// Validate log format
	validFormats := []string{"text", "json"}
	if !contains(validFormats, c.Logging.Format) {
		errors = append(errors, ValidationError{
			Field:   "logging.format",
			Value:   c.Logging.Format,
			Message: "log format must be either 'text' or 'json'",
		})
	}

	// Validate output file if specified
	if c.Logging.Output != "" && c.Logging.Output != "stdout" && c.Logging.Output != "stderr" {
		// Check if we can create/write to the log file
		dir := filepath.Dir(c.Logging.Output)
		if err := os.MkdirAll(dir, 0755); err != nil {
			errors = append(errors, ValidationError{
				Field:   "logging.output",
				Value:   c.Logging.Output,
				Message: fmt.Sprintf("cannot create log directory: %v", err),
			})
		}
	}

	return errors
}

// Helper validation functions

// isValidImageName validates a container image name
func isValidImageName(image string) bool {
	// Basic validation - should contain valid characters and format
	imageRegex := regexp.MustCompile(`^[a-z0-9]+([._-][a-z0-9]+)*(/[a-z0-9]+([._-][a-z0-9]+)*)*(:[\w\.-]+)?$`)
	return imageRegex.MatchString(image)
}

// isValidPlatform validates a platform string (os/arch format)
func isValidPlatform(platform string) bool {
	parts := strings.Split(platform, "/")
	if len(parts) != 2 {
		return false
	}
	
	validOS := []string{"linux", "darwin", "windows"}
	validArch := []string{"amd64", "arm64", "386", "arm"}
	
	return contains(validOS, parts[0]) && contains(validArch, parts[1])
}

// isValidSizeString validates a size string like "10GB", "512MB"
func isValidSizeString(size string) bool {
	sizeRegex := regexp.MustCompile(`^[0-9]+(\.[0-9]+)?(KB|MB|GB|TB|K|M|G|T)$`)
	return sizeRegex.MatchString(strings.ToUpper(size))
}

// contains checks if a string slice contains a specific string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}