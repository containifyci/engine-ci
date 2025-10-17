package dummy

import (
	"testing"

	"github.com/containifyci/engine-ci/pkg/container"
	"github.com/stretchr/testify/assert"
)

func TestDummyBuildStep(t *testing.T) {
	step := New()

	assert.Equal(t, "dummy", step.Name())
	assert.True(t, step.Matches(container.Build{}))
	assert.False(t, step.IsAsync())
	err := step.RunWithBuild(container.Build{
		Image: "dummy-image",
	})
	assert.NoError(t, err)
}
