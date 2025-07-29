package common

import (
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"
	"time"

	"github.com/containifyci/engine-ci/pkg/container"
	"github.com/containifyci/engine-ci/pkg/cri/types"
	"github.com/containifyci/engine-ci/pkg/cri/utils"
	"github.com/containifyci/engine-ci/pkg/filesystem"
	"github.com/containifyci/engine-ci/pkg/memory"
	u "github.com/containifyci/engine-ci/pkg/utils"
)

// ComputeChecksum computes SHA256 checksum of data.
// This function consolidates the duplicate implementations across language packages.
func ComputeChecksum(data []byte) string {
	start := time.Now()
	defer func() {
		memory.TrackOperation(time.Since(start))
	}()
	
	hash := sha256.Sum256(data)
	result := hex.EncodeToString(hash[:])
	
	memory.TrackAllocation(int64(len(result)))
	return result
}

// ImageURIFromDockerfile creates a container image URI from a Dockerfile and base name.
// This function encapsulates the common pattern used across all language builders.
func ImageURIFromDockerfile(fs embed.FS, dockerfilePath, baseName, registry string) string {
	dockerFile, err := fs.ReadFile(dockerfilePath)
	if err != nil {
		slog.Error("Failed to read Dockerfile", "path", dockerfilePath, "error", err)
		os.Exit(1)
	}
	
	tag := ComputeChecksum(dockerFile)
	return utils.ImageURI(registry, baseName, tag)
}

// BuildIntermediateImage builds an intermediate container image using the provided Dockerfile.
// This consolidates the common intermediate image building pattern across all language builders.
func BuildIntermediateImage(c *container.Container, fs embed.FS, dockerfilePath, imageName string, platforms ...string) error {
	dockerFile, err := fs.ReadFile(dockerfilePath)
	if err != nil {
		slog.Error("Failed to read Dockerfile", "path", dockerfilePath, "error", err)
		return fmt.Errorf("failed to read dockerfile %s: %w", dockerfilePath, err)
	}

	if len(platforms) == 0 {
		platforms = types.GetPlatforms(c.GetBuild().Platform)
	}
	
	slog.Info("Building intermediate image", "image", imageName, "platforms", platforms)
	return c.BuildIntermidiateContainer(imageName, dockerFile, platforms...)
}

// CacheFolderFromEnv determines the cache folder location from environment variables or defaults.
// This function provides a consistent approach to cache folder resolution across all builders.
func CacheFolderFromEnv(envVars []string, defaultSubDir string) string {
	// Try environment variables first
	for _, envVar := range envVars {
		if value := os.Getenv(envVar); value != "" {
			return value
		}
	}
	
	// Fallback to user home directory
	usr, err := user.Current()
	if err != nil {
		slog.Error("Failed to get current user", "error", err)
		os.Exit(1)
	}
	
	cacheDir := filepath.Join(usr.HomeDir, defaultSubDir)
	slog.Info("Cache directory not set via environment, using default", "cacheDir", cacheDir)
	
	// Ensure directory exists
	if err := filesystem.DirectoryExists(cacheDir); err != nil {
		slog.Error("Failed to create cache folder", "error", err)
		os.Exit(1)
	}
	
	return cacheDir
}

// CacheFolderFromCommand determines the cache folder by executing a command.
// This is used by languages like Go that have built-in cache location commands.
func CacheFolderFromCommand(command, subcommand string, fallbackDir string) string {
	// Try to get cache location from command
	cmd := exec.Command(command, subcommand)
	output, err := cmd.Output()
	if err != nil {
		slog.Warn("Failed to execute cache location command", "command", command, "subcommand", subcommand, "error", err)
		return fallbackDir
	}
	
	cacheLocation := strings.TrimSpace(string(output))
	if cacheLocation != "" {
		slog.Info("Cache location detected", "location", cacheLocation)
		return cacheLocation
	}
	
	return fallbackDir
}

// SetupCommonVolumes creates the common volume mounts used by most language builders.
// This includes source code mount and cache directory mount.
func SetupCommonVolumes(sourceDir, cacheDir, sourceMountPath, cacheMountPath string) []types.Volume {
	// Resolve absolute paths
	absSourceDir, err := filepath.Abs(sourceDir)
	if err != nil {
		slog.Error("Failed to resolve source directory", "dir", sourceDir, "error", err)
		absSourceDir = sourceDir
	}
	
	absCacheDir, err := filepath.Abs(cacheDir)
	if err != nil {
		slog.Error("Failed to resolve cache directory", "dir", cacheDir, "error", err)
		absCacheDir = cacheDir
	}
	
	return []types.Volume{
		{
			Type:   "bind",
			Source: absSourceDir,
			Target: sourceMountPath,
		},
		{
			Type:   "bind",
			Source: absCacheDir,
			Target: cacheMountPath,
		},
	}
}

// SetupEnvironmentVariables creates common environment variables for container builds.
// This function encapsulates environment setup patterns shared across language builders.
func SetupEnvironmentVariables(languageSpecific []string, cacheVars map[string]string) []string {
	env := make([]string, 0, len(languageSpecific)+len(cacheVars))
	
	// Add language-specific environment variables
	env = append(env, languageSpecific...)
	
	// Add cache-related environment variables
	for key, value := range cacheVars {
		env = append(env, fmt.Sprintf("%s=%s", key, value))
	}
	
	return env
}

// CreateProdContainer sets up a production container with common configuration.
// This consolidates the production container setup pattern used across language builders.
func CreateProdContainer(c *container.Container, prodImage string) error {
	opts := types.ContainerConfig{
		Image:     prodImage,
		Env:       []string{},
		Cmd:       []string{"sleep", "300"},
		Platform:  types.AutoPlatform,
		WorkingDir: "/src",
	}
	
	return c.Create(opts)
}

// AddUserToContainer adds a user and group to the container for production deployments.
// This encapsulates the common pattern of creating non-root users in production containers.
func AddUserToContainer(c *container.Container, groupID, userID, groupName, userName string) error {
	// Add group
	if err := c.Exec("addgroup", "-g", groupID, groupName); err != nil {
		return fmt.Errorf("failed to add group: %w", err)
	}
	
	// Add user
	if err := c.Exec("adduser", "-D", "-u", userID, "-G", groupName, userName); err != nil {
		return fmt.Errorf("failed to add user: %w", err)
	}
	
	return nil
}

// GetDefaultPlatforms returns the appropriate platforms for building based on the host platform.
// This consolidates the platform detection logic used across language builders.
func GetDefaultPlatforms(buildPlatform types.Platform) []*types.PlatformSpec {
	platforms := []*types.PlatformSpec{buildPlatform.Container}
	
	// Add cross-platform support when host and container platforms differ
	if !buildPlatform.Same() {
		slog.Info("Different platform detected", "host", buildPlatform.Host, "container", buildPlatform.Container)
		// For cross-platform builds, include both darwin/arm64 and linux/arm64
		platforms = []*types.PlatformSpec{
			types.ParsePlatform("darwin/arm64"),
			types.ParsePlatform("linux/arm64"),
		}
	}
	
	return platforms
}

// GetDefaultPlatformStrings returns platform strings for building, compatible with BuildIntermediateContainer.
// This is a convenience function that returns []string instead of []*PlatformSpec.
func GetDefaultPlatformStrings(buildPlatform types.Platform) []string {
	return types.GetPlatforms(buildPlatform)
}

// DefaultCacheLocations provides standard cache directory locations for different languages.
var DefaultCacheLocations = map[string]string{
	"go":     ".cache/go",
	"maven":  ".m2",
	"python": ".cache/pip",
	"node":   ".cache/npm",
	"generic": ".cache/build",
}

// GetLanguageCacheDir returns the default cache directory for a specific language.
func GetLanguageCacheDir(language string) string {
	if dir, exists := DefaultCacheLocations[language]; exists {
		return dir
	}
	return DefaultCacheLocations["generic"]
}

// ValidateRequiredFiles checks if required files exist in the project directory.
// This is used by language builders to validate project structure before building.
func ValidateRequiredFiles(projectDir string, requiredFiles []string) error {
	for _, file := range requiredFiles {
		filePath := filepath.Join(projectDir, file)
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			return fmt.Errorf("required file not found: %s", file)
		}
	}
	return nil
}

// GetContainifyHost retrieves the CONTAINIFYCI_HOST from custom configuration.
// This is used for internal container communication in the containify CI system.
func GetContainifyHost(custom container.Custom) string {
	if v, ok := custom["CONTAINIFYCI_HOST"]; ok && len(v) > 0 {
		return v[0]
	}
	return ""
}

// IsPrivilegedMode checks if containers should run in privileged mode.
// This affects security settings and capabilities available to containers.
func IsPrivilegedMode() bool {
	return u.GetEnv("CONTAINER_PRIVILGED", "build") != "false"
}