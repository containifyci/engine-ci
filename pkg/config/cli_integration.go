package config

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/containifyci/engine-ci/pkg/container"
)

// AddConfigFlags adds configuration-related flags to a Cobra command.
// This function integrates the configuration system with existing CLI commands.
func AddConfigFlags(cmd *cobra.Command, config *Config) {
	// General flags
	cmd.PersistentFlags().String("config", "", "Path to configuration file (YAML/JSON)")
	cmd.PersistentFlags().String("log-level", config.Logging.Level, "Logging level (debug, info, warn, error)")
	cmd.PersistentFlags().Bool("verbose", false, "Enable verbose logging")

	// Language-specific flags - Go
	cmd.Flags().String("go-version", config.Language.Go.Version, "Go language version")
	cmd.Flags().String("go-lint-image", config.Language.Go.LintImage, "Go linting Docker image")
	cmd.Flags().Duration("go-test-timeout", config.Language.Go.TestTimeout, "Go test timeout")
	cmd.Flags().String("go-coverage-mode", config.Language.Go.CoverageMode, "Go coverage mode (binary, text)")

	// Language-specific flags - Maven
	cmd.Flags().String("maven-prod-image", config.Language.Maven.ProdImage, "Maven production Docker image")
	cmd.Flags().String("maven-java-version", config.Language.Maven.JavaVersion, "Java version for Maven builds")
	cmd.Flags().String("maven-java-opts", config.Language.Maven.JavaOpts, "Java JVM options")

	// Language-specific flags - Python
	cmd.Flags().String("python-base-image", config.Language.Python.BaseImage, "Python base Docker image")
	cmd.Flags().String("python-version", config.Language.Python.Version, "Python version")
	cmd.Flags().Bool("python-uv-enabled", config.Language.Python.UVEnabled, "Enable UV package manager")

	// Container flags
	cmd.Flags().String("registry", config.Container.Registry, "Default container registry")
	cmd.Flags().String("pull-policy", config.Container.Images.PullPolicy, "Image pull policy")
	cmd.Flags().Duration("container-timeout", config.Container.Timeouts.Container, "Container operation timeout")
	cmd.Flags().Duration("build-timeout", config.Container.Timeouts.Build, "Build timeout")
	cmd.Flags().Duration("test-timeout", config.Container.Timeouts.Test, "Test timeout")

	// Security flags
	cmd.Flags().Bool("create-non-root-user", config.Security.UserManagement.CreateNonRootUser, "Create non-root user in containers")
	cmd.Flags().String("user-uid", config.Security.UserManagement.UID, "User ID for non-root user")
	cmd.Flags().String("user-gid", config.Security.UserManagement.GID, "Group ID for non-root user")

	// Cache flags
	cmd.Flags().Bool("cache-enabled", config.Cache.Enabled, "Enable caching")
	cmd.Flags().String("cache-cleanup-policy", config.Cache.CleanupPolicy, "Cache cleanup policy")
}

// ApplyFlagsToConfig applies CLI flag values to the configuration.
// This function updates the configuration based on command-line flags.
func ApplyFlagsToConfig(cmd *cobra.Command, config *Config) error {
	// Helper function to get flag values safely
	getString := func(name string) (string, error) {
		return cmd.Flags().GetString(name)
	}
	getBool := func(name string) (bool, error) {
		return cmd.Flags().GetBool(name)
	}
	getDuration := func(name string) (time.Duration, error) {
		return cmd.Flags().GetDuration(name)
	}

	// Apply general flags
	if logLevel, err := getString("log-level"); err == nil && cmd.Flags().Changed("log-level") {
		config.Logging.Level = logLevel
	}

	if verbose, err := getBool("verbose"); err == nil && cmd.Flags().Changed("verbose") {
		if verbose {
			config.Logging.Level = "debug"
			config.Logging.AddSource = true
		}
	}

	// Apply Go language flags
	if goVersion, err := getString("go-version"); err == nil && cmd.Flags().Changed("go-version") {
		config.Language.Go.Version = goVersion
	}

	if goLintImage, err := getString("go-lint-image"); err == nil && cmd.Flags().Changed("go-lint-image") {
		config.Language.Go.LintImage = goLintImage
	}

	if goTestTimeout, err := getDuration("go-test-timeout"); err == nil && cmd.Flags().Changed("go-test-timeout") {
		config.Language.Go.TestTimeout = goTestTimeout
	}

	if goCoverageMode, err := getString("go-coverage-mode"); err == nil && cmd.Flags().Changed("go-coverage-mode") {
		config.Language.Go.CoverageMode = goCoverageMode
	}

	// Apply Maven language flags
	if mavenProdImage, err := getString("maven-prod-image"); err == nil && cmd.Flags().Changed("maven-prod-image") {
		config.Language.Maven.ProdImage = mavenProdImage
	}

	if mavenJavaVersion, err := getString("maven-java-version"); err == nil && cmd.Flags().Changed("maven-java-version") {
		config.Language.Maven.JavaVersion = mavenJavaVersion
	}

	if mavenJavaOpts, err := getString("maven-java-opts"); err == nil && cmd.Flags().Changed("maven-java-opts") {
		config.Language.Maven.JavaOpts = mavenJavaOpts
	}

	// Apply Python language flags
	if pythonBaseImage, err := getString("python-base-image"); err == nil && cmd.Flags().Changed("python-base-image") {
		config.Language.Python.BaseImage = pythonBaseImage
	}

	if pythonVersion, err := getString("python-version"); err == nil && cmd.Flags().Changed("python-version") {
		config.Language.Python.Version = pythonVersion
	}

	if pythonUVEnabled, err := getBool("python-uv-enabled"); err == nil && cmd.Flags().Changed("python-uv-enabled") {
		config.Language.Python.UVEnabled = pythonUVEnabled
	}

	// Apply container flags
	if registry, err := getString("registry"); err == nil && cmd.Flags().Changed("registry") {
		config.Container.Registry = registry
	}

	if pullPolicy, err := getString("pull-policy"); err == nil && cmd.Flags().Changed("pull-policy") {
		config.Container.Images.PullPolicy = pullPolicy
	}

	// Apply security flags
	if createNonRootUser, err := getBool("create-non-root-user"); err == nil && cmd.Flags().Changed("create-non-root-user") {
		config.Security.UserManagement.CreateNonRootUser = createNonRootUser
	}

	if userUID, err := getString("user-uid"); err == nil && cmd.Flags().Changed("user-uid") {
		config.Security.UserManagement.UID = userUID
	}

	if userGID, err := getString("user-gid"); err == nil && cmd.Flags().Changed("user-gid") {
		config.Security.UserManagement.GID = userGID
	}

	// Apply cache flags
	if cacheEnabled, err := getBool("cache-enabled"); err == nil && cmd.Flags().Changed("cache-enabled") {
		config.Cache.Enabled = cacheEnabled
	}

	if cacheCleanupPolicy, err := getString("cache-cleanup-policy"); err == nil && cmd.Flags().Changed("cache-cleanup-policy") {
		config.Cache.CleanupPolicy = cacheCleanupPolicy
	}

	return nil
}

// CreateConfigAwareBuild creates a container.Build with configuration overrides.
// This function bridges the old container.Build structure with the new configuration system.
func CreateConfigAwareBuild(baseBuild container.Build, config *Config) container.Build {
	// Start with the base build
	newBuild := baseBuild

	// Apply configuration overrides
	if config.Container.Registry != "" {
		newBuild.Registry = config.Container.Registry
	}

	if config.Environment.Type != "" {
		newBuild.Env = config.Environment.Type
	}

	// Apply runtime configuration
	if config.Container.Runtime.Type == "podman" {
		newBuild.Runtime = "podman"
	} else {
		newBuild.Runtime = "docker"
	}

	// Apply verbose setting from logging configuration
	if config.Logging.Level == "debug" {
		newBuild.Verbose = true
	}

	return newBuild
}

// GetConfigForBuildType returns language-specific configuration for a build type.
// This function provides easy access to configuration based on build type.
func GetConfigForBuildType(config *Config, buildType container.BuildType) (interface{}, error) {
	switch buildType {
	case container.GoLang:
		return config.Language.Go, nil
	case container.Maven:
		return config.Language.Maven, nil
	case container.Python:
		return config.Language.Python, nil
	case container.Generic:
		return config.Language.Protobuf, nil
	default:
		return nil, fmt.Errorf("unsupported build type: %s", buildType)
	}
}

// UpdateConfigFromBuild updates configuration from a container.Build.
// This function provides reverse compatibility for existing code that modifies Build structs.
func UpdateConfigFromBuild(config *Config, build container.Build) {
	// Update environment type
	config.Environment.Type = build.Env

	// Update registry
	if build.Registry != "" {
		config.Container.Registry = build.Registry
	}

	// Update verbose setting
	if build.Verbose {
		config.Logging.Level = "debug"
		config.Logging.AddSource = true
	}

	// Update runtime type
	if build.Runtime == "podman" {
		config.Container.Runtime.Type = "podman"
	} else {
		config.Container.Runtime.Type = "docker"
	}
}

// PrintConfigSummary prints a summary of the current configuration.
// This is useful for debugging and verification.
func PrintConfigSummary(config *Config) {
	fmt.Printf("Engine-CI Configuration Summary:\n")
	fmt.Printf("================================\n")
	fmt.Printf("Version: %s\n", config.Version)
	fmt.Printf("Environment: %s\n", config.Environment.Type)
	fmt.Printf("Log Level: %s\n", config.Logging.Level)
	fmt.Printf("\nLanguage Configuration:\n")
	fmt.Printf("  Go Version: %s\n", config.Language.Go.Version)
	fmt.Printf("  Go Lint Image: %s\n", config.Language.Go.LintImage)
	fmt.Printf("  Maven Prod Image: %s\n", config.Language.Maven.ProdImage)
	fmt.Printf("  Maven Java Version: %s\n", config.Language.Maven.JavaVersion)
	fmt.Printf("  Python Base Image: %s\n", config.Language.Python.BaseImage)
	fmt.Printf("  Python Version: %s\n", config.Language.Python.Version)
	fmt.Printf("\nContainer Configuration:\n")
	fmt.Printf("  Registry: %s\n", config.Container.Registry)
	fmt.Printf("  Pull Policy: %s\n", config.Container.Images.PullPolicy)
	fmt.Printf("  Runtime: %s\n", config.Container.Runtime.Type)
	fmt.Printf("  Container Timeout: %s\n", config.Container.Timeouts.Container)
	fmt.Printf("  Build Timeout: %s\n", config.Container.Timeouts.Build)
	fmt.Printf("\nSecurity Configuration:\n")
	fmt.Printf("  Create Non-Root User: %t\n", config.Security.UserManagement.CreateNonRootUser)
	fmt.Printf("  User UID: %s\n", config.Security.UserManagement.UID)
	fmt.Printf("  User GID: %s\n", config.Security.UserManagement.GID)
	fmt.Printf("\nCache Configuration:\n")
	fmt.Printf("  Cache Enabled: %t\n", config.Cache.Enabled)
	fmt.Printf("  Cleanup Policy: %s\n", config.Cache.CleanupPolicy)
	fmt.Printf("  Go Cache Dir: %s\n", config.Cache.Directories.Go)
	fmt.Printf("  Maven Cache Dir: %s\n", config.Cache.Directories.Maven)
	fmt.Printf("  Python Cache Dir: %s\n", config.Cache.Directories.Python)
	fmt.Printf("\n")
}

// ValidateConfigForCommand validates configuration for a specific command context.
// This provides command-specific validation that goes beyond basic config validation.
func ValidateConfigForCommand(config *Config, buildType container.BuildType, env container.EnvType) error {
	// First validate the basic configuration
	if err := ValidateConfig(config); err != nil {
		return fmt.Errorf("basic configuration validation failed: %w", err)
	}

	// Validate for specific environment
	if err := ValidateForEnvironment(config, env); err != nil {
		return fmt.Errorf("environment-specific validation failed: %w", err)
	}

	// Validate build-type specific requirements
	switch buildType {
	case container.GoLang:
		if !isValidSemanticVersion(config.Language.Go.Version) {
			return fmt.Errorf("invalid Go version format: %s", config.Language.Go.Version)
		}
		if !ValidateDockerImage(config.Language.Go.LintImage) {
			return fmt.Errorf("invalid Go lint image format: %s", config.Language.Go.LintImage)
		}
	case container.Maven:
		if !isValidJavaVersion(config.Language.Maven.JavaVersion) {
			return fmt.Errorf("invalid Java version format: %s", config.Language.Maven.JavaVersion)
		}
		if !ValidateDockerImage(config.Language.Maven.ProdImage) {
			return fmt.Errorf("invalid Maven production image format: %s", config.Language.Maven.ProdImage)
		}
	case container.Python:
		if !isValidPythonVersion(config.Language.Python.Version) {
			return fmt.Errorf("invalid Python version format: %s", config.Language.Python.Version)
		}
		if !ValidateDockerImage(config.Language.Python.BaseImage) {
			return fmt.Errorf("invalid Python base image format: %s", config.Language.Python.BaseImage)
		}
	}

	return nil
}
