package doctor

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/containifyci/engine-ci/pkg/container"
	"github.com/containifyci/engine-ci/pkg/cri"
	"github.com/containifyci/engine-ci/pkg/cri/types"
	"github.com/containifyci/engine-ci/pkg/cri/utils"
)

// VolumeConfigCheck verifies volume mount configuration
type VolumeConfigCheck struct {
	*Check
}

// NewVolumeConfigCheck creates a new volume configuration check
func NewVolumeConfigCheck() *VolumeConfigCheck {
	return &VolumeConfigCheck{
		Check: &Check{
			Name:      "Volume Mount Configuration",
			Category:  CategoryPermissions,
			Severity:  SeverityWarning,
			ShouldRun: true,
		},
	}
}

func (c *VolumeConfigCheck) Run(_ context.Context) CheckResult {
	result := c.NewCheckResult()

	// Detect OS
	operatingSystem := runtime.GOOS
	result.Metadata["os"] = operatingSystem

	// Check environment variables (all platforms)
	if privileged := os.Getenv("CONTAINER_PRIVILGED"); privileged != "" {
		result.Metadata["container_privileged"] = privileged
		result.Details = append(result.Details, fmt.Sprintf("CONTAINER_PRIVILGED: %s", privileged))
	}

	// Detect container runtime
	detectedRuntime := cri.DetectContainerRuntime()
	result.Metadata["runtime"] = string(detectedRuntime)

	// OS-specific checks
	switch operatingSystem {
	case "linux":
		platformMsg := fmt.Sprintf("Platform: Linux (%s)", detectedRuntime)
		result.Details = append(result.Details, platformMsg)
		switch detectedRuntime {
		case utils.Docker:
			c.checkLinuxDockerConfig(&result)
		case utils.Podman:
			result.Details = append(result.Details, "Podman uses different permission model - user namespace checks not applicable")
		}

	case "darwin":
		switch detectedRuntime {
		case utils.Docker:
			result.Details = append(result.Details, "Platform: macOS (Docker Desktop)")
			result.Details = append(result.Details, "Docker Desktop manages volume permissions via VM")
			result.Details = append(result.Details, "No host configuration needed - handled by Docker Desktop")
		case utils.Podman:
			result.Details = append(result.Details, "Platform: macOS (Podman)")
			result.Details = append(result.Details, "Podman on macOS runs in a VM (similar to Docker Desktop)")
			result.Details = append(result.Details, "Volume permissions handled by Podman machine VM")
		default:
			result.Details = append(result.Details, fmt.Sprintf("Platform: macOS (%s)", detectedRuntime))
			result.Details = append(result.Details, "Volume permissions handled by container runtime")
		}

	case "windows":
		switch detectedRuntime {
		case utils.Docker:
			result.Details = append(result.Details, "Platform: Windows (Docker Desktop)")
			result.Details = append(result.Details, "Volume permissions handled by Docker Desktop/WSL2")
		case utils.Podman:
			result.Details = append(result.Details, "Platform: Windows (Podman)")
			result.Details = append(result.Details, "Podman on Windows runs in WSL2 VM")
			result.Details = append(result.Details, "Volume permissions handled by WSL2")
		default:
			result.Details = append(result.Details, fmt.Sprintf("Platform: Windows (%s)", detectedRuntime))
			result.Details = append(result.Details, "Volume permissions handled by container runtime")
		}

	default:
		result.Details = append(result.Details, fmt.Sprintf("Platform: %s (unknown)", operatingSystem))
	}

	result.Status = StatusPass
	result.Message = fmt.Sprintf("Volume configuration checked on %s", operatingSystem)

	return result
}

// checkLinuxDockerConfig checks Linux-specific Docker configuration
func (c *VolumeConfigCheck) checkLinuxDockerConfig(result *CheckResult) {
	// Check /etc/subuid
	if _, err := os.Stat("/etc/subuid"); err == nil {
		result.Metadata["subuid_exists"] = true
		result.Details = append(result.Details, "/etc/subuid: exists")
	} else {
		result.Metadata["subuid_exists"] = false
		result.Details = append(result.Details, "/etc/subuid: not found")
	}

	// Check /etc/subgid
	if _, err := os.Stat("/etc/subgid"); err == nil {
		result.Metadata["subgid_exists"] = true
		result.Details = append(result.Details, "/etc/subgid: exists")
	} else {
		result.Metadata["subgid_exists"] = false
		result.Details = append(result.Details, "/etc/subgid: not found")
	}

	// Check daemon.json
	daemonJSON := "/etc/docker/daemon.json"
	if data, err := os.ReadFile(daemonJSON); err == nil {
		var config map[string]interface{}
		if err := json.Unmarshal(data, &config); err == nil {
			if usernsRemap, ok := config["userns-remap"]; ok {
				result.Metadata["userns_remap"] = usernsRemap
				result.Details = append(result.Details, fmt.Sprintf("daemon.json: userns-remap=%v", usernsRemap))
			} else {
				result.Metadata["userns_remap"] = nil
				result.Details = append(result.Details, "daemon.json: no userns-remap configured")
			}
		} else {
			result.Details = append(result.Details, "daemon.json: parse error")
		}
	} else {
		result.Details = append(result.Details, "daemon.json: not found")
		result.Suggestions = append(result.Suggestions,
			"Consider running 'engine-ci github_actions' to configure volume permissions for GitHub Actions",
		)
	}
}

// VolumeWriteTestCheck performs an integration test of volume mounting
type VolumeWriteTestCheck struct {
	*Check
	KeepTestContainers bool
}

// NewVolumeWriteTestCheck creates a new volume write test check
func NewVolumeWriteTestCheck(keepTestContainers bool) *VolumeWriteTestCheck {
	detectedRuntime := cri.DetectContainerRuntime()
	return &VolumeWriteTestCheck{
		Check: &Check{
			Name:      "Volume Write Permission Test",
			Category:  CategoryPermissions,
			Severity:  SeverityCritical,
			ShouldRun: detectedRuntime != utils.RuntimeType("unknown"),
		},
		KeepTestContainers: keepTestContainers,
	}
}

func (c *VolumeWriteTestCheck) Run(ctx context.Context) CheckResult {
	result := c.NewCheckResult()

	// Setup timeout
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Track cleanup resources and test success
	var containerID string
	var tempDir string
	testPassed := false

	// Defer cleanup - only cleanup if test passed AND KeepTestContainers is false
	defer func() {
		shouldCleanup := testPassed && !c.KeepTestContainers

		if containerID != "" && shouldCleanup {
			// Use background context for cleanup
			cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cleanupCancel()

			manager, err := cri.InitContainerRuntime()
			if err == nil {
				_ = manager.RemoveContainer(cleanupCtx, containerID)
			}
		} else if containerID != "" && !shouldCleanup {
			// Log that container was kept for investigation
			slog.Info("Test container preserved for investigation",
				"container_id", containerID,
				"reason", func() string {
					if !testPassed {
						return "test failed"
					}
					return "KeepTestContainers option enabled"
				}())
		}

		if tempDir != "" && shouldCleanup {
			_ = os.RemoveAll(tempDir)
		} else if tempDir != "" && !shouldCleanup {
			slog.Info("Test directory preserved for investigation", "temp_dir", tempDir)
		}
	}()

	// Phase 1: Setup temp directory
	var err error
	tempDir, err = os.MkdirTemp("", "engine-ci-doctor-*")
	if err != nil {
		result.Status = StatusFail
		result.Message = "Failed to create temporary directory"
		result.Error = err
		result.Details = []string{fmt.Sprintf("Error: %v", err)}
		result.Suggestions = []string{
			"Check disk space: df -h",
			"Check temp directory permissions",
		}
		return result
	}

	// Generate test content
	testContent := fmt.Sprintf("engine-ci-volume-test-%d", time.Now().Unix())
	result.Metadata["test_content"] = testContent
	result.Metadata["temp_dir"] = tempDir

	// Phase 2: Initialize container manager
	manager, err := cri.InitContainerRuntime()
	if err != nil {
		result.Status = StatusFail
		result.Message = "Failed to initialize container runtime"
		result.Error = err
		result.Details = []string{fmt.Sprintf("Error: %v", err)}
		result.Suggestions = []string{
			"Check runtime connectivity with 'Runtime API Connectivity' check",
			"Verify Docker/Podman daemon is running",
		}
		return result
	}

	// Phase 3: Create and run test container
	containerID, err = c.runVolumeTest(ctx, manager, tempDir, testContent)
	if err != nil {
		result.Status = StatusFail
		result.Message = "Volume write test failed"
		result.Error = err
		result.Details = []string{fmt.Sprintf("Error: %v", err)}
		result.Suggestions = c.getSuggestions(err)
		return result
	}

	result.Metadata["container_id"] = containerID

	runtime := cri.DetectContainerRuntime()
	result.Metadata["runtime"] = runtime

	if runtime == utils.Host {
		tempDir = tempDir + "/src"
	}

	if runtime == utils.Test {
		result.Status = StatusSkipped
		result.Message = "Volume write test skipped for test runtime"
		return result
	}

	// Phase 4: Verify file was created
	testFile := fmt.Sprintf("%s/testfile.txt", tempDir)
	content, err := os.ReadFile(testFile)
	if err != nil {
		result.Status = StatusFail
		result.Message = "Test file was not created on host filesystem"
		result.Error = err
		result.Details = []string{
			"Container executed successfully but file is missing",
			fmt.Sprintf("Expected file: %s", testFile),
		}
		result.Suggestions = []string{
			"Volume mounting may not be working correctly",
			"Check SELinux/AppArmor policies if on Linux",
			"Verify mount propagation settings",
		}
		return result
	}

	// Verify content matches
	actualContent := string(content)
	if actualContent != testContent+"\n" { // echo adds newline
		result.Status = StatusFail
		result.Message = "Test file content does not match expected"
		result.Details = []string{
			fmt.Sprintf("Expected: %s", testContent),
			fmt.Sprintf("Got: %s", actualContent),
		}
		result.Suggestions = []string{
			"Volume data corruption or caching issue",
		}
		return result
	}

	// Success!
	result.Status = StatusPass
	result.Message = "Volume write permissions working correctly"
	result.Metadata["test_file_created"] = true
	result.Metadata["content_verified"] = true
	testPassed = true // Mark test as passed for cleanup logic

	return result
}

// runVolumeTest creates and runs the test container
func (c *VolumeWriteTestCheck) runVolumeTest(ctx context.Context, manager cri.ContainerManager, tempDir, testContent string) (string, error) {
	// Prepare container configuration
	containerName := fmt.Sprintf("engine-ci-doctor-volume-test-%d", time.Now().Unix())
	imageName := "alpine:latest"

	// Create container config
	opts := types.ContainerConfig{
		Name:  containerName,
		Image: imageName,
		Cmd: []string{
			"/bin/sh", "-c",
			fmt.Sprintf("echo '%s' > /src/testfile.txt && cat /src/testfile.txt && sleep 1", testContent),
		},
		Volumes: []types.Volume{
			{
				Type:   "bind",
				Source: tempDir,
				Target: "/src",
			},
		},
	}

	con := container.NewWithManager(manager)
	con.Name = containerName
	con.Image = imageName

	// Try to pull the image
	err := con.Pull(imageName)
	if err != nil {
		return "", fmt.Errorf("image not found and pull failed: %w", err)
	}

	err = con.Create(opts)
	if err != nil {
		return "", fmt.Errorf("failed to create container after image pull: %w", err)
	}

	if err := con.Start(); err != nil {
		return con.ID, fmt.Errorf("failed to start container: %w", err)
	}

	containerInfo, err := con.Inspect()
	if err != nil {
		fmt.Printf("Failed to inspect container %v", err)
		os.Exit(1)
	}

	slog.Info("Container info", "id", con.ID, "name", containerInfo.Name, "image", containerInfo.Image, "arch", containerInfo.Platform.Container.Architecture, "os", containerInfo.Platform.Container.OS, "varian", containerInfo.Platform.Container.Variant)

	err = con.Wait()
	if err != nil {
		return con.ID, fmt.Errorf("failed to wait for container: %w", err)
	}

	return con.ID, nil
}

// getSuggestions returns appropriate suggestions based on the error
func (c *VolumeWriteTestCheck) getSuggestions(err error) []string {
	if err == nil {
		return nil
	}

	errMsg := err.Error()

	// Image-related errors
	if strings.Contains(errMsg, "No such image") || strings.Contains(errMsg, "image not found") {
		return []string{
			"Test image (alpine:latest) not found locally",
			"Check network connectivity to pull images",
			"Try manually pulling: docker pull alpine:latest",
			"If behind a proxy, configure Docker proxy settings",
		}
	}

	if strings.Contains(errMsg, "pull") && strings.Contains(errMsg, "failed") {
		return []string{
			"Failed to pull test image from registry",
			"Check network connectivity",
			"Verify Docker Hub or registry is accessible",
			"If behind a firewall, check registry access",
		}
	}

	// Permission-related errors
	if strings.Contains(errMsg, "permission denied") || strings.Contains(errMsg, "access denied") {
		return []string{
			"Volume mount permissions may be incorrect",
			"Check user namespace configuration if on Linux",
			"If using Docker, verify userns-remap settings",
			"Try running: engine-ci github_actions",
		}
	}

	// Container execution errors
	if strings.Contains(errMsg, "exited with code") {
		return []string{
			"Container failed to execute write test",
			"Volume mount may not be writable",
			"Check SELinux/AppArmor policies if on Linux",
			"Verify mount permissions with: ls -la <temp-dir>",
		}
	}

	// Generic suggestions
	return []string{
		"Volume write test encountered an error",
		"Check runtime logs for more details",
		"Verify Docker/Podman daemon is healthy",
	}
}
