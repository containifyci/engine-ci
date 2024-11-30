package critest

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"strings"

	"github.com/containifyci/engine-ci/pkg/cri/types"
)

// Error types for better testing simulation
var (
	ErrContainerNotFound = errors.New("container not found")
	ErrImageNotFound     = errors.New("image not found")
)

// MockContainerManager is a mock implementation of the ContainerManager interface.
type MockContainerManager struct {
	Containers           map[string]*MockContainerLifecycle
	ContainerLogsEntries map[string][]string
	Images               map[string]*MockImageLifecycle
	Errors               map[string]error
	ImagesLogEntries     []string
}

type MockContainerLifecycle struct {
	Opts   *types.ContainerConfig
	Volume *MockContainerVolume
	ID     string
	State  string
}

type MockContainerVolume struct {
	Content string
	SrcPath string
	DstPath string
}

type MockImageLifecycle struct {
	ID        string
	Opts      *types.ImageInfo
	BuildInfo MockImageBuildInfo
}

type MockImageBuildInfo struct {
	ID         string
	Name       string
	Dockerfile []byte
}

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

// randString generates a random string of length n.
func randString(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}

func NewMockContainerManager() (*MockContainerManager, error) {
	return &MockContainerManager{
		Containers:           make(map[string]*MockContainerLifecycle),
		ContainerLogsEntries: make(map[string][]string),
		Images:               make(map[string]*MockImageLifecycle),
		ImagesLogEntries:     []string{},
		Errors:               make(map[string]error),
	}, nil
}

func (m *MockContainerManager) Reset() {
	m.Containers = make(map[string]*MockContainerLifecycle)
	m.ContainerLogsEntries = make(map[string][]string)
	m.Images = make(map[string]*MockImageLifecycle)
	m.ImagesLogEntries = []string{}
	m.Errors = make(map[string]error)
}

func (m *MockContainerManager) GetContainerByImage(image string) *MockContainerLifecycle {
	for _, con := range m.Containers {
		if con.Opts.Image == image {
			return con
		}
	}
	return nil
}

func (m *MockContainerManager) GetContainer(id string) *MockContainerLifecycle {
	return m.Containers[id]
}

func (m *MockContainerManager) GetImage(id string) *MockImageLifecycle {
	return m.Images[id]
}

func (m *MockContainerManager) CreateContainer(ctx context.Context, opts *types.ContainerConfig, authBase64 string) (string, error) {
	id := randString(6)
	m.Containers[id] = &MockContainerLifecycle{ID: id, Opts: opts, State: "created"}
	return id, nil
}

func (m *MockContainerManager) StartContainer(ctx context.Context, id string) error {
	if _, exists := m.Containers[id]; !exists {
		return ErrContainerNotFound
	}
	m.Containers[id].State = "started"
	m.ContainerLogsEntries[m.Containers[id].Opts.Image] = []string{"container starting", "container running"}
	return nil
}

func (m *MockContainerManager) StopContainer(ctx context.Context, id string, signal string) error {
	if _, exists := m.Containers[id]; !exists {
		return ErrContainerNotFound
	}
	m.ContainerLogsEntries[m.Containers[id].Opts.Image] = append(m.ContainerLogsEntries[m.Containers[id].Opts.Image], "container stopped")
	m.Containers[id].State = "stopped"
	return nil
}

func (m *MockContainerManager) CommitContainer(ctx context.Context, containerID string, opts types.CommitOptions) (string, error) {
	imageID := containerID
	return imageID, nil
}

func (m *MockContainerManager) RemoveContainer(ctx context.Context, containerID string) error {
	delete(m.Containers, containerID)
	return nil
}

func (m *MockContainerManager) ContainerList(ctx context.Context, all bool) ([]*types.Container, error) {
	var containerList []*types.Container
	for id, con := range m.Containers {
		containerList = append(containerList, &types.Container{ID: id, Image: con.Opts.Image, ImageID: id, Names: []string{con.Opts.Name}})
	}
	return containerList, nil
}

func (m *MockContainerManager) ContainerLogs(ctx context.Context, id string, showStdout bool, showStderr bool, follow bool) (io.ReadCloser, error) {
	con, exists := m.Containers[id]
	if !exists {
		return nil, ErrContainerNotFound
	}

	logs, exists := m.ContainerLogsEntries[con.Opts.Image]
	if exists {
		return io.NopCloser(strings.NewReader(strings.Join(logs, "\n"))), nil
	}
	return nil, ErrContainerNotFound
}

func (m *MockContainerManager) CopyContentToContainer(ctx context.Context, id, content, dest string) error {
	m.Containers[id].Volume = &MockContainerVolume{Content: content, DstPath: dest}
	return nil
}

func (m *MockContainerManager) CopyDirectorToContainer(ctx context.Context, id, srcPath, dstPath string) error {
	m.Containers[id].Volume = &MockContainerVolume{SrcPath: srcPath, DstPath: dstPath}
	return nil
}

func (m *MockContainerManager) CopyToContainer(ctx context.Context, id, srcPath, dstPath string) error {
	m.Containers[id].Volume = &MockContainerVolume{SrcPath: srcPath, DstPath: dstPath}
	return nil
}

func (m *MockContainerManager) CopyFileFromContainer(ctx context.Context, id string, srcPath string) (string, error) {
	return m.Containers[id].Volume.Content, nil
}

func (m *MockContainerManager) ExecContainer(ctx context.Context, id string, cmd []string, attachStdOut bool) (io.Reader, error) {
	return strings.NewReader("mock_exec_output"), nil
}

func (m *MockContainerManager) InspectContainer(ctx context.Context, id string) (*types.ContainerConfig, error) {
	if container, exists := m.Containers[id]; exists {
		return container.Opts, nil
	}
	return nil, ErrContainerNotFound
}

func (m *MockContainerManager) WaitContainer(ctx context.Context, id string, waitCondition string) (*int64, error) {
	exitCode := int64(0)
	return &exitCode, nil
}

func (m *MockContainerManager) BuildImage(ctx context.Context, dockerfile []byte, imageName string, platform string) (io.ReadCloser, error) {
	id := randString(6)
	platformSpec := types.ParsePlatform(platform)
	m.Images[imageName] = &MockImageLifecycle{ID: id, Opts: &types.ImageInfo{ID: id, Platform: platformSpec}, BuildInfo: MockImageBuildInfo{ID: id, Name: imageName, Dockerfile: dockerfile}}
	return io.NopCloser(strings.NewReader("mock_build_output")), nil
}

func (m *MockContainerManager) BuildMultiArchImage(ctx context.Context, dockerfile []byte, dockerCtx *bytes.Buffer, imageName string, platforms []string, authBase64 string) (io.ReadCloser, []string, error) {
	for _, platform := range platforms {
		_, err := m.BuildImage(ctx, dockerfile, imageName + "-" + platform, platform)
		if err != nil {
			return nil, nil, err
		}
	}

	return io.NopCloser(strings.NewReader("mock_multiarch_build_output")), platforms, nil
}

func (m *MockContainerManager) ListImage(ctx context.Context, image string) ([]string, error) {
	images := []string{}
	for _, img := range m.Images {
		if img.BuildInfo.Name == image {
			images = append(images, img.BuildInfo.Name)
		}
	}
	return images, nil
}

func (m *MockContainerManager) PullImage(ctx context.Context, image string, authBase64 string, platform string) (io.ReadCloser, error) {
	if _, exists := m.Errors[image]; exists {
		return nil, m.Errors[image]
	}

	id := randString(6)
	m.Images[image] = &MockImageLifecycle{ID: id, Opts: &types.ImageInfo{ID: id}}
	m.ImagesLogEntries = append(m.ImagesLogEntries, fmt.Sprintf("%s pulled", image))
	return io.NopCloser(strings.NewReader(fmt.Sprintf("%s pulled", image))), nil
}

func (m *MockContainerManager) TagImage(ctx context.Context, source, target string) error {
	if img, exists := m.Images[source]; exists {
		m.Images[target] = img
	}
	return nil
}

func (m *MockContainerManager) PushImage(ctx context.Context, target string, authBase64 string) (io.ReadCloser, error) {
	return io.NopCloser(strings.NewReader("mock_push_output")), nil
}

func (m *MockContainerManager) RemoveImage(ctx context.Context, target string) error {
	delete(m.Images, target)
	return nil
}

func (m *MockContainerManager) InspectImage(ctx context.Context, image string) (*types.ImageInfo, error) {
	if img, exists := m.Images[image]; exists {
		return img.Opts, nil
	}
	return nil, ErrImageNotFound
}

func (m *MockContainerManager) Name() string {
	return "MockContainerManager"
}
