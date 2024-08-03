package critest

import (
	"bytes"
	"context"
	"io"
	"testing"

	"github.com/containifyci/engine-ci/pkg/cri/types"
	"github.com/stretchr/testify/assert"
)

func TestMockContainer_ContainerLifecycle(t *testing.T) {
	ctx := context.TODO()
	// Create a new MockContainerManager
	m, _ := NewMockContainerManager()

	// Create a new container configuration
	containerConfig := &types.ContainerConfig{
		Name: "test-container",
		Image: "test-image",
	}

	// Create a new container
	containerID, err := m.CreateContainer(ctx, containerConfig, "")
	assert.NoError(t, err)
	assert.Equal(t, "created", m.GetContainer(containerID).State)

	cons, err := m.ContainerList(ctx, true)
	assert.NoError(t, err)
	assert.NotNil(t, cons)
	assert.Len(t, cons, 1)
	assert.Equal(t, containerID, cons[0].ID)

	// Start the container
	err = m.StartContainer(ctx, containerID)
	assert.NoError(t, err)
	assert.Equal(t, "started", m.GetContainer(containerID).State)

	// Stop the container
	err = m.StopContainer(ctx, containerID, "")
	assert.NoError(t, err)
	assert.Equal(t, "stopped", m.GetContainer(containerID).State)

	// Commit the container
	_, err = m.CommitContainer(ctx, containerID, types.CommitOptions{})
	assert.NoError(t, err)

	cnt, err := m.ContainerLogs(ctx, containerID, true, true, true)
	assert.NoError(t, err)
	assert.NotNil(t, cnt)
	var b bytes.Buffer
	_, err = io.Copy(&b, cnt)
	assert.NoError(t, err)
	assert.NotEmpty(t, b.String())
	assert.Equal(t, "container starting\ncontainer running\ncontainer stopped", b.String())

	// Remove the container
	err = m.RemoveContainer(ctx, containerID)
	assert.NoError(t, err)

	cons, err = m.ContainerList(ctx, true)
	assert.NoError(t, err)
	assert.Nil(t, cons)
	assert.Len(t, cons, 0)
}

func TestMockContainer_ImageLifecycle(t *testing.T) {
	ctx := context.TODO()
	// Create a new MockContainerManager
	m, _ := NewMockContainerManager()

	cnt, err := m.BuildImage(ctx, []byte("TestDockerFiles"),"test-image", "")
	assert.NoError(t, err)
	assert.NotNil(t, cnt)
	var b bytes.Buffer
	_, err = io.Copy(&b, cnt)
	assert.NoError(t, err)
	assert.NotEmpty(t, b.String())
	assert.Equal(t, "mock_build_output", b.String())

	imgs, err := m.ListImage(ctx, "test-image")
	assert.NoError(t, err)
	assert.NotNil(t, imgs)
	assert.Len(t, imgs, 1)
	assert.Equal(t, "test-image", imgs[0])

	cnt, err = m.PullImage(ctx, "test-pull-image", "", "")
	assert.NoError(t, err)
	assert.NotNil(t, cnt)
	b.Reset()
	_, err = io.Copy(&b, cnt)
	assert.NoError(t, err)
	assert.NotEmpty(t, b.String())
	assert.Equal(t, "test-pull-image pulled", b.String())
	assert.Len(t, m.Images, 2)

	imgInfo, err := m.InspectImage(ctx, "test-image")
	assert.NoError(t, err)
	assert.NotNil(t, imgInfo)

	err = m.RemoveImage(ctx, "test-pull-image")
	assert.NoError(t, err)
	assert.Len(t, m.Images, 1)

	err = m.TagImage(ctx, "test-image", "test-tag-image")
	assert.NoError(t, err)
	assert.Len(t, m.Images, 2)
}
