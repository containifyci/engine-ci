// Package build defines the core interfaces and contracts for the engine-ci
// build system.
//
// This package provides the foundational interfaces that eliminate code
// duplication across language packages and enable a modular, extensible
// build architecture:
//
//   - LanguageBuilder: Contract for all language-specific builders
//   - BuildStep: Individual build pipeline steps with dependencies
//   - ConfigProvider: Centralized configuration management
//   - CacheManager: Language-specific cache management
//   - BuildOrchestrator: Multi-step build coordination with parallel execution
//   - ErrorHandler: Standardized error handling replacing os.Exit(1) patterns
//   - Validator: Configuration and component validation
//
// These interfaces replace the previous scattered, duplicated code patterns
// with a cohesive architecture that supports:
//
//   - Interface-driven development for better testing and modularity
//   - Shared functionality through composition rather than inheritance
//   - Standardized error handling with context and recovery
//   - Configurable build pipelines with dependency resolution
//   - Language-agnostic caching strategies
//   - Parallel execution with proper synchronization
//
// Example usage:
//
//	// Implement a language builder
//	type PythonBuilder struct {
//	    *language.BaseLanguageBuilder
//	}
//
//	func (p *PythonBuilder) BuildImage(ctx context.Context) (string, error) {
//	    // Language-specific build logic
//	}
//
//	// Use build orchestrator
//	orchestrator := NewBuildOrchestrator()
//	orchestrator.AddStep(pythonStep)
//	orchestrator.AddStep(dockerStep)
//	
//	if err := orchestrator.Execute(ctx); err != nil {
//	    // Handle build failures with proper error context
//	}
package build

import (
	"context"
	"time"
)

// LanguageBuilder defines the contract for all language-specific builders.
// This interface eliminates code duplication across language packages by
// providing a common set of operations that all language builders must implement.
type LanguageBuilder interface {
	// Core identification methods
	Name() string  // Returns the name of the language (e.g., "golang", "python", "maven")
	IsAsync() bool // Indicates if this builder can run asynchronously

	// Image management methods
	BaseImage() string                  // Returns the base container image for this language
	Images() []string                   // Returns all images required by this builder
	BuildImage() (string, error)        // Builds the intermediate language-specific image
	ComputeImageTag(data []byte) string // Computes a deterministic tag from dockerfile content

	// Container operations
	Pull() error            // Pulls required base images
	Build() (string, error) // Executes the build process and returns the resulting image ID

	// Configuration methods
	CacheLocation() string        // Returns the default cache directory inside the container
	DefaultEnvironment() []string // Returns default environment variables for this language
	BuildTimeout() time.Duration  // Returns the maximum build time allowed

	// Build script generation
	BuildScript() string // Generates the build script for this language

	// Lifecycle management
	PreBuild() error  // Executes pre-build setup operations
	PostBuild() error // Executes post-build cleanup operations

	// Validation
	Validate() error // Validates the builder configuration and dependencies
}

// BuildStep represents a single step in the build pipeline.
// This interface enables better orchestration, dependency management,
// and validation of build operations.
type BuildStep interface {
	// Core identification
	Name() string // Returns a human-readable name for this step

	// Execution
	Execute(ctx context.Context) error // Executes the build step

	// Dependencies and ordering
	Dependencies() []string // Returns the names of steps this step depends on
	IsAsync() bool          // Indicates if this step can run in parallel with others

	// Configuration
	Timeout() time.Duration // Returns the maximum execution time for this step

	// Validation
	Validate() error // Validates the step configuration before execution
}

// ConfigProvider manages configuration for different components.
// This interface centralizes configuration management and enables
// validation and type-safe configuration access.
type ConfigProvider interface {
	// Generic configuration access
	Get(key string) (interface{}, error)
	Has(key string) bool

	// Type-safe configuration access
	GetString(key string) (string, error)
	GetInt(key string) (int, error)
	GetBool(key string) (bool, error)
	GetDuration(key string) (time.Duration, error)
	GetStringSlice(key string) ([]string, error)
	GetStringMap(key string) (map[string]string, error)

	// Configuration with defaults
	GetStringWithDefault(key, defaultValue string) string
	GetIntWithDefault(key string, defaultValue int) int
	GetBoolWithDefault(key string, defaultValue bool) bool
	GetDurationWithDefault(key string, defaultValue time.Duration) time.Duration

	// Validation and management
	Validate() error       // Validates the entire configuration
	Reload() error         // Reloads configuration from source
	GetConfigPath() string // Returns the path to the configuration file
}

// CacheManager handles caching strategies for different languages.
// This interface standardizes cache management across all language builders
// and provides configurable caching strategies.
type CacheManager interface {
	// Cache directory management
	GetCacheDir(language string) (string, error) // Returns the cache directory for a language
	EnsureCacheDir(language string) error        // Ensures the cache directory exists
	CleanCache(language string) error            // Cleans the cache for a specific language
	CleanAllCaches() error                       // Cleans all language caches

	// Cache key generation
	CacheKey(language string, dependencies []string) string // Generates a cache key

	// Cache metadata
	CacheSize(language string) (int64, error)         // Returns the size of a language's cache
	CacheLastUsed(language string) (time.Time, error) // Returns when the cache was last used

	// Cache configuration
	SetMaxSize(language string, maxSize int64) error // Sets maximum cache size for a language
	SetTTL(language string, ttl time.Duration) error // Sets time-to-live for cache entries
}

// BuildOrchestrator manages the execution of multiple build steps with
// dependency resolution, parallel execution, and error handling.
type BuildOrchestrator interface {
	// Build step management
	AddStep(step BuildStep) error      // Adds a build step to the orchestrator
	AddSteps(steps ...BuildStep) error // Adds multiple build steps
	RemoveStep(name string) error      // Removes a build step by name

	// Execution
	Execute(ctx context.Context) error                       // Executes all build steps
	ExecuteStep(ctx context.Context, name string) error      // Executes a specific step
	ExecuteSteps(ctx context.Context, names ...string) error // Executes specific steps

	// Dependency management
	ResolveDependencies() ([][]string, error) // Returns build steps organized by execution order
	ValidateDependencies() error              // Validates that all dependencies can be resolved

	// Status and monitoring
	GetStepStatus(name string) BuildStepStatus      // Returns the status of a specific step
	GetAllStepStatuses() map[string]BuildStepStatus // Returns status of all steps

	// Configuration
	SetMaxParallel(max int)                 // Sets maximum number of parallel steps
	SetGlobalTimeout(timeout time.Duration) // Sets overall execution timeout
}

// BuildStepStatus represents the current status of a build step
type BuildStepStatus int

const (
	StatusPending BuildStepStatus = iota
	StatusRunning
	StatusCompleted
	StatusFailed
	StatusSkipped
)

func (s BuildStepStatus) String() string {
	switch s {
	case StatusPending:
		return "pending"
	case StatusRunning:
		return "running"
	case StatusCompleted:
		return "completed"
	case StatusFailed:
		return "failed"
	case StatusSkipped:
		return "skipped"
	default:
		return "unknown"
	}
}

// ErrorHandler provides standardized error handling across the build system.
// This interface replaces the scattered os.Exit(1) patterns with proper
// error handling and recovery mechanisms.
type ErrorHandler interface {
	// Error handling
	HandleError(err error, context string) error // Handles and potentially recovers from errors
	WrapError(err error, operation string) error // Wraps errors with additional context

	// Error classification
	IsRetryable(err error) bool               // Determines if an error is retryable
	IsTemporary(err error) bool               // Determines if an error is temporary
	GetErrorSeverity(err error) ErrorSeverity // Returns the severity of an error

	// Recovery
	SuggestRecovery(err error) []string // Suggests recovery actions for an error
}

// ErrorSeverity represents the severity level of an error
type ErrorSeverity int

const (
	SeverityInfo ErrorSeverity = iota
	SeverityWarning
	SeverityError
	SeverityFatal
)

func (s ErrorSeverity) String() string {
	switch s {
	case SeverityInfo:
		return "info"
	case SeverityWarning:
		return "warning"
	case SeverityError:
		return "error"
	case SeverityFatal:
		return "fatal"
	default:
		return "unknown"
	}
}

// ValidationResult represents the result of a validation operation
type ValidationResult struct {
	Context  map[string]interface{}
	Errors   []error
	Warnings []string
	IsValid  bool
}

// Validator provides validation capabilities for different components
type Validator interface {
	Validate(ctx context.Context, target interface{}) (*ValidationResult, error)
	ValidateConfiguration(config interface{}) (*ValidationResult, error)
	ValidateBuildStep(step BuildStep) (*ValidationResult, error)
	ValidateLanguageBuilder(builder LanguageBuilder) (*ValidationResult, error)
}
