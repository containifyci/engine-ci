// Package language provides common interfaces and utilities for language-specific container builds.
// This package defines the LanguageStrategy interface that abstracts language-specific behaviors
// used by the ContainerBuildOrchestrator to eliminate code duplication across different language implementations.
package language

import (
	"context"

	"github.com/containifyci/engine-ci/pkg/cri/types"
)

// LanguageStrategy defines the contract for language-specific behavior that enables
// the ContainerBuildOrchestrator to handle different programming languages uniformly.
// This interface eliminates code duplication by abstracting the language-specific
// methods that were previously scattered across individual language implementations.
//
// The strategy pattern allows the orchestrator to work with any language implementation
// without knowing the specific details of how each language handles builds, images,
// or scripts. This promotes maintainability and extensibility when adding new language support.
//
// Key benefits:
//   - Eliminates duplicated orchestration logic across language implementations
//   - Provides a uniform interface for container build operations
//   - Enables easy addition of new programming languages
//   - Centralizes build workflow logic in the orchestrator
//   - Maintains language-specific customization through strategy implementations
//
// Example usage:
//
//	// Go language strategy implementation
//	goStrategy := &GoLanguageStrategy{
//	    container: goContainer,
//	    version:   "1.24.2",
//	}
//
//	// Python language strategy implementation
//	pythonStrategy := &PythonLanguageStrategy{
//	    container: pythonContainer,
//	    version:   "3.11",
//	}
//
//	// Orchestrator can work with any language uniformly
//	orchestrator := NewContainerBuildOrchestrator(goStrategy)
//	err := orchestrator.ExecuteBuild(ctx)
//
//	// Same orchestrator logic works for Python
//	orchestrator = NewContainerBuildOrchestrator(pythonStrategy)
//	err = orchestrator.ExecuteBuild(ctx)
type LanguageStrategy interface {
	// GetIntermediateImage returns the language-specific intermediate container image
	// that contains the build tools and dependencies needed for compilation.
	//
	// This method abstracts the language-specific image creation logic:
	//   - Go: Returns result of GoImage() method with golang base image
	//   - Python: Returns result of PythonImage() method with python base image
	//   - Java: Would return JavaImage() with JDK base image
	//
	// The orchestrator uses this image for the build container where compilation occurs.
	// The image typically includes language runtime, build tools, and cached dependencies.
	//
	// Returns:
	//   - string: Fully qualified image URI (e.g., "registry.io/golang-1.24.2-alpine:abc123")
	//   - error: If image cannot be determined or built
	//
	// Example implementations:
	//   - Go Alpine: "containify.io/golang-1.24.2-alpine:sha256hash"
	//   - Python Slim: "containify.io/python-3.11-slim-bookworm:sha256hash"
	GetIntermediateImage(ctx context.Context) (string, error)

	// GenerateBuildScript returns the language-specific build script that will be
	// executed inside the build container to compile/build the application.
	//
	// This method eliminates duplication of build script generation logic:
	//   - Go: Returns BuildScript() with go build, test, and platform-specific commands
	//   - Python: Returns BuildScript() with pip install, pytest, and package commands
	//   - Java: Would return Maven/Gradle build commands
	//
	// The generated script should handle:
	//   - Dependency installation/resolution
	//   - Compilation or build process
	//   - Testing execution
	//   - Artifact generation
	//   - Platform-specific optimizations
	//
	// Returns:
	//   - string: Shell script content to execute in the build container
	//
	// Example Go script:
	//   #!/bin/bash
	//   set -e
	//   go mod download
	//   go test ./...
	//   CGO_ENABLED=0 GOOS=linux go build -o /out/app ./main.go
	//
	// Example Python script:
	//   #!/bin/bash
	//   set -e
	//   pip install -r requirements.txt
	//   python -m pytest
	//   python -m build --wheel --outdir /out/
	GenerateBuildScript() string

	// GetAdditionalImages returns a list of additional container images that need
	// to be pulled before the build process can start.
	//
	// This method abstracts language-specific image dependencies:
	//   - Go: Returns ["alpine:latest"] for final lightweight runtime image
	//   - Python: Returns ["python:3.11-slim"] for runtime image
	//   - Node.js: Would return ["node:18-alpine"] for runtime
	//
	// The orchestrator uses this list to:
	//   - Pre-pull all required images in parallel for better performance
	//   - Ensure all dependencies are available before starting build
	//   - Enable offline/air-gapped builds by pre-staging images
	//
	// Returns:
	//   - []string: List of fully qualified image names to pull
	//
	// Example return values:
	//   - Go: ["alpine:latest", "scratch"]
	//   - Python: ["python:3.11-slim-bookworm"]
	//   - Java: ["openjdk:11-jre-slim"]
	GetAdditionalImages() []string

	// ShouldCommitResult determines whether the orchestrator should commit the
	// final build result to create a new container image.
	//
	// This method allows language-specific control over result handling:
	//   - Go: Returns true to create optimized final image with compiled binary
	//   - Python: Returns true to create image with installed packages and code
	//   - Script languages: Might return false if only artifacts are needed
	//
	// When true, the orchestrator will:
	//   - Commit the final container state after build completion
	//   - Tag the resulting image appropriately
	//   - Make the image available for deployment or further processing
	//
	// When false, the orchestrator will:
	//   - Extract build artifacts from the container
	//   - Clean up the build container without committing
	//   - Provide artifacts through alternative means (volumes, copying)
	//
	// Returns:
	//   - bool: true if result should be committed as container image
	ShouldCommitResult() bool

	// GetCommitCommand returns the container commit command to use when
	// ShouldCommitResult() returns true.
	//
	// This method provides language-specific commit behavior:
	//   - Go: Returns optimized commit command with minimal layers
	//   - Python: Returns commit with proper Python entrypoint and metadata
	//   - Web apps: Might return commit with web server configuration
	//
	// The command should specify:
	//   - Appropriate entrypoint for the language/application type
	//   - Required environment variables
	//   - Working directory
	//   - Exposed ports (if applicable)
	//   - Metadata labels for traceability
	//
	// Returns:
	//   - string: Container commit command with appropriate configuration
	//
	// Example Go commit command:
	//   --change 'ENTRYPOINT ["/app"]' --change 'WORKDIR /app' --change 'EXPOSE 8080'
	//
	// Example Python commit command:
	//   --change 'ENTRYPOINT ["python", "/app/main.py"]' --change 'WORKDIR /app'
	//
	// If ShouldCommitResult() returns false, this method may return empty string
	// or may not be called by the orchestrator.
	GetCommitCommand() string

	// GetIntermediateImageDockerfile returns the dockerfile content used to build
	// the intermediate image for this language strategy.
	//
	// This method enables the orchestrator to build the intermediate image when needed.
	// The dockerfile should contain all build tools, dependencies, and language-specific
	// setup required for the compilation process.
	//
	// Returns:
	//   - []byte: Raw dockerfile content
	//   - error: If dockerfile cannot be read or accessed
	GetIntermediateImageDockerfile(ctx context.Context) ([]byte, error)

	// GetIntermediateImagePlatforms returns the platforms for which the intermediate
	// image should be built.
	//
	// This method provides platform information for multi-platform builds:
	//   - Single platform: []*types.PlatformSpec{types.ParsePlatform("linux/amd64")}
	//   - Multi-platform: []*types.PlatformSpec{types.ParsePlatform("linux/amd64"), types.ParsePlatform("linux/arm64")}
	//
	// Returns:
	//   - []*types.PlatformSpec: List of target platforms for the intermediate image
	GetIntermediateImagePlatforms() []*types.PlatformSpec
}

// LanguageStrategyConfig provides configuration options for language strategy implementations.
// This struct can be embedded or used as a parameter to provide common configuration
// values that strategies might need.
type LanguageStrategyConfig struct {
	Environment  map[string]string
	Registry     string
	BuildTimeout string
	Platform     string
	Tags         []string
	Verbose      bool
}

// Examples of how this interface eliminates code duplication:
//
// BEFORE (duplicated across each language):
//   func (orchestrator *ContainerBuildOrchestrator) ExecuteGoBuild(ctx context.Context, goContainer *GoContainer) error {
//       // Pull base images
//       images := goContainer.Images() // Go-specific method
//       orchestrator.pullImages(images)
//
//       // Get intermediate image
//       intermediateImg, err := goContainer.GoImage() // Go-specific method
//       if err != nil { return err }
//
//       // Generate build script
//       script := goContainer.BuildScript() // Go-specific method
//
//       // Execute build
//       return orchestrator.executeBuild(intermediateImg, script)
//   }
//
//   func (orchestrator *ContainerBuildOrchestrator) ExecutePythonBuild(ctx context.Context, pythonContainer *PythonContainer) error {
//       // Pull base images - DUPLICATED LOGIC
//       images := pythonContainer.Images() // Python-specific method
//       orchestrator.pullImages(images)
//
//       // Get intermediate image - DUPLICATED LOGIC
//       intermediateImg, err := pythonContainer.PythonImage() // Python-specific method
//       if err != nil { return err }
//
//       // Generate build script - DUPLICATED LOGIC
//       script := pythonContainer.BuildScript() // Python-specific method
//
//       // Execute build - DUPLICATED LOGIC
//       return orchestrator.executeBuild(intermediateImg, script)
//   }
//
// AFTER (unified with strategy pattern):
//   func (orchestrator *ContainerBuildOrchestrator) ExecuteBuild(ctx context.Context, strategy LanguageStrategy) error {
//       // Pull base images - UNIFIED LOGIC
//       images := strategy.GetAdditionalImages()
//       orchestrator.pullImages(images)
//
//       // Get intermediate image - UNIFIED LOGIC
//       intermediateImg, err := strategy.GetIntermediateImage(ctx)
//       if err != nil { return err }
//
//       // Generate build script - UNIFIED LOGIC
//       script := strategy.GenerateBuildScript()
//
//       // Execute build - UNIFIED LOGIC
//       err = orchestrator.executeBuild(intermediateImg, script)
//       if err != nil { return err }
//
//       // Handle result - UNIFIED LOGIC
//       if strategy.ShouldCommitResult() {
//           commitCmd := strategy.GetCommitCommand()
//           return orchestrator.commitResult(commitCmd)
//       }
//
//       return nil
//   }
//
// This reduces code duplication from O(n) language implementations to O(1) orchestrator
// plus O(n) simple strategy implementations, significantly improving maintainability.
