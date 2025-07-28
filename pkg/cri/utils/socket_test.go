package utils

import (
	"testing"

	"github.com/containifyci/engine-ci/pkg/cri/types"
	"github.com/stretchr/testify/assert"
)

func TestDockerSocket(t *testing.T) {
	socket, err := DockerSocket()
	assert.NoError(t, err)
	assert.NotNil(t, socket)
	assert.Equal(t, Docker, socket.RuntimeType)
	assert.Equal(t, "/var/run/docker.sock", socket.Source)
	assert.Equal(t, "/var/run/docker.sock", socket.Target)
}

func TestPodmanSocket(t *testing.T) {
	// Since calling PodmanSocket would require podman to be installed on the system, we can only run this test
	// if Podman is available. For the purpose of this exercise, assume it is installed.

	socket, err := PodmanSocket()

	if assert.NoError(t, err) && assert.NotNil(t, socket) {
		assert.Equal(t, Podman, socket.RuntimeType)
		assert.Equal(t, "/var/run/podman.sock", socket.Target)
		assert.NotEmpty(t, socket.Source, "Podman socket source should not be empty if command is successful")
	}
}

func TestSocket(t *testing.T) {
	tests := []struct {
		name        string
		runtimeType RuntimeType
		expectError bool
	}{
		{"Docker Socket", Docker, false},
		{"Podman Socket", Podman, false},
		{"Unknown Runtime", "unknown", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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
		name        string
		runtimeType RuntimeType
		initialOpts types.ContainerConfig
		expectedLen int
	}{
		{"Docker Apply", Docker, types.ContainerConfig{}, 1},
		{"Podman Apply", Podman, types.ContainerConfig{}, 1},
		{"Unknown Apply", "unknown", types.ContainerConfig{}, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ApplySocket(tt.runtimeType, &tt.initialOpts)
			assert.Equal(t, tt.expectedLen, len(result.Volumes))
		})
	}
}
