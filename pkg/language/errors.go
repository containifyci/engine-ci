// This file provides standardized error types for language builders.
//
// These error types replace the antipattern of os.Exit(1) calls scattered
// throughout the codebase. They provide:
//
//   - Structured error information with context
//   - Error wrapping for better debugging  
//   - Consistent error messages across all language packages
//   - Support for error recovery and handling strategies
//
// The error types follow Go's error handling best practices and integrate
// with the standard errors package for error unwrapping and type checking.
package language

import "fmt"

// ValidationError represents a validation error in language builders
type ValidationError struct {
	Field   string      // The field that failed validation
	Value   interface{} // The value that failed validation
	Message string      // Human-readable error message
}

// Error implements the error interface
func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation error for field '%s': %s (got: %v)", e.Field, e.Message, e.Value)
}

// NewValidationError creates a new validation error
func NewValidationError(field string, value interface{}, message string) *ValidationError {
	return &ValidationError{
		Field:   field,
		Value:   value,
		Message: message,
	}
}

// BuildError represents a build-related error
type BuildError struct {
	Cause     error
	Context   map[string]interface{}
	Operation string
	Language  string
}

// Error implements the error interface
func (e *BuildError) Error() string {
	if e.Language != "" {
		return fmt.Sprintf("build failed [%s:%s]: %v", e.Language, e.Operation, e.Cause)
	}
	return fmt.Sprintf("build failed [%s]: %v", e.Operation, e.Cause)
}

// Unwrap returns the underlying error for error unwrapping
func (e *BuildError) Unwrap() error {
	return e.Cause
}

// NewBuildError creates a new build error
func NewBuildError(operation, language string, cause error) *BuildError {
	return &BuildError{
		Operation: operation,
		Language:  language,
		Cause:     cause,
		Context:   make(map[string]interface{}),
	}
}

// WithContext adds context information to the build error
func (e *BuildError) WithContext(key string, value interface{}) *BuildError {
	if e.Context == nil {
		e.Context = make(map[string]interface{})
	}
	e.Context[key] = value
	return e
}

// ContainerError represents a container-related error
type ContainerError struct {
	Cause       error
	Operation   string
	ContainerID string
	ImageName   string
}

// Error implements the error interface
func (e *ContainerError) Error() string {
	if e.ContainerID != "" {
		return fmt.Sprintf("container operation failed [%s] on container %s: %v", e.Operation, e.ContainerID, e.Cause)
	}
	if e.ImageName != "" {
		return fmt.Sprintf("container operation failed [%s] on image %s: %v", e.Operation, e.ImageName, e.Cause)
	}
	return fmt.Sprintf("container operation failed [%s]: %v", e.Operation, e.Cause)
}

// Unwrap returns the underlying error for error unwrapping
func (e *ContainerError) Unwrap() error {
	return e.Cause
}

// NewContainerError creates a new container error
func NewContainerError(operation string, cause error) *ContainerError {
	return &ContainerError{
		Operation: operation,
		Cause:     cause,
	}
}

// WithContainer adds container ID to the error
func (e *ContainerError) WithContainer(containerID string) *ContainerError {
	e.ContainerID = containerID
	return e
}

// WithImage adds image name to the error
func (e *ContainerError) WithImage(imageName string) *ContainerError {
	e.ImageName = imageName
	return e
}

// CacheError represents a cache-related error
type CacheError struct {
	Cause     error
	Operation string
	Language  string
	CachePath string
}

// Error implements the error interface
func (e *CacheError) Error() string {
	if e.Language != "" {
		return fmt.Sprintf("cache operation failed [%s] for %s cache: %v", e.Operation, e.Language, e.Cause)
	}
	return fmt.Sprintf("cache operation failed [%s]: %v", e.Operation, e.Cause)
}

// Unwrap returns the underlying error for error unwrapping
func (e *CacheError) Unwrap() error {
	return e.Cause
}

// NewCacheError creates a new cache error
func NewCacheError(operation, language string, cause error) *CacheError {
	return &CacheError{
		Operation: operation,
		Language:  language,
		Cause:     cause,
	}
}

// WithPath adds cache path to the error
func (e *CacheError) WithPath(cachePath string) *CacheError {
	e.CachePath = cachePath
	return e
}
