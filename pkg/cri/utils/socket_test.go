package utils

import (
	"os/exec"
	"testing"

	"github.com/containifyci/engine-ci/pkg/cri/types"
	"github.com/stretchr/testify/assert"
)

// isPodmanAvailable checks if podman is available and properly configured
func isPodmanAvailable() bool {
	_, err := exec.LookPath("podman")
	if err != nil {
		return false
	}

	// Test if podman is properly configured by trying a simple command
	cmd := exec.Command("podman", "info", "-f", "{{ .Host.RemoteSocket.Path }}")
	err = cmd.Run()
	return err == nil
}

// isDockerAvailable checks if docker is available and properly configured
func isDockerAvailable() bool {
	_, err := exec.LookPath("docker")
	if err != nil {
		return false
	}

	// Test if docker is properly configured by trying a simple command
	cmd := exec.Command("docker", "info")
	err = cmd.Run()
	return err == nil
}

func TestDockerSocket(t *testing.T) {
	socket, err := DockerSocket()
	assert.NoError(t, err)
	assert.NotNil(t, socket)
	assert.Equal(t, Docker, socket.RuntimeType)
	assert.Equal(t, "/var/run/docker.sock", socket.Source)
	assert.Equal(t, "/var/run/docker.sock", socket.Target)
}

func TestPodmanSocket(t *testing.T) {
	// Skip test if podman is not available
	if !isPodmanAvailable() {
		t.Skip("Podman not available, skipping test")
	}

	socket, err := PodmanSocket()

	if assert.NoError(t, err) && assert.NotNil(t, socket) {
		assert.Equal(t, Podman, socket.RuntimeType)
		assert.Equal(t, "/var/run/podman.sock", socket.Target)
		assert.NotEmpty(t, socket.Source, "Podman socket source should not be empty if command is successful")
	}
}

func TestSocket(t *testing.T) {
	tests := []struct { //nolint:govet
		skipIf      func() bool
		name        string
		runtimeType RuntimeType
		expectError bool
	}{
		{func() bool { return !isDockerAvailable() }, "Docker Socket", Docker, false},
		{func() bool { return !isPodmanAvailable() }, "Podman Socket", Podman, false},
		{func() bool { return false }, "Unknown Runtime", "unknown", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skipIf() {
				t.Skipf("Skipping test - %s runtime not available", tt.runtimeType)
			}

			socket, err := Socket(tt.runtimeType)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, socket)
			}
		})
	}
}

func TestApplySocket(t *testing.T) {
	tests := []struct {
		skipIf      func() bool
		name        string
		runtimeType RuntimeType
		initialOpts types.ContainerConfig
		expectedLen int
	}{
		{func() bool { return !isDockerAvailable() }, "Docker Apply", Docker, types.ContainerConfig{}, 1},
		{func() bool { return !isPodmanAvailable() }, "Podman Apply", Podman, types.ContainerConfig{}, 1},
		{func() bool { return false }, "Unknown Apply", "unknown", types.ContainerConfig{}, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skipIf() {
				t.Skipf("Skipping test - %s runtime not available", tt.runtimeType)
			}

			result := ApplySocket(tt.runtimeType, &tt.initialOpts)
			assert.Equal(t, tt.expectedLen, len(result.Volumes))
		})
	}
}
