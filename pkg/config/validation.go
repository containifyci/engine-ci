package config

import (
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/containifyci/engine-ci/pkg/container"
)

// Validation error types
type ValidationErrors []ValidationError

func (ve ValidationErrors) Error() string {
	var messages []string
	for _, err := range ve {
		messages = append(messages, err.Message)
	}
	return strings.Join(messages, "; ")
}

// ValidateConfig validates the entire configuration structure.
func ValidateConfig(config *Config) error {
	if config == nil {
		return fmt.Errorf("configuration is nil")
	}

	var errors ValidationErrors

	// Validate basic configuration structure
	if err := validateBasicConfig(config, &errors); err != nil {
		return err
	}

	// Additional custom validations
	if err := validateCustomRules(config); err != nil {
		return err
	}

	if len(errors) > 0 {
		return errors
	}

	return nil
}

// validateBasicConfig validates basic configuration requirements
func validateBasicConfig(config *Config, errors *ValidationErrors) error {
	// Validate Go configuration
	if config.Language.Go.Version == "" {
		*errors = append(*errors, ValidationError{
			Field:   "Language.Go.Version",
			Message: "Go version is required",
			Code:    "required",
		})
	}

	if config.Language.Go.LintImage == "" {
		*errors = append(*errors, ValidationError{
			Field:   "Language.Go.LintImage",
			Message: "Go lint image is required",
			Code:    "required",
		})
	}

	if config.Language.Go.ProjectMount == "" {
		*errors = append(*errors, ValidationError{
			Field:   "Language.Go.ProjectMount",
			Message: "Go project mount path is required",
			Code:    "required",
		})
	}

	if config.Language.Go.OutputDir == "" {
		*errors = append(*errors, ValidationError{
			Field:   "Language.Go.OutputDir",
			Message: "Go output directory is required",
			Code:    "required",
		})
	}

	// Validate Maven configuration
	if config.Language.Maven.ProdImage == "" {
		*errors = append(*errors, ValidationError{
			Field:   "Language.Maven.ProdImage",
			Message: "Maven production image is required",
			Code:    "required",
		}) 
	}

	if config.Language.Maven.BaseImage == "" {
		*errors = append(*errors, ValidationError{
			Field:   "Language.Maven.BaseImage",
			Message: "Maven base image is required",
			Code:    "required",
		})
	}

	// Validate Python configuration
	if config.Language.Python.BaseImage == "" {
		*errors = append(*errors, ValidationError{
			Field:   "Language.Python.BaseImage",
			Message: "Python base image is required",
			Code:    "required",
		})
	}

	if config.Language.Python.Version == "" {
		*errors = append(*errors, ValidationError{
			Field:   "Language.Python.Version",
			Message: "Python version is required",
			Code:    "required",
		})
	}

	// Validate container volumes
	if config.Container.Volumes.SourceMount == "" {
		*errors = append(*errors, ValidationError{
			Field:   "Container.Volumes.SourceMount",
			Message: "Source mount path is required",
			Code:    "required",
		})
	}

	if config.Container.Volumes.OutputDir == "" {
		*errors = append(*errors, ValidationError{
			Field:   "Container.Volumes.OutputDir",
			Message: "Output directory is required",
			Code:    "required",
		})
	}

	// Validate cache directories
	if config.Cache.Directories.Go == "" {
		*errors = append(*errors, ValidationError{
			Field:   "Cache.Directories.Go",
			Message: "Go cache directory is required",
			Code:    "required",
		})
	}

	if config.Cache.Directories.Maven == "" {
		*errors = append(*errors, ValidationError{
			Field:   "Cache.Directories.Maven",
			Message: "Maven cache directory is required",
			Code:    "required",
		})
	}

	if config.Cache.Directories.Python == "" {
		*errors = append(*errors, ValidationError{
			Field:   "Cache.Directories.Python",
			Message: "Python cache directory is required",
			Code:    "required",
		})
	}

	if config.Cache.Directories.Trivy == "" {
		*errors = append(*errors, ValidationError{
			Field:   "Cache.Directories.Trivy",
			Message: "Trivy cache directory is required",
			Code:    "required",
		})
	}

	return nil
}

// ValidateForEnvironment validates configuration for a specific environment.
func ValidateForEnvironment(config *Config, env container.EnvType) error {
	if err := ValidateConfig(config); err != nil {
		return err
	}

	// Apply environment-specific validation rules
	switch env {
	case container.LocalEnv:
		return validateLocalEnvironment(config)
	case container.BuildEnv:
		return validateBuildEnvironment(config)
	case container.ProdEnv:
		return validateProductionEnvironment(config)
	default:
		return fmt.Errorf("unsupported environment type: %s", env)
	}
}

// validateCustomRules applies custom validation rules that can't be expressed with struct tags.
func validateCustomRules(config *Config) error {
	// Validate language versions
	if err := validateLanguageVersions(config); err != nil {
		return err
	}

	// Validate timeout relationships
	if err := validateTimeouts(config); err != nil {
		return err
	}

	// Validate path configurations
	if err := validatePaths(config); err != nil {
		return err
	}

	// Validate security settings
	if err := validateSecuritySettings(config); err != nil {
		return err
	}

	// Validate resource limits
	if err := validateResourceLimits(config); err != nil {
		return err
	}

	return nil
}

// validateLanguageVersions validates language version formats and compatibility.
func validateLanguageVersions(config *Config) error {
	// Validate Go version
	if !isValidSemanticVersion(config.Language.Go.Version) {
		return fmt.Errorf("invalid Go version format: %s", config.Language.Go.Version)
	}

	// Validate Java version (can be semantic version or major version)
	if config.Language.Maven.JavaVersion != "" {
		if !isValidJavaVersion(config.Language.Maven.JavaVersion) {
			return fmt.Errorf("invalid Java version format: %s", config.Language.Maven.JavaVersion)
		}
	}

	// Validate Python version
	if !isValidPythonVersion(config.Language.Python.Version) {
		return fmt.Errorf("invalid Python version format: %s", config.Language.Python.Version)
	}

	return nil
}

// validateTimeouts ensures timeout relationships make sense.
func validateTimeouts(config *Config) error {
	timeouts := config.Container.Timeouts

	// Container start timeout should be reasonable
	if timeouts.ContainerStart > 5*time.Minute {
		return fmt.Errorf("container start timeout too long: %v (max: 5m)", timeouts.ContainerStart)
	}

	// Test timeout should be less than build timeout
	if timeouts.Test > timeouts.Build {
		return fmt.Errorf("test timeout (%v) cannot be greater than build timeout (%v)", 
			timeouts.Test, timeouts.Build)
	}

	// Pull timeout should be reasonable for large images
	if timeouts.Pull < 30*time.Second {
		return fmt.Errorf("pull timeout too short: %v (min: 30s)", timeouts.Pull)
	}

	return nil
}

// validatePaths validates path configurations.
func validatePaths(config *Config) error {
	// Validate mount paths are absolute
	if !strings.HasPrefix(config.Container.Volumes.SourceMount, "/") {
		return fmt.Errorf("source mount path must be absolute: %s", config.Container.Volumes.SourceMount)
	}

	if !strings.HasPrefix(config.Container.Volumes.OutputDir, "/") {
		return fmt.Errorf("output directory path must be absolute: %s", config.Container.Volumes.OutputDir)
	}

	// Validate cache directories
	for lang, path := range config.Cache.Directories.Custom {
		if !strings.HasPrefix(path, "/") {
			return fmt.Errorf("cache directory for %s must be absolute: %s", lang, path)
		}
	}

	return nil
}

// validateSecuritySettings validates security-related configuration.
func validateSecuritySettings(config *Config) error {
	// Validate UID/GID format
	if config.Security.UserManagement.CreateNonRootUser {
		if _, err := strconv.Atoi(config.Security.UserManagement.UID); err != nil {
			return fmt.Errorf("invalid UID format: %s", config.Security.UserManagement.UID)
		}
		
		if _, err := strconv.Atoi(config.Security.UserManagement.GID); err != nil {
			return fmt.Errorf("invalid GID format: %s", config.Security.UserManagement.GID)
		}

		// Ensure non-root UID
		uid, _ := strconv.Atoi(config.Security.UserManagement.UID)
		if uid == 0 {
			return fmt.Errorf("UID cannot be 0 (root) when creating non-root user")
		}
	}

	// Validate registry URLs
	if config.Security.Registries.DefaultRegistry != "" {
		if !isValidRegistryURL(config.Security.Registries.DefaultRegistry) {
			return fmt.Errorf("invalid default registry URL: %s", config.Security.Registries.DefaultRegistry)
		}
	}

	return nil
}

// validateResourceLimits validates resource limit formats and values.
func validateResourceLimits(config *Config) error {
	resources := config.Container.Resources

	// Validate memory formats
	if resources.MemoryLimit != "" {
		if !isValidResourceQuantity(resources.MemoryLimit) {
			return fmt.Errorf("invalid memory limit format: %s", resources.MemoryLimit)
		}
	}

	if resources.MemoryRequest != "" {
		if !isValidResourceQuantity(resources.MemoryRequest) {
			return fmt.Errorf("invalid memory request format: %s", resources.MemoryRequest)
		}
	}

	// Validate CPU formats
	if resources.CPULimit != "" {
		if !isValidCPUQuantity(resources.CPULimit) {
			return fmt.Errorf("invalid CPU limit format: %s", resources.CPULimit)
		}
	}

	if resources.CPURequest != "" {
		if !isValidCPUQuantity(resources.CPURequest) {
			return fmt.Errorf("invalid CPU request format: %s", resources.CPURequest)
		}
	}

	return nil
}

// Environment-specific validation functions
func validateLocalEnvironment(config *Config) error {
	// In local environment, we might want to be more lenient
	// For example, allow shorter timeouts for faster feedback
	return nil
}

func validateBuildEnvironment(config *Config) error {
	// Build environment should have reasonable resource limits
	if config.Container.Resources.MemoryLimit == "" {
		return fmt.Errorf("memory limit should be set in build environment")
	}

	// Cache should be enabled for build performance
	if !config.Cache.Enabled {
		return fmt.Errorf("cache should be enabled in build environment")
	}

	return nil
}

func validateProductionEnvironment(config *Config) error {
	// Production environment requires stricter settings
	if !config.Security.UserManagement.CreateNonRootUser {
		return fmt.Errorf("non-root user must be created in production environment")
	}

	if !config.Security.Registries.VerifyTLS {
		return fmt.Errorf("TLS verification must be enabled in production environment")
	}

	// Resource limits are mandatory in production
	if config.Container.Resources.MemoryLimit == "" {
		return fmt.Errorf("memory limit is required in production environment")
	}

	if config.Container.Resources.CPULimit == "" {
		return fmt.Errorf("CPU limit is required in production environment")
	}

	return nil
}

// Helper validation functions
func isValidSemanticVersion(version string) bool {
	// Simplified semantic version validation
	semverRegex := regexp.MustCompile(`^v?(\d+)\.(\d+)\.(\d+)(-[a-zA-Z0-9.-]+)?(\+[a-zA-Z0-9.-]+)?$`)
	return semverRegex.MatchString(version)
}

func isValidJavaVersion(version string) bool {
	// Java versions can be major (8, 11, 17) or semantic (1.8.0, 11.0.1)
	majorRegex := regexp.MustCompile(`^\d+$`)
	semverRegex := regexp.MustCompile(`^\d+\.\d+(\.\d+)?`)
	return majorRegex.MatchString(version) || semverRegex.MatchString(version)
}

func isValidPythonVersion(version string) bool {
	// Python versions like 3.11, 3.11.1
	pythonRegex := regexp.MustCompile(`^\d+\.\d+(\.\d+)?$`)
	return pythonRegex.MatchString(version)
}

func isValidRegistryURL(registryURL string) bool {
	// Allow simple registry names (docker.io) or full URLs
	if !strings.Contains(registryURL, "://") {
		// Simple registry name
		return regexp.MustCompile(`^[a-zA-Z0-9.-]+$`).MatchString(registryURL)
	}
	
	// Full URL
	_, err := url.Parse(registryURL)
	return err == nil
}

func isValidResourceQuantity(quantity string) bool {
	// Kubernetes-style resource quantities (1GB, 500MB, 2Gi, etc.)
	quantityRegex := regexp.MustCompile(`^\d+(\.\d+)?(Ki|Mi|Gi|Ti|Pi|Ei|K|M|G|T|P|E|B|b)?$`)
	return quantityRegex.MatchString(strings.ReplaceAll(quantity, " ", ""))
}

func isValidCPUQuantity(quantity string) bool {
	// CPU can be decimal (0.5, 1.5) or millicpu (500m, 1500m)
	cpuRegex := regexp.MustCompile(`^\d+(\.\d+)?m?$`)
	return cpuRegex.MatchString(strings.ReplaceAll(quantity, " ", ""))
}

// Validation helper functions that can be used throughout the codebase

// ValidateDockerImage validates a Docker image name format
func ValidateDockerImage(image string) bool {
	if image == "" {
		return false
	}
	// Simple docker image validation: name:tag format
	if strings.Contains(image, ":") {
		parts := strings.SplitN(image, ":", 2)
		return len(parts) == 2 && parts[0] != "" && parts[1] != ""
	}
	return true // Allow images without explicit tags
}

// ValidateAbsolutePath validates that a path is absolute
func ValidateAbsolutePath(path string) bool {
	return strings.HasPrefix(path, "/")
}

// ValidatePartialConfig validates only a subset of the configuration.
// This is useful for validating configuration updates without requiring a complete config.
func ValidatePartialConfig(config interface{}) error {
	// For now, we implement basic validation for known types
	switch cfg := config.(type) {
	case *GoConfig:
		return validateGoConfig(cfg)
	case *MavenConfig:
		return validateMavenConfig(cfg)
	case *PythonConfig:
		return validatePythonConfig(cfg)
	default:
		return nil // Skip validation for unknown types
	}
}

func validateGoConfig(config *GoConfig) error {
	var errors ValidationErrors
	
	if config.Version == "" {
		errors = append(errors, ValidationError{
			Field:   "Version",
			Message: "Go version is required",
			Code:    "required",
		})
	} else if !isValidSemanticVersion(config.Version) {
		errors = append(errors, ValidationError{
			Field:   "Version",
			Message: "Go version must be a valid semantic version",
			Code:    "invalid_format",
		})
	}
	
	if config.LintImage == "" {
		errors = append(errors, ValidationError{
			Field:   "LintImage",
			Message: "Go lint image is required",
			Code:    "required",
		})
	}
	
	if len(errors) > 0 {
		return errors
	}
	
	return nil
}

func validateMavenConfig(config *MavenConfig) error {
	var errors ValidationErrors
	
	if config.ProdImage == "" {
		errors = append(errors, ValidationError{
			Field:   "ProdImage",
			Message: "Maven production image is required",
			Code:    "required",
		})
	}
	
	if config.JavaVersion == "" {
		errors = append(errors, ValidationError{
			Field:   "JavaVersion",
			Message: "Java version is required",
			Code:    "required",
		})
	}
	
	if len(errors) > 0 {
		return errors
	}
	
	return nil
}

func validatePythonConfig(config *PythonConfig) error {
	var errors ValidationErrors
	
	if config.BaseImage == "" {
		errors = append(errors, ValidationError{
			Field:   "BaseImage",
			Message: "Python base image is required",
			Code:    "required",
		})
	}
	
	if config.Version == "" {
		errors = append(errors, ValidationError{
			Field:   "Version",
			Message: "Python version is required",
			Code:    "required",
		})
	}
	
	if len(errors) > 0 {
		return errors
	}
	
	return nil
}

// ValidateLanguageConfig validates language-specific configuration.
func ValidateLanguageConfig(langConfig *LanguageConfig) error {
	var errors ValidationErrors
	
	if err := validateGoConfig(&langConfig.Go); err != nil {
		if ve, ok := err.(ValidationErrors); ok {
			errors = append(errors, ve...)
		} else {
			errors = append(errors, ValidationError{
				Field:   "Go",
				Message: err.Error(),
				Code:    "validation_failed",
			})
		}
	}
	
	if err := validateMavenConfig(&langConfig.Maven); err != nil {
		if ve, ok := err.(ValidationErrors); ok {
			errors = append(errors, ve...)
		} else {
			errors = append(errors, ValidationError{
				Field:   "Maven",
				Message: err.Error(),
				Code:    "validation_failed",
			})
		}
	}
	
	if err := validatePythonConfig(&langConfig.Python); err != nil {
		if ve, ok := err.(ValidationErrors); ok {
			errors = append(errors, ve...)
		} else {
			errors = append(errors, ValidationError{
				Field:   "Python",
				Message: err.Error(),
				Code:    "validation_failed",
			})
		}
	}
	
	if len(errors) > 0 {
		return errors
	}
	
	return nil
}

// ValidateContainerConfig validates container-specific configuration.
func ValidateContainerConfig(containerConfig *ContainerConfig) error {
	// Basic container configuration validation
	if containerConfig.Volumes.SourceMount == "" {
		return fmt.Errorf("source mount path is required")
	}
	
	if !ValidateAbsolutePath(containerConfig.Volumes.SourceMount) {
		return fmt.Errorf("source mount path must be absolute: %s", containerConfig.Volumes.SourceMount)
	}
	
	return nil
}

// ValidateSecurityConfig validates security-specific configuration.
func ValidateSecurityConfig(securityConfig *SecurityConfig) error {
	// Create a minimal config for security validation
	config := &Config{Security: *securityConfig}
	return validateSecuritySettings(config)
}

// GetValidationRules returns documentation about validation rules.
func GetValidationRules() map[string]string {
	return map[string]string{
		"semver":        "Must be a valid semantic version (e.g., 1.2.3, v1.2.3-alpha)",
		"docker_image":  "Must be a valid Docker image name (e.g., alpine:latest, registry.io/image:tag)",
		"absolute_path": "Must be an absolute file system path starting with /",
		"resource":      "Must be a valid resource quantity (e.g., 1GB, 500MB, 2Gi)",
		"cpu":           "Must be a valid CPU quantity (e.g., 1, 0.5, 500m)",
		"required":      "Field is required and cannot be empty",
		"min":           "Value must meet minimum threshold",
		"max":           "Value must not exceed maximum threshold",
		"oneof":         "Value must be one of the specified options",
	}
}

// IsValidConfig performs a quick validation check without detailed error messages.
func IsValidConfig(config *Config) bool {
	return ValidateConfig(config) == nil
}

// ValidateAndFixConfig attempts to validate and automatically fix common configuration issues.
func ValidateAndFixConfig(config *Config) (*Config, []string, error) {
	var warnings []string
	
	// Make a copy to avoid modifying the original
	fixedConfig := *config
	
	// Fix common issues
	if fixedConfig.Container.Timeouts.ContainerStart == 0 {
		fixedConfig.Container.Timeouts.ContainerStart = 30 * time.Second
		warnings = append(warnings, "Set container start timeout to default 30s")
	}
	
	if fixedConfig.Container.Timeouts.ContainerStop == 0 {
		fixedConfig.Container.Timeouts.ContainerStop = 10 * time.Second
		warnings = append(warnings, "Set container stop timeout to default 10s")
	}
	
	// Fix empty required fields with defaults
	if fixedConfig.Language.Go.Version == "" {
		fixedConfig.Language.Go.Version = "1.24.2"
		warnings = append(warnings, "Set Go version to default 1.24.2")
	}
	
	if fixedConfig.Language.Go.LintImage == "" {
		fixedConfig.Language.Go.LintImage = "golangci/golangci-lint:v2.1.2"
		warnings = append(warnings, "Set Go lint image to default")
	}
	
	// Validate the fixed configuration
	if err := ValidateConfig(&fixedConfig); err != nil {
		return nil, warnings, err
	}
	
	return &fixedConfig, warnings, nil
}