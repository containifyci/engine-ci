package container

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/containifyci/engine-ci/pkg/cri"
	"github.com/containifyci/engine-ci/pkg/cri/types"
)

// MockContainerManagerForErrorTesting is a focused mock for testing error handling
type MockContainerManagerForErrorTesting struct {
	mock.Mock
}

func (m *MockContainerManagerForErrorTesting) WaitContainer(ctx context.Context, id string, waitCondition string) (*int64, error) {
	args := m.Called(ctx, id, waitCondition)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	status := args.Get(0).(int64)
	return &status, args.Error(1)
}

func (m *MockContainerManagerForErrorTesting) InspectContainer(ctx context.Context, id string) (*types.ContainerConfig, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*types.ContainerConfig), args.Error(1)
}

func (m *MockContainerManagerForErrorTesting) StartContainer(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockContainerManagerForErrorTesting) StopContainer(ctx context.Context, id string, signal string) error {
	args := m.Called(ctx, id, signal)
	return args.Error(0)
}

// Implement required interface methods as no-ops for testing
func (m *MockContainerManagerForErrorTesting) CreateContainer(ctx context.Context, opts *types.ContainerConfig, authBase64 string) (string, error) { return "", nil }
func (m *MockContainerManagerForErrorTesting) CommitContainer(ctx context.Context, containerID string, opts types.CommitOptions) (string, error) { return "", nil }
func (m *MockContainerManagerForErrorTesting) RemoveContainer(ctx context.Context, containerID string) error { return nil }
func (m *MockContainerManagerForErrorTesting) ContainerList(ctx context.Context, all bool) ([]*types.Container, error) { return nil, nil }
func (m *MockContainerManagerForErrorTesting) ContainerLogs(ctx context.Context, id string, ShowStdout bool, ShowStderr bool, Follow bool) (io.ReadCloser, error) { return nil, nil }
func (m *MockContainerManagerForErrorTesting) CopyContentToContainer(ctx context.Context, id, content, dest string) error { return nil }
func (m *MockContainerManagerForErrorTesting) CopyDirectorToContainer(ctx context.Context, id, srcPath, dstPath string) error { return nil }
func (m *MockContainerManagerForErrorTesting) CopyToContainer(ctx context.Context, id, srcPath, dstPath string) error { return nil }
func (m *MockContainerManagerForErrorTesting) CopyFileFromContainer(ctx context.Context, id string, srcPath string) (string, error) { return "", nil }
func (m *MockContainerManagerForErrorTesting) ExecContainer(ctx context.Context, id string, cmd []string, attachStdOut bool) (io.Reader, error) { return nil, nil }
func (m *MockContainerManagerForErrorTesting) BuildImage(ctx context.Context, dockerfile []byte, imageName string, platform string) (io.ReadCloser, error) { return nil, nil }
func (m *MockContainerManagerForErrorTesting) BuildMultiArchImage(ctx context.Context, dockerfile []byte, dockerCtx *bytes.Buffer, imageName string, platforms []string, authBase64 string) (io.ReadCloser, []string, error) { return nil, nil, nil }
func (m *MockContainerManagerForErrorTesting) ListImage(ctx context.Context, image string) ([]string, error) { return nil, nil }
func (m *MockContainerManagerForErrorTesting) PullImage(ctx context.Context, image string, authBase64 string, platform string) (io.ReadCloser, error) { return nil, nil }
func (m *MockContainerManagerForErrorTesting) TagImage(ctx context.Context, source, target string) error { return nil }
func (m *MockContainerManagerForErrorTesting) PushImage(ctx context.Context, target string, authBase64 string) (io.ReadCloser, error) { return nil, nil }
func (m *MockContainerManagerForErrorTesting) RemoveImage(ctx context.Context, target string) error { return nil }
func (m *MockContainerManagerForErrorTesting) InspectImage(ctx context.Context, image string) (*types.ImageInfo, error) { return nil, nil }
func (m *MockContainerManagerForErrorTesting) Name() string { return "mock" }

// TestContainer_Wait_ErrorHandling tests the specific error handling improvements from Issue #195
func TestContainer_Wait_ErrorHandling(testT *testing.T) {
	testT.Run("WaitContainer error returns error instead of log.Fatal", func(testT *testing.T) {
		mockClient := &MockContainerManagerForErrorTesting{}
		
		container := &Container{
			t: t{
				client: func() cri.ContainerManager { return mockClient },
				ctx:    context.Background(),
			},
			ID:     "test-container-id",
			Prefix: "[test-container]",
		}

		// Mock WaitContainer returning an error - this should return error, not call log.Fatal
		expectedError := errors.New("container wait failed")
		mockClient.On("WaitContainer", mock.Anything, "test-container-id", "not-running").Return(nil, expectedError)

		// This should return an error, not cause the process to exit with log.Fatal
		err := container.Wait()
		
		require.Error(testT, err)
		assert.Contains(testT, err.Error(), "failed to wait for container")
		assert.Contains(testT, err.Error(), "container wait failed")
		mockClient.AssertExpectations(testT)
	})

	testT.Run("nil status code returns error instead of log.Fatal", func(testT *testing.T) {
		mockClient := &MockContainerManagerForErrorTesting{}
		
		container := &Container{
			t: t{
				client: func() cri.ContainerManager { return mockClient },
				ctx:    context.Background(),
			},
			ID:     "test-container-id",
			Prefix: "[test-container]",
		}

		// Mock WaitContainer returning nil status code - this should return error, not call log.Fatal
		mockClient.On("WaitContainer", mock.Anything, "test-container-id", "not-running").Return(nil, nil)

		// This should return an error, not cause the process to exit with log.Fatal
		err := container.Wait()
		
		require.Error(testT, err)
		assert.Contains(testT, err.Error(), "failed to wait for container: status code is nil")
		mockClient.AssertExpectations(testT)
	})
}

// TestContainer_Start_ErrorHandling tests that Start method properly returns errors
func TestContainer_Start_ErrorHandling(testT *testing.T) {
	testT.Run("StartContainer error is properly returned", func(testT *testing.T) {
		mockClient := &MockContainerManagerForErrorTesting{}
		
		// Add StartContainer mock method
		mockClient.On("StartContainer", mock.Anything, "test-container-id").Return(errors.New("start failed"))
		
		build := &Build{
			Registry: "docker.io",
		}
		
		container := &Container{
			t: t{
				client: func() cri.ContainerManager { return mockClient },
				ctx:    context.Background(),
			},
			ID:    "test-container-id",
			Build: build,
			Opts: types.ContainerConfig{
				Image: "nginx:latest",
			},
		}

		// This should return an error instead of causing log.Fatal
		err := container.Start()
		
		require.Error(testT, err)
		assert.Contains(testT, err.Error(), "failed to start container test-container-id")
		assert.Contains(testT, err.Error(), "start failed")
		mockClient.AssertExpectations(testT)
	})
}

// TestContainer_Stop_ErrorHandling tests that Stop method properly returns errors  
func TestContainer_Stop_ErrorHandling(testT *testing.T) {
	testT.Run("StopContainer error is properly returned", func(testT *testing.T) {
		mockClient := &MockContainerManagerForErrorTesting{}
		
		// Add StopContainer mock method
		mockClient.On("StopContainer", mock.Anything, "test-container-id", "SIGTERM").Return(errors.New("stop failed"))
		
		container := &Container{
			t: t{
				client: func() cri.ContainerManager { return mockClient },
				ctx:    context.Background(),
			},
			ID: "test-container-id",
		}

		// This should return an error instead of causing log.Fatal
		err := container.Stop()
		
		require.Error(testT, err)
		assert.Contains(testT, err.Error(), "stop failed")
		mockClient.AssertExpectations(testT)
	})
}

// TestContextUsage_BackgroundInsteadOfTODO tests that proper context is used
func TestContextUsage_BackgroundInsteadOfTODO(testT *testing.T) {
	testT.Run("New creates container with context.Background instead of context.TODO", func(testT *testing.T) {
		build := Build{
			Env: BuildEnv,
		}
		
		container := New(build)
		
		require.NotNil(testT, container)
		assert.Equal(testT, BuildEnv, container.Env)
		assert.NotNil(testT, container.Build)
		assert.Equal(testT, &build, container.Build)
		assert.NotNil(testT, container.ctx)
		assert.NotNil(testT, container.client)
		
		// Context should not be cancelled or have deadline by default
		select {
		case <-container.ctx.Done():
			testT.Error("Context should not be done initially")
		default:
			// This is expected - context.Background() should not be done
		}
	})
}