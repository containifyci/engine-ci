package doctor

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/containifyci/engine-ci/pkg/cri"
	"github.com/containifyci/engine-ci/pkg/cri/utils"
)

type runtimeDetectionCheck struct {
	*Check
}

// NewRuntimeDetectionCheck creates a new runtime detection check
func NewRuntimeDetectionCheck() *runtimeDetectionCheck {
	return &runtimeDetectionCheck{
		Check: &Check{
			Name:      "Container Runtime Detection",
			Category:  CategoryRuntime,
			Severity:  SeverityCritical,
			ShouldRun: true,
		},
	}
}

func (c *runtimeDetectionCheck) Run(_ context.Context) CheckResult {
	result := c.NewCheckResult()

	// Detect runtime using existing logic
	runtime := cri.DetectContainerRuntime()

	if runtime == utils.RuntimeType("unknown") {
		result.Status = StatusFail
		result.Message = "No container runtime detected"
		result.Details = []string{
			"Neither Docker nor Podman was found in PATH",
			"Engine-CI requires a container runtime to function",
		}
		result.Suggestions = []string{
			"Install Docker: https://docs.docker.com/engine/install/",
			"Or install Podman: https://podman.io/getting-started/installation",
			"Ensure the runtime binary is in your PATH",
		}
		result.Metadata["runtime"] = "none"
		return result
	}

	result.Status = StatusPass
	result.Message = fmt.Sprintf("Detected container runtime: %s", runtime)
	result.Metadata["runtime"] = string(runtime)

	return result
}

// RuntimeConnectivityCheck verifies runtime API connectivity
type RuntimeConnectivityCheck struct {
	*Check
}

// NewRuntimeConnectivityCheck creates a new runtime connectivity check
func NewRuntimeConnectivityCheck() *RuntimeConnectivityCheck {
	runtime := cri.DetectContainerRuntime()
	return &RuntimeConnectivityCheck{
		Check: &Check{
			Name:      "Runtime API Connectivity",
			Category:  CategoryConnectivity,
			Severity:  SeverityCritical,
			ShouldRun: runtime != utils.RuntimeType("unknown"),
		},
	}
}

func (c *RuntimeConnectivityCheck) Run(ctx context.Context) CheckResult {
	result := c.NewCheckResult()

	runtime := cri.DetectContainerRuntime()

	// Try to initialize the container manager
	manager, err := cri.InitContainerRuntime()
	if err != nil {
		result.Status = StatusFail
		result.Message = fmt.Sprintf("Failed to connect to %s", runtime)
		result.Error = err
		result.Details = []string{
			fmt.Sprintf("Error: %v", err),
		}

		// Provide runtime-specific suggestions
		switch runtime {
		case utils.Docker:
			result.Suggestions = []string{
				"Check if Docker daemon is running: systemctl status docker",
				"Verify Docker socket exists: ls -la /var/run/docker.sock",
				"Check socket permissions: docker ps",
				"Add your user to docker group: sudo usermod -aG docker $USER",
				"Restart your session after adding to docker group",
			}
		case utils.Podman:
			result.Suggestions = []string{
				"Check if Podman socket is running: systemctl --user status podman.socket",
				"Start Podman socket: systemctl --user start podman.socket",
				"Verify Podman connection: podman info",
			}
		default:
			result.Suggestions = []string{
				"Check runtime daemon/service status",
				"Verify socket accessibility",
			}
		}

		result.Metadata["error"] = err.Error()
		return result
	}

	// Test basic connectivity by getting container list
	_, err = manager.ContainerList(ctx, false)
	if err != nil {
		result.Status = StatusWarning
		result.Message = fmt.Sprintf("Connected to %s but API call failed", runtime)
		result.Error = err
		result.Details = []string{
			fmt.Sprintf("Error listing containers: %v", err),
		}
		result.Suggestions = []string{
			"Check runtime logs for errors",
			"Verify runtime daemon is healthy",
		}
		result.Metadata["connected"] = true
		result.Metadata["api_working"] = false
		return result
	}

	result.Status = StatusPass
	result.Message = fmt.Sprintf("%s API is accessible and working", runtime)
	result.Metadata["connected"] = true
	result.Metadata["api_working"] = true

	return result
}

// RuntimeVersionCheck verifies runtime version compatibility
type RuntimeVersionCheck struct {
	*Check
}

// NewRuntimeVersionCheck creates a new runtime version check
func NewRuntimeVersionCheck() *RuntimeVersionCheck {
	runtime := cri.DetectContainerRuntime()
	return &RuntimeVersionCheck{
		Check: &Check{
			Name:      "Runtime Version Check",
			Category:  CategoryRuntime,
			Severity:  SeverityWarning,
			ShouldRun: runtime != utils.RuntimeType("unknown"),
		},
	}
}

func (c *RuntimeVersionCheck) Run(_ context.Context) CheckResult {
	result := c.NewCheckResult()

	runtime := cri.DetectContainerRuntime()

	var cmd *exec.Cmd
	switch runtime {
	case utils.Docker:
		cmd = exec.Command("docker", "version", "--format", "{{.Server.Version}}")
	case utils.Podman:
		cmd = exec.Command("podman", "version", "--format", "{{.Version}}")
	default:
		result.Status = StatusSkipped
		result.Message = "Version check not supported for this runtime"
		return result
	}

	output, err := cmd.Output()
	if err != nil {
		result.Status = StatusWarning
		result.Message = "Could not determine runtime version"
		result.Error = err
		result.Details = []string{
			fmt.Sprintf("Error: %v", err),
		}
		result.Suggestions = []string{
			fmt.Sprintf("Verify %s is properly installed", runtime),
			fmt.Sprintf("Try running: %s version", runtime),
		}
		return result
	}

	version := strings.TrimSpace(string(output))
	result.Status = StatusPass
	result.Message = fmt.Sprintf("%s version: %s", runtime, version)
	result.Metadata["version"] = version
	result.Metadata["runtime"] = string(runtime)

	return result
}
