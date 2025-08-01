// Package language provides container build orchestration that eliminates
// code duplication across all language implementations.
//
// The ContainerBuildOrchestrator centralizes the ~90% of container orchestration
// logic that was previously duplicated across golang/alpine, golang/debian,
// golang/debiancgo, python, maven, and other language packages.
//
// Key benefits:
//   - Eliminates duplicated SSH forwarding setup across all language packages  
//   - Centralizes container configuration and environment variable handling
//   - Provides uniform volume mounting for source code and caches
//   - Standardizes image pulling and management workflows
//   - Reduces maintenance burden by consolidating container orchestration logic
//   - Enables consistent error handling and logging across all languages
//
// The orchestrator uses the Strategy pattern to delegate only the truly
// language-specific behavior (intermediate image creation, build scripts,
// commit decisions) while handling all the shared container orchestration.
//
// Example usage:
//
//	// Create a language strategy (e.g., for Go)
//	strategy := &GoLanguageStrategy{
//	    container: goContainer,
//	    baseBuilder: baseLanguageBuilder,
//	}
//
//	// Create orchestrator with the strategy
//	orchestrator := NewContainerBuildOrchestrator(strategy, baseLanguageBuilder)
//
//	// Execute unified build workflow
//	imageID, err := orchestrator.Build(ctx)
//	if err != nil {
//	    return fmt.Errorf("build failed: %w", err)
//	}
//
//	// Pull all required images
//	if err := orchestrator.Pull(); err != nil {
//	    return fmt.Errorf("pull failed: %w", err)
//	}
//
//	// Get all images needed
//	images := orchestrator.Images()
//
// This design eliminates the need for each language package to implement
// the same container orchestration logic repeatedly, reducing the codebase
// from O(n) duplicated implementations to O(1) orchestrator + O(n) simple
// strategy implementations.
package language

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/containifyci/engine-ci/pkg/cri/types"
	"github.com/containifyci/engine-ci/pkg/network"
)

// ContainerBuildOrchestrator eliminates ~90% of duplicated container orchestration 
// code across all language packages by centralizing the shared build workflow logic.
//
// This orchestrator handles all the common container operations that were previously
// duplicated across golang/alpine, golang/debian, golang/debiancgo, python, maven,
// and other language implementations:
//
//   - SSH forwarding setup (identical across all languages)
//   - Container configuration and environment variables (identical pattern)
//   - Working directory and volume mount setup (identical logic)
//   - Source code and cache volume mounting (identical implementation)
//   - BuildingContainer execution (identical call pattern)
//   - Image pulling workflows (100% identical)
//   - Images list generation (90% identical pattern)
//
// The orchestrator uses composition with BaseLanguageBuilder for shared functionality
// and delegation to LanguageStrategy for language-specific behavior. This eliminates
// code duplication while maintaining the flexibility for language-specific customization.
//
// Performance benefits:
//   - Reduces duplicated code by ~90% across language packages
//   - Centralizes container orchestration logic for easier maintenance
//   - Provides consistent error handling and logging
//   - Enables uniform caching and volume management
//   - Standardizes SSH forwarding and security setup
type ContainerBuildOrchestrator struct {
	// strategy handles language-specific behavior (intermediate images, build scripts, commit decisions)
	strategy LanguageStrategy

	// baseBuilder provides shared functionality (configuration, container access, logging, validation)
	baseBuilder *BaseLanguageBuilder

	// logger provides structured logging with orchestrator context
	logger *slog.Logger
}

// NewContainerBuildOrchestrator creates a new container build orchestrator that eliminates
// duplicated container orchestration code across language implementations.
//
// The orchestrator centralizes the shared build workflow logic while delegating
// language-specific behavior to the provided strategy. This design reduces code
// duplication from O(n) language implementations to O(1) orchestrator + O(n) strategies.
//
// Parameters:
//   - strategy: Language-specific strategy implementing LanguageStrategy interface
//   - baseBuilder: Shared base functionality from BaseLanguageBuilder
//
// Returns:
//   - *ContainerBuildOrchestrator: Configured orchestrator ready for build operations
//
// Example usage:
//
//	strategy := &GoLanguageStrategy{container: goContainer}
//	orchestrator := NewContainerBuildOrchestrator(strategy, baseBuilder)
//	
//	// Orchestrator now handles all shared container logic
//	imageID, err := orchestrator.Build(ctx)
func NewContainerBuildOrchestrator(strategy LanguageStrategy, baseBuilder *BaseLanguageBuilder) *ContainerBuildOrchestrator {
	return &ContainerBuildOrchestrator{
		strategy:    strategy,
		baseBuilder: baseBuilder,
		logger:      slog.With("component", "container-build-orchestrator", "language", baseBuilder.Name()),
	}
}

// Build executes the complete container build workflow, centralizing the ~90% of
// duplicated orchestration code that was scattered across language packages.
//
// This method eliminates the following duplicated patterns:
//   1. SSH forwarding setup (identical across golang/debian, golang/alpine, python)
//   2. Container configuration setup (identical pattern in all language packages)
//   3. Environment variables from config (identical logic everywhere)
//   4. Working directory setup (identical implementation)
//   5. Source volume mount (identical across all languages)
//   6. Cache volume mount (identical logic)
//   7. SSH application to config (identical call)
//   8. Build script setting (identical pattern)
//   9. BuildingContainer call (identical across all packages)
//   10. Optional commit step (varies by language via strategy)
//
// The method delegates only the truly language-specific parts to the strategy:
//   - GetIntermediateImage(): Language-specific image creation (GoImage(), PythonImage(), etc.)
//   - GenerateBuildScript(): Language-specific build commands
//   - ShouldCommitResult(): Language-specific commit decision
//   - GetCommitCommand(): Language-specific commit configuration
//
// Parameters:
//   - ctx: Context for cancellation and timeouts
//
// Returns:
//   - string: Container/image ID of the build result
//   - error: Build error with proper context and error wrapping
//
// Error handling:
//   - SSH forwarding errors → BuildError with "ssh_forward" operation
//   - Container configuration errors → BuildError with appropriate operation context
//   - Build execution errors → BuildError with "building_container" operation
//   - Commit errors → BuildError with "commit_container" operation
func (o *ContainerBuildOrchestrator) Build(ctx context.Context) (string, error) {
	o.logger.Info("Starting container build orchestration")

	// Step 1: Get language-specific intermediate image (replaces GoImage(), PythonImage(), etc.)
	imageTag, err := o.strategy.GetIntermediateImage(ctx)
	if err != nil {
		return "", NewBuildError("get_intermediate_image", o.baseBuilder.Name(), err)
	}
	o.logger.Debug("Using intermediate image", "image", imageTag)

	// Step 1.5: Build intermediate image if needed (equivalent to BuildGoImage(), BuildPythonImage(), etc.)
	err = o.buildIntermediateImageIfNeeded(ctx, imageTag)
	if err != nil {
		return "", NewBuildError("build_intermediate_image", o.baseBuilder.Name(), err)
	}

	// Step 2: Setup SSH forwarding (identical logic previously duplicated across all packages)
	ssh, err := network.SSHForward(*o.baseBuilder.GetContainer().GetBuild())
	if err != nil {
		return "", NewBuildError("ssh_forward", o.baseBuilder.Name(), err)
	}
	o.logger.Debug("SSH forwarding configured")

	// Step 3: Initialize container configuration (eliminates duplicated config setup)
	opts := types.ContainerConfig{}
	opts.Image = imageTag

	// Step 4: Setup environment variables from configuration (eliminates duplicated env setup)
	cfg := o.baseBuilder.GetConfig()
	for key, value := range cfg.Environment {
		opts.Env = append(opts.Env, fmt.Sprintf("%s=%s", key, value))
	}
	o.logger.Debug("Environment variables configured", "count", len(cfg.Environment))

	// Step 5: Set working directory (eliminates duplicated working directory logic)
	opts.WorkingDir = cfg.WorkingDir

	// Step 6: Setup source volume mount (eliminates duplicated volume mounting logic)
	dir, err := o.getSourceDirectory()
	if err != nil {
		return "", NewBuildError("get_source_directory", o.baseBuilder.Name(), err)
	}

	// Step 7: Setup cache volume mount (eliminates duplicated cache mounting logic)
	cacheDir, err := o.getCacheDirectory()
	if err != nil {
		return "", NewBuildError("get_cache_directory", o.baseBuilder.Name(), err)
	}

	// Configure volumes (eliminates duplicated volume configuration)
	opts.Volumes = []types.Volume{
		{
			Type:   "bind",
			Source: dir,
			Target: cfg.WorkingDir,
		},
		{
			Type:   "bind",
			Source: cacheDir,
			Target: cfg.CacheLocation,
		},
	}
	o.logger.Debug("Volume mounts configured", "source", dir, "cache", cacheDir)

	// Step 8: Apply SSH configuration (eliminates duplicated SSH application)
	opts = ssh.Apply(&opts)

	// Step 9: Set build script (delegates to language-specific implementation)
	opts.Script = o.strategy.GenerateBuildScript()
	o.logger.Debug("Build script configured")

	// Step 10: Execute container build (eliminates duplicated BuildingContainer calls)
	err = o.baseBuilder.GetContainer().BuildingContainer(opts)
	if err != nil {
		return "", NewBuildError("building_container", o.baseBuilder.Name(), err)
	}
	o.logger.Info("Container build completed successfully")

	// Step 11: Handle commit decision (language-specific via strategy)
	if o.strategy.ShouldCommitResult() {
		return o.commitResult()
	}

	// Return container ID for non-committed builds
	containerID := o.baseBuilder.GetContainer().ID
	o.logger.Info("Build completed without commit", "containerId", containerID)
	return containerID, nil
}

// Pull executes the image pulling workflow, eliminating 100% duplicated pull logic
// that was scattered across all language packages.
//
// This method centralizes the identical pull patterns from:
//   - golang/debian/golang.go: Pull() method (lines 97-110)
//   - golang/alpine/golang.go: Pull() method  
//   - golang/debiancgo/golang.go: Pull() method
//   - python/python.go: Pull() method (line 76-78)
//   - All other language packages with identical logic
//
// The method:
//   1. Pulls the base image (identical across all languages)
//   2. Pulls additional images via strategy (replaces hardcoded alpine:latest, etc.)
//   3. Provides consistent error handling and logging
//
// Returns:
//   - error: Pull error with proper context (ContainerError with image information)
func (o *ContainerBuildOrchestrator) Pull() error {
	o.logger.Info("Starting image pull orchestration")

	// Step 1: Pull base image (eliminates duplicated base image pull logic)
	baseImage := o.baseBuilder.BaseImage()
	o.logger.Debug("Pulling base image", "image", baseImage)
	
	if err := o.baseBuilder.GetContainer().Pull(baseImage); err != nil {
		return NewContainerError("pull_base_image", err).WithImage(baseImage)
	}

	// Step 2: Pull additional images via strategy (eliminates hardcoded alpine:latest, etc.)
	additionalImages := o.strategy.GetAdditionalImages()
	for _, image := range additionalImages {
		o.logger.Debug("Pulling additional image", "image", image)
		
		if err := o.baseBuilder.GetContainer().Pull(image); err != nil {
			return NewContainerError("pull_additional_image", err).WithImage(image)
		}
	}

	o.logger.Info("All images pulled successfully", 
		"base_image", baseImage, 
		"additional_images", len(additionalImages))
	return nil
}

// Images returns all container images required for the build, eliminating 90% 
// duplicated image list logic across all language packages.
//
// This method centralizes the nearly identical Images() patterns from:
//   - golang/debian/golang.go: Images() method (lines 134-142)
//   - golang/alpine/golang.go: Images() method
//   - python/python.go: Images() method (lines 80-87)
//   - All other language packages with similar logic
//
// The method:
//   1. Gets base image (identical across all languages)
//   2. Gets intermediate image via strategy (replaces GoImage(), PythonImage(), etc.)
//   3. Gets additional images via strategy (replaces hardcoded lists)
//   4. Handles errors gracefully with fallback (identical error handling pattern)
//   5. Returns combined list (identical return pattern)
//
// Returns:
//   - []string: Complete list of images required for the build
func (o *ContainerBuildOrchestrator) Images() []string {
	o.logger.Debug("Generating required images list")

	// Step 1: Get base image (eliminates duplicated base image logic)
	baseImage := o.baseBuilder.BaseImage()
	images := []string{baseImage}

	// Step 2: Get intermediate image via strategy (eliminates GoImage(), PythonImage(), etc.)
	intermediateImage, err := o.strategy.GetIntermediateImage(context.Background())
	if err != nil {
		// Handle error gracefully with logging (eliminates duplicated error handling)
		o.logger.Error("Failed to get intermediate image, using base image only", 
			"error", err, "base_image", baseImage)
	} else {
		images = append(images, intermediateImage)
	}

	// Step 3: Get additional images via strategy (eliminates hardcoded alpine:latest, etc.)
	additionalImages := o.strategy.GetAdditionalImages()
	images = append(images, additionalImages...)

	o.logger.Debug("Images list generated", "count", len(images), "images", images)
	return images
}

// getSourceDirectory determines the source directory for volume mounting,
// centralizing the logic that was duplicated across language packages.
//
// This eliminates the duplicated pattern from golang/debian, python, etc.:
//   dir, _ := filepath.Abs(".")
//   if c.Folder != "" {
//       dir, _ = filepath.Abs(c.Folder)  
//   }
func (o *ContainerBuildOrchestrator) getSourceDirectory() (string, error) {
	build := o.baseBuilder.GetContainer().GetBuild()
	
	// Use Folder if specified, otherwise current directory (eliminates duplicated logic)
	if build.Folder != "" {
		dir, err := filepath.Abs(build.Folder)
		if err != nil {
			return "", fmt.Errorf("failed to resolve folder path %s: %w", build.Folder, err)
		}
		return dir, nil
	}

	// Default to current directory
	dir, err := filepath.Abs(".")
	if err != nil {
		return "", fmt.Errorf("failed to resolve current directory: %w", err)
	}
	return dir, nil
}

// getCacheDirectory determines the cache directory for volume mounting,
// centralizing cache management logic across language packages.
//
// This eliminates the need for each language to implement its own cache
// directory resolution logic (CacheFolder() functions in golang, python, etc.)
func (o *ContainerBuildOrchestrator) getCacheDirectory() (string, error) {
	// Try to use cache manager if available
	if cacheManager := o.baseBuilder.GetCacheManager(); cacheManager != nil {
		cacheDir, err := cacheManager.GetCacheDir(o.baseBuilder.Name())
		if err == nil {
			return cacheDir, nil
		}
		o.logger.Warn("Cache manager failed, falling back to temp cache", "error", err)
	}

	// Fallback to temporary cache directory (consistent across all languages)
	tempCache := filepath.Join(".tmp", o.baseBuilder.Name())
	if err := os.MkdirAll(tempCache, os.ModePerm); err != nil {
		return "", NewCacheError("create_temp_cache", o.baseBuilder.Name(), err).WithPath(tempCache)
	}

	o.logger.Debug("Using temporary cache directory", "path", tempCache)
	return filepath.Abs(tempCache)
}

// commitResult handles the container commit process for languages that require it.
// This centralizes commit logic that varies slightly between languages.
func (o *ContainerBuildOrchestrator) commitResult() (string, error) {
	build := o.baseBuilder.GetContainer().GetBuild()
	
	// Handle empty image name (skip commit, return container ID)
	// This matches the original behavior: "Skip No image specified to push"
	if build.Image == "" {
		containerID := o.baseBuilder.GetContainer().ID
		o.logger.Info("Skipping commit - no image name specified", "containerId", containerID)
		return containerID, nil
	}
	
	commitCommand := o.strategy.GetCommitCommand()
	o.logger.Debug("Committing container result", "command", commitCommand, 
		"imageTag", fmt.Sprintf("%s:%s", build.Image, build.ImageTag))

	// Use the commit command from strategy plus standard container configuration
	// The original pattern used: CMD ["/app/app"], "USER app", "WORKDIR /app"
	imageID, err := o.baseBuilder.GetContainer().Commit(
		fmt.Sprintf("%s:%s", build.Image, build.ImageTag),
		"Created by ContainerBuildOrchestrator",
		commitCommand,    // Language-specific command (e.g., CMD ["/app/engine-ci"])
		"USER app",       // Standard user
		"WORKDIR /app",   // Standard working directory
	)
	if err != nil {
		return "", NewBuildError("commit_container", o.baseBuilder.Name(), err)
	}

	o.logger.Info("Container committed successfully", "imageId", imageID)
	return imageID, nil
}

// Validate validates the orchestrator configuration and strategy.
// This provides comprehensive validation across the orchestrator and strategy.
func (o *ContainerBuildOrchestrator) Validate() error {
	o.logger.Debug("Validating orchestrator configuration")

	// Validate base builder (eliminates duplicated validation logic)
	if err := o.baseBuilder.Validate(); err != nil {
		return fmt.Errorf("base builder validation failed: %w", err)
	}

	// Validate strategy is provided
	if o.strategy == nil {
		return NewValidationError("strategy", nil, "language strategy is required")
	}

	// Validate that strategy can provide required information
	ctx := context.Background()
	if _, err := o.strategy.GetIntermediateImage(ctx); err != nil {
		return NewValidationError("intermediate_image", nil, 
			fmt.Sprintf("strategy failed to provide intermediate image: %v", err))
	}

	if script := o.strategy.GenerateBuildScript(); script == "" {
		return NewValidationError("build_script", script, 
			"strategy must provide non-empty build script")
	}

	o.logger.Debug("Orchestrator validation completed successfully")
	return nil
}

// GetStrategy returns the language strategy for advanced usage.
// This allows access to language-specific functionality when needed.
func (o *ContainerBuildOrchestrator) GetStrategy() LanguageStrategy {
	return o.strategy
}

// GetBaseBuilder returns the base language builder for shared functionality access.
// This provides access to configuration, container, logging, and other shared services.
func (o *ContainerBuildOrchestrator) GetBaseBuilder() *BaseLanguageBuilder {
	return o.baseBuilder
}

// SetStrategy updates the language strategy. This enables runtime strategy switching
// for advanced use cases or testing scenarios.
func (o *ContainerBuildOrchestrator) SetStrategy(strategy LanguageStrategy) {
	o.logger.Info("Updating language strategy", "previous", fmt.Sprintf("%T", o.strategy), "new", fmt.Sprintf("%T", strategy))
	o.strategy = strategy
}

// GetLogger returns the orchestrator's logger for consistent logging integration.
func (o *ContainerBuildOrchestrator) GetLogger() *slog.Logger {
	return o.logger
}

// buildIntermediateImageIfNeeded builds the intermediate image if it doesn't exist.
// This replaces the BuildGoImage(), BuildPythonImage(), etc. methods from individual packages.
func (o *ContainerBuildOrchestrator) buildIntermediateImageIfNeeded(ctx context.Context, imageTag string) error {
	o.logger.Debug("Building intermediate image if needed", "image", imageTag)

	// Get dockerfile content from the strategy
	dockerFile, err := o.strategy.GetIntermediateImageDockerfile(ctx)
	if err != nil {
		return NewBuildError("get_dockerfile", o.baseBuilder.Name(), err)
	}

	// Get platforms from the strategy
	platformSpecs := o.strategy.GetIntermediateImagePlatforms()
	
	// Convert platform specs to strings
	platforms := make([]string, len(platformSpecs))
	for i, spec := range platformSpecs {
		platforms[i] = spec.String()
	}

	// Build the intermediate image using the container's BuildIntermediateContainer method
	o.logger.Info("Building intermediate image", "image", imageTag, "platforms", platforms)
	return o.baseBuilder.GetContainer().BuildIntermidiateContainer(imageTag, dockerFile, platforms...)
}

// This orchestrator eliminates the following specific code duplication patterns:
//
// BEFORE (duplicated across golang/debian, golang/alpine, python, etc.):
//
// func (c *GoContainer) Build() (string, error) {
//     imageTag, err := c.GoImage()                    // Language-specific
//     if err != nil { return "", err }
//     
//     ssh, err := network.SSHForward(...)             // DUPLICATED
//     if err != nil { return "", err }
//     
//     opts := types.ContainerConfig{}                 // DUPLICATED  
//     opts.Image = imageTag                           // DUPLICATED
//     for key, value := range cfg.Environment {       // DUPLICATED
//         opts.Env = append(opts.Env, ...)            // DUPLICATED
//     }
//     opts.WorkingDir = cfg.WorkingDir                // DUPLICATED
//     
//     dir, _ := filepath.Abs(".")                     // DUPLICATED
//     if c.Folder != "" {                             // DUPLICATED
//         dir, _ = filepath.Abs(c.Folder)             // DUPLICATED
//     }
//     
//     cache, err := CacheFolder()                     // DUPLICATED PATTERN
//     opts.Volumes = []types.Volume{                  // DUPLICATED
//         {Source: dir, Target: cfg.WorkingDir},      // DUPLICATED
//         {Source: cache, Target: cfg.CacheLocation}, // DUPLICATED
//     }
//     
//     opts = ssh.Apply(&opts)                         // DUPLICATED
//     opts.Script = c.BuildScript()                   // Language-specific
//     
//     err = c.GetContainer().BuildingContainer(opts)  // DUPLICATED
//     if err != nil { return "", err }
//     // Optional commit logic...                     // Varies by language
// }
//
// func (c *GoContainer) Pull() error {
//     if err := c.GetContainer().Pull(c.BaseImage()); err != nil {  // DUPLICATED
//         return err
//     }
//     if err := c.GetContainer().Pull("alpine:latest"); err != nil { // DUPLICATED PATTERN
//         return err  
//     }
//     return nil
// }
//
// func (c *GoContainer) Images() []string {
//     baseImage := c.BaseImage()                      // DUPLICATED
//     goImage, err := c.GoImage()                     // Language-specific method name
//     if err != nil {                                 // DUPLICATED
//         return []string{baseImage, "alpine:latest"}  // DUPLICATED PATTERN
//     }
//     return []string{baseImage, "alpine:latest", goImage} // DUPLICATED PATTERN
// }
//
// AFTER (unified with orchestrator):
//
// func (orchestrator *ContainerBuildOrchestrator) Build(ctx context.Context) (string, error) {
//     imageTag, err := orchestrator.strategy.GetIntermediateImage(ctx)  // UNIFIED
//     ssh, err := network.SSHForward(...)                              // UNIFIED
//     opts := setupContainerConfig(imageTag, ssh, ...)                 // UNIFIED
//     script := orchestrator.strategy.GenerateBuildScript()            // UNIFIED
//     err = orchestrator.baseBuilder.GetContainer().BuildingContainer(opts) // UNIFIED
//     if orchestrator.strategy.ShouldCommitResult() {                  // UNIFIED
//         return orchestrator.commitResult()                           // UNIFIED
//     }
//     return containerID, nil
// }
//
// This reduces:
//   - Build() method: ~90% code deduplication across all language packages  
//   - Pull() method: 100% code deduplication across all language packages
//   - Images() method: ~90% code deduplication across all language packages
//   - Total impact: Eliminates hundreds of lines of duplicated orchestration code