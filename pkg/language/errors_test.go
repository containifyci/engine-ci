package language

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidationError(t *testing.T) {
	t.Run("basic error creation", func(t *testing.T) {
		err := NewValidationError("test_field", "invalid_value", "field is invalid")
		
		assert.Equal(t, "test_field", err.Field)
		assert.Equal(t, "invalid_value", err.Value)
		assert.Equal(t, "field is invalid", err.Message)
		
		expectedMessage := "validation error for field 'test_field': field is invalid (got: invalid_value)"
		assert.Equal(t, expectedMessage, err.Error())
	})
	
	t.Run("nil value handling", func(t *testing.T) {
		err := NewValidationError("nil_field", nil, "field cannot be nil")
		
		assert.Equal(t, "nil_field", err.Field)
		assert.Nil(t, err.Value)
		assert.Equal(t, "field cannot be nil", err.Message)
		
		assert.Contains(t, err.Error(), "nil_field")
		assert.Contains(t, err.Error(), "field cannot be nil")
		assert.Contains(t, err.Error(), "<nil>")
	})
}

func TestBuildError(t *testing.T) {
	originalErr := errors.New("original error")
	
	t.Run("basic build error", func(t *testing.T) {
		err := NewBuildError("test_operation", "python", originalErr)
		
		assert.Equal(t, "test_operation", err.Operation)
		assert.Equal(t, "python", err.Language)
		assert.Equal(t, originalErr, err.Cause)
		
		expectedMessage := "build failed [python:test_operation]: original error"
		assert.Equal(t, expectedMessage, err.Error())
	})
	
	t.Run("build error without language", func(t *testing.T) {
		err := NewBuildError("generic_operation", "", originalErr)
		
		assert.Equal(t, "generic_operation", err.Operation)
		assert.Equal(t, "", err.Language)
		assert.Equal(t, originalErr, err.Cause)
		
		expectedMessage := "build failed [generic_operation]: original error"
		assert.Equal(t, expectedMessage, err.Error())
	})
	
	t.Run("error unwrapping", func(t *testing.T) {
		err := NewBuildError("unwrap_test", "golang", originalErr)
		
		// Should be able to unwrap to original error
		unwrapped := errors.Unwrap(err)
		assert.Equal(t, originalErr, unwrapped)
		
		// Should work with errors.Is
		assert.True(t, errors.Is(err, originalErr))
	})
	
	t.Run("context addition", func(t *testing.T) {
		err := NewBuildError("context_test", "python", originalErr)
		err = err.WithContext("dockerfile_path", "/path/to/Dockerfile")
		err = err.WithContext("build_stage", "intermediate")
		
		assert.Equal(t, "/path/to/Dockerfile", err.Context["dockerfile_path"])
		assert.Equal(t, "intermediate", err.Context["build_stage"])
		assert.Len(t, err.Context, 2)
	})
}

func TestContainerError(t *testing.T) {
	originalErr := errors.New("container operation failed")
	
	t.Run("basic container error", func(t *testing.T) {
		err := NewContainerError("start", originalErr)
		
		assert.Equal(t, "start", err.Operation)
		assert.Equal(t, originalErr, err.Cause)
		
		expectedMessage := "container operation failed [start]: container operation failed"
		assert.Equal(t, expectedMessage, err.Error())
	})
	
	t.Run("container error with container ID", func(t *testing.T) {
		err := NewContainerError("stop", originalErr)
		err = err.WithContainer("container_123")
		
		assert.Equal(t, "container_123", err.ContainerID)
		
		expectedMessage := "container operation failed [stop] on container container_123: container operation failed"
		assert.Equal(t, expectedMessage, err.Error())
	})
	
	t.Run("container error with image name", func(t *testing.T) {
		err := NewContainerError("build", originalErr)
		err = err.WithImage("python:3.11-slim")
		
		assert.Equal(t, "python:3.11-slim", err.ImageName)
		
		expectedMessage := "container operation failed [build] on image python:3.11-slim: container operation failed"
		assert.Equal(t, expectedMessage, err.Error())
	})
	
	t.Run("error unwrapping", func(t *testing.T) {
		err := NewContainerError("exec", originalErr)
		
		unwrapped := errors.Unwrap(err)
		assert.Equal(t, originalErr, unwrapped)
		assert.True(t, errors.Is(err, originalErr))
	})
}

func TestCacheError(t *testing.T) {
	originalErr := errors.New("cache operation failed")
	
	t.Run("basic cache error", func(t *testing.T) {
		err := NewCacheError("clean", "python", originalErr)
		
		assert.Equal(t, "clean", err.Operation)
		assert.Equal(t, "python", err.Language)
		assert.Equal(t, originalErr, err.Cause)
		
		expectedMessage := "cache operation failed [clean] for python cache: cache operation failed"
		assert.Equal(t, expectedMessage, err.Error())
	})
	
	t.Run("cache error without language", func(t *testing.T) {
		err := NewCacheError("init", "", originalErr)
		
		expectedMessage := "cache operation failed [init]: cache operation failed"
		assert.Equal(t, expectedMessage, err.Error())
	})
	
	t.Run("cache error with path", func(t *testing.T) {
		err := NewCacheError("create", "golang", originalErr)
		err = err.WithPath("/go/pkg/mod")
		
		assert.Equal(t, "/go/pkg/mod", err.CachePath)
	})
	
	t.Run("error unwrapping", func(t *testing.T) {
		err := NewCacheError("validate", "maven", originalErr)
		
		unwrapped := errors.Unwrap(err)
		assert.Equal(t, originalErr, unwrapped)
		assert.True(t, errors.Is(err, originalErr))
	})
}

func TestErrorIntegration(t *testing.T) {
	// Test that errors work well together and maintain Go error conventions
	
	t.Run("error type checking", func(t *testing.T) {
		originalErr := errors.New("base error")
		
		validationErr := NewValidationError("field", "value", "message")
		buildErr := NewBuildError("build", "python", originalErr)
		containerErr := NewContainerError("start", originalErr)
		cacheErr := NewCacheError("clean", "golang", originalErr)
		
		// Should all implement error interface
		var err1 error = validationErr
		var err2 error = buildErr
		var err3 error = containerErr
		var err4 error = cacheErr
		
		assert.NotNil(t, err1.Error())
		assert.NotNil(t, err2.Error())
		assert.NotNil(t, err3.Error())
		assert.NotNil(t, err4.Error())
		
		// Wrapped errors should unwrap correctly
		assert.True(t, errors.Is(buildErr, originalErr))
		assert.True(t, errors.Is(containerErr, originalErr))
		assert.True(t, errors.Is(cacheErr, originalErr))
	})
	
	t.Run("error type assertion", func(t *testing.T) {
		validationErr := NewValidationError("test", "value", "message")
		buildErr := NewBuildError("build", "python", errors.New("build failed"))
		
		// Should be able to assert to specific error types
		var err error = validationErr
		var validationErrPtr *ValidationError
		assert.True(t, errors.As(err, &validationErrPtr))
		assert.Equal(t, "test", validationErrPtr.Field)
		
		err = buildErr
		var buildErrPtr *BuildError
		assert.True(t, errors.As(err, &buildErrPtr))
		assert.Equal(t, "build", buildErrPtr.Operation)
	})
}

func TestErrorBackwardCompatibility(t *testing.T) {
	// Test that error types provide better alternatives to os.Exit(1) patterns
	
	t.Run("replacement for os.Exit patterns", func(t *testing.T) {
		// Before: Code would call os.Exit(1) on dockerfile read failure
		// After: Code should return BuildError
		
		dockerfileErr := errors.New("dockerfile not found")
		buildErr := NewBuildError("read_dockerfile", "python", dockerfileErr)
		
		// Error should contain sufficient context for debugging
		assert.Contains(t, buildErr.Error(), "python")
		assert.Contains(t, buildErr.Error(), "read_dockerfile")
		assert.Contains(t, buildErr.Error(), "dockerfile not found")
		
		// Should be recoverable by caller
		assert.True(t, errors.Is(buildErr, dockerfileErr))
	})
	
	t.Run("structured error handling", func(t *testing.T) {
		// Errors should provide structured information for automated handling
		
		validationErr := NewValidationError("base_image", "", "base image cannot be empty")
		assert.Equal(t, "base_image", validationErr.Field)
		assert.Equal(t, "", validationErr.Value)
		
		containerErr := NewContainerError("create", errors.New("image not found"))
		containerErr = containerErr.WithImage("nonexistent:latest")
		assert.Equal(t, "create", containerErr.Operation)
		assert.Equal(t, "nonexistent:latest", containerErr.ImageName)
		
		// This enables callers to handle errors programmatically
		// instead of just logging and exiting
	})
}