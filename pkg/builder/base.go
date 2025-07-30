package builder

import (
	"fmt"
	"log/slog"

	"github.com/containifyci/engine-ci/pkg/builder/common"
	"github.com/containifyci/engine-ci/pkg/container"
	"github.com/containifyci/engine-ci/pkg/cri/types"
	"github.com/containifyci/engine-ci/pkg/network"
)

// BaseBuilder provides common functionality that can be embedded by all language builders.
// This struct implements shared patterns and reduces code duplication across implementations.
type BaseBuilder struct {
	*container.Container
	baseImage string
	cacheDir  string
	sourceDir string
	Defaults  common.LanguageDefaults
	images    []string
	Config    BuildConfiguration
}

// NewBaseBuilder creates a new BaseBuilder instance with common initialization.
func NewBaseBuilder(build container.Build, defaults common.LanguageDefaults) *BaseBuilder {
	// Create configuration from build
	config := BuildConfiguration{
		Platform:    build.Platform,
		Platforms:   common.GetDefaultPlatforms(build.Platform),
		Registry:    build.ContainifyRegistry,
		Environment: build.Env,
		Verbose:     build.Verbose,
		App:         build.App,
		File:        build.File,
		Folder:      build.Folder,
		Image:       build.Image,
		ImageTag:    build.ImageTag,
		Tags:        build.Custom.Strings("tags"),
		Custom:      build.Custom,
	}

	return &BaseBuilder{
		Container: container.New(build),
		Config:    config,
		Defaults:  defaults,
		images:    []string{},
		baseImage: defaults.BaseImage,
		sourceDir: ".",
	}
}

// Name returns the language name from defaults.
func (b *BaseBuilder) Name() string {
	return b.Defaults.Language
}

// IsAsync returns false by default. Override in specific implementations if needed.
func (b *BaseBuilder) IsAsync() bool {
	return false
}

// Images returns the list of Docker images used by this builder.
func (b *BaseBuilder) Images() []string {
	if len(b.images) == 0 {
		// Build default image list
		b.images = []string{b.baseImage}

		// Add intermediate image if different
		if intermediateImg := b.IntermediateImage(); intermediateImg != b.baseImage {
			b.images = append(b.images, intermediateImg)
		}
	}
	return b.images
}

// CacheFolder returns the language-specific cache directory.
// This method should be overridden by specific language implementations.
func (b *BaseBuilder) CacheFolder() string {
	if b.cacheDir == "" {
		// Use language-specific cache directory from defaults
		cacheDir := common.GetLanguageCacheDir(b.Defaults.Language)
		b.cacheDir = common.CacheFolderFromEnv(
			[]string{fmt.Sprintf("%s_CACHE_DIR", b.Defaults.Language)},
			cacheDir,
		)
	}
	return b.cacheDir
}

// ApplyContainerOptions applies common container configuration options.
// This method encapsulates shared logic for setting up container environments.
func (b *BaseBuilder) ApplyContainerOptions(opts *types.ContainerConfig) {
	// Common container setup that's shared across all language builders
	if opts.WorkingDir == "" {
		opts.WorkingDir = "/src"
	}

	// Apply verbose flag if set
	if b.Config.Verbose && len(opts.Cmd) > 0 && opts.Cmd[0] == "sh" {
		opts.Cmd = append(opts.Cmd, "-v")
	}
}

// SetupContainerEnvironment prepares common container environment settings.
func (b *BaseBuilder) SetupContainerEnvironment(opts *types.ContainerConfig) {
	// Apply base configuration
	b.ApplyContainerOptions(opts)

	// Set up environment variables
	if opts.Env == nil {
		opts.Env = []string{}
	}

	// Add language-specific environment variables
	for key, value := range b.Defaults.DefaultEnv {
		opts.Env = append(opts.Env, fmt.Sprintf("%s=%s", key, value))
	}

	// Add containify host if configured
	if host := common.GetContainifyHost(b.Config.Custom); host != "" {
		opts.Env = append(opts.Env, fmt.Sprintf("CONTAINIFYCI_HOST=%s", host))
	}
}

// SetupContainerVolumes prepares common volume mounts for the container.
func (b *BaseBuilder) SetupContainerVolumes(opts *types.ContainerConfig) {
	// Always mount the project root directory (not the specific folder)
	// The build script will cd into the specific folder as needed
	sourceDir := b.sourceDir

	// Set up common volumes (source + cache)
	opts.Volumes = common.SetupCommonVolumes(
		sourceDir,
		b.CacheFolder(),
		b.Defaults.SourceMount,
		b.Defaults.CacheMount,
	)
}

// SetupSSHForwarding configures SSH agent forwarding for the container.
func (b *BaseBuilder) SetupSSHForwarding(opts *types.ContainerConfig) error {
	ssh, err := network.SSHForward(*b.GetBuild())
	if err != nil {
		slog.Error("Failed to forward SSH", "error", err)
		return fmt.Errorf("failed to setup SSH forwarding: %w", err)
	}

	*opts = ssh.Apply(opts)
	return nil
}

// ExecuteBuildContainer runs the build process in a container with common setup.
func (b *BaseBuilder) ExecuteBuildContainer(imageTag, script string) error {
	opts := types.ContainerConfig{
		Image:      imageTag,
		WorkingDir: b.Defaults.SourceMount,
		Script:     script,
	}

	// Apply common container setup
	b.SetupContainerEnvironment(&opts)
	b.SetupContainerVolumes(&opts)

	// Setup SSH forwarding
	if err := b.SetupSSHForwarding(&opts); err != nil {
		return err
	}

	// Execute the build
	err := b.BuildingContainer(opts)
	if err != nil {
		slog.Error("Failed to build container", "error", err)
		return fmt.Errorf("build failed: %w", err)
	}

	return nil
}

// ValidateProject checks if the project has required files for this language.
func (b *BaseBuilder) ValidateProject() error {
	projectDir := b.sourceDir
	if b.Config.Folder != "" {
		projectDir = b.Config.Folder
	}

	return common.ValidateRequiredFiles(projectDir, b.Defaults.RequiredFiles)
}

// CreateProductionContainer sets up a production container with standard configuration.
func (b *BaseBuilder) CreateProductionContainer(prodImage string) error {
	if b.GetBuild().Env == container.LocalEnv {
		slog.Info("Skip building prod image in local environment")
		return nil
	}

	if b.Config.Image == "" {
		slog.Info("Skip No image specified to push")
		return nil
	}

	return common.CreateProdContainer(b.Container, prodImage)
}

// CommitProductionImage creates and tags the final production image.
func (b *BaseBuilder) CommitProductionImage(cmd, user, workdir string) (string, error) {
	imageUri := fmt.Sprintf("%s:%s", b.Config.Image, b.Config.ImageTag)

	commitArgs := []string{
		fmt.Sprintf("CMD [%s]", cmd),
	}

	if user != "" {
		commitArgs = append(commitArgs, fmt.Sprintf("USER %s", user))
	}

	if workdir != "" {
		commitArgs = append(commitArgs, fmt.Sprintf("WORKDIR %s", workdir))
	}

	return b.Commit(imageUri, "Created from container", commitArgs...)
}

// SetupUserInContainer creates a non-root user in the production container.
func (b *BaseBuilder) SetupUserInContainer() error {
	return common.AddUserToContainer(b.Container, "11211", "1121", "app", "app")
}

// CopyApplicationToContainer copies the built application to the production container.
func (b *BaseBuilder) CopyApplicationToContainer(sourcePath, targetPath string) error {
	// Get container info for architecture-specific binary selection
	containerInfo, err := b.Inspect()
	if err != nil {
		slog.Error("Failed to inspect container", "error", err)
		return fmt.Errorf("failed to inspect container: %w", err)
	}

	slog.Info("Container info",
		"name", containerInfo.Name,
		"image", containerInfo.Image,
		"arch", containerInfo.Platform.Container.Architecture,
		"os", containerInfo.Platform.Container.OS,
	)

	// Build architecture-specific source path if needed
	if b.Config.App != "" {
		sourcePath = fmt.Sprintf("%s/%s-%s-%s",
			b.Config.Folder,
			b.Config.App,
			containerInfo.Platform.Container.OS,
			containerInfo.Platform.Container.Architecture,
		)

		targetPath = fmt.Sprintf("%s/%s", targetPath, b.Config.App)
	}

	return b.CopyFileTo(sourcePath, targetPath)
}

// DefaultPullImages pulls the base images required by this builder.
func (b *BaseBuilder) DefaultPullImages(additionalImages ...string) error {
	imagesToPull := []string{b.baseImage}
	imagesToPull = append(imagesToPull, additionalImages...)

	for _, image := range imagesToPull {
		if err := b.Container.Pull(image); err != nil {
			return fmt.Errorf("failed to pull image %s: %w", image, err)
		}
	}

	return nil
}

// IntermediateImage returns the intermediate image name.
// This should be overridden by specific language implementations.
func (b *BaseBuilder) IntermediateImage() string {
	// Default implementation returns base image
	// Language-specific builders should override this
	return b.baseImage
}

// BuildIntermediateImage builds the intermediate language-specific image.
// This should be overridden by specific language implementations.
func (b *BaseBuilder) BuildIntermediateImage() error {
	// Default implementation is a no-op
	// Language-specific builders should override this
	slog.Info("Using base image as intermediate image", "image", b.baseImage)
	return nil
}

// BuildScript generates the build script for the container.
// This should be overridden by specific language implementations.
func (b *BaseBuilder) BuildScript() string {
	// Default implementation returns empty script
	// Language-specific builders should override this
	return "echo 'No build script defined for " + b.Defaults.Language + "'"
}

// Build executes the main build process.
// This should be overridden by specific language implementations.
func (b *BaseBuilder) Build() error {
	// Validate project structure
	if err := b.ValidateProject(); err != nil {
		return fmt.Errorf("project validation failed: %w", err)
	}

	// Default implementation calls ExecuteBuildContainer
	return b.ExecuteBuildContainer(b.IntermediateImage(), b.BuildScript())
}

// Pull pulls required base images.
// This can be overridden by specific language implementations if needed.
func (b *BaseBuilder) Pull() error {
	return b.DefaultPullImages()
}

// Run executes the complete build pipeline.
// This can be overridden by specific language implementations if needed.
func (b *BaseBuilder) Run() error {
	// Pull base images
	if err := b.Pull(); err != nil {
		slog.Error("Failed to pull base images", "error", err)
		return err
	}

	// Build intermediate image
	if err := b.BuildIntermediateImage(); err != nil {
		slog.Error("Failed to build intermediate image", "error", err)
		return err
	}

	// Execute main build
	if err := b.Build(); err != nil {
		slog.Error("Failed to execute build", "error", err)
		return err
	}

	slog.Info("Build completed successfully", "containerId", b.ID)
	return nil
}

// Prod creates a production-ready container image.
// This should be overridden by specific language implementations.
func (b *BaseBuilder) Prod() error {
	// Default implementation returns an error
	return fmt.Errorf("production build not implemented for %s", b.Defaults.Language)
}
