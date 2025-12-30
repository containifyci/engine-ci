package build

import (
	"testing"

	"github.com/containifyci/engine-ci/protos2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//nolint:testparallel // use t.Setenv which is not compatible with t.Parallel
func TestNewAIBuild(t *testing.T) {
	t.Run("creates AI build with name", func(t *testing.T) {
		build := NewAIBuild("test-ai")
		require.NotNil(t, build)
		assert.Equal(t, "test-ai", build.Application)
		assert.Equal(t, protos2.BuildType_AI, build.BuildType)
	})

	t.Run("uses COMMIT_SHA for image tag", func(t *testing.T) {
		t.Setenv("COMMIT_SHA", "def456")

		build := NewAIBuild("test-ai")
		assert.Equal(t, "def456", build.ImageTag)
	})

	t.Run("uses local when COMMIT_SHA not set", func(t *testing.T) {
		build := NewAIBuild("test-ai")
		assert.Equal(t, "local", build.ImageTag)
	})

	t.Run("uses correct environment", func(t *testing.T) {
		t.Setenv("ENV", "local")

		build := NewAIBuild("test-ai")
		assert.Equal(t, protos2.EnvType_local, build.Environment)
	})
}

func TestNewClaudeBuild(t *testing.T) {
	t.Parallel()
	t.Run("creates Claude build with prompt", func(t *testing.T) {
		t.Parallel()
		prompt := "Fix the bug in main.go"
		build := NewClaudeBuild(prompt)
		require.NotNil(t, build)
		assert.Equal(t, "claude-agent", build.Application)
		assert.Equal(t, protos2.BuildType_AI, build.BuildType)
	})

	t.Run("includes prompt in properties", func(t *testing.T) {
		t.Parallel()
		prompt := "Write tests for package"
		build := NewClaudeBuild(prompt)
		require.NotNil(t, build.Properties)
		assert.Contains(t, build.Properties, "ai_prompt")
		promptList := build.Properties["ai_prompt"]
		require.NotNil(t, promptList)
		assert.Len(t, promptList.Values, 1)
		assert.Equal(t, prompt, promptList.Values[0].GetStringValue())
	})

	t.Run("handles empty prompt", func(t *testing.T) {
		t.Parallel()
		build := NewClaudeBuild("")
		require.NotNil(t, build)
		assert.Contains(t, build.Properties, "ai_prompt")
	})

	t.Run("handles multi-line prompt", func(t *testing.T) {
		t.Parallel()
		prompt := "Line 1\nLine 2\nLine 3"
		build := NewClaudeBuild(prompt)
		promptList := build.Properties["ai_prompt"]
		assert.Equal(t, prompt, promptList.Values[0].GetStringValue())
	})
}

func TestNewAgentBuild(t *testing.T) {
	t.Parallel()
	t.Run("creates agent build with prompt", func(t *testing.T) {
		t.Parallel()
		prompt := "Implement feature X"
		build := NewAgentBuild(prompt, 5)
		require.NotNil(t, build)
		assert.Equal(t, "claude-agent", build.Application)
		assert.Equal(t, protos2.BuildType_AI, build.BuildType)
	})

	t.Run("sets agent_mode to true", func(t *testing.T) {
		t.Parallel()
		build := NewAgentBuild("test prompt", 3)
		require.NotNil(t, build.Properties)
		assert.Contains(t, build.Properties, "agent_mode")
		agentMode := build.Properties["agent_mode"]
		require.NotNil(t, agentMode)
		assert.Equal(t, "true", agentMode.Values[0].GetStringValue())
	})

	t.Run("sets max_iterations when positive", func(t *testing.T) {
		t.Parallel()
		build := NewAgentBuild("test prompt", 10)
		require.NotNil(t, build.Properties)
		assert.Contains(t, build.Properties, "max_iterations")
		maxIter := build.Properties["max_iterations"]
		require.NotNil(t, maxIter)
		assert.Equal(t, "10", maxIter.Values[0].GetStringValue())
	})

	t.Run("does not set max_iterations when zero", func(t *testing.T) {
		t.Parallel()
		build := NewAgentBuild("test prompt", 0)
		require.NotNil(t, build.Properties)
		assert.NotContains(t, build.Properties, "max_iterations")
	})

	t.Run("does not set max_iterations when negative", func(t *testing.T) {
		t.Parallel()
		build := NewAgentBuild("test prompt", -1)
		require.NotNil(t, build.Properties)
		assert.NotContains(t, build.Properties, "max_iterations")
	})

	t.Run("includes prompt in properties", func(t *testing.T) {
		t.Parallel()
		prompt := "Complex multi-step task"
		build := NewAgentBuild(prompt, 5)
		assert.Contains(t, build.Properties, "ai_prompt")
		promptList := build.Properties["ai_prompt"]
		assert.Equal(t, prompt, promptList.Values[0].GetStringValue())
	})
}
