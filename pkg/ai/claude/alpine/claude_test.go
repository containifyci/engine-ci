package alpine

import (
	"testing"

	"github.com/containifyci/engine-ci/pkg/build"
	"github.com/containifyci/engine-ci/pkg/container"
	"github.com/containifyci/engine-ci/protos2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	t.Run("returns valid BuildStepv3", func(t *testing.T) {
		step := New()
		require.NotNil(t, step)
		assert.Implements(t, (*build.BuildStepv3)(nil), step)
	})

	t.Run("has correct name and alias", func(t *testing.T) {
		step := New()
		assert.Equal(t, "claude", step.Name())
		assert.Equal(t, "ai", step.Alias())
	})

	t.Run("is not async", func(t *testing.T) {
		step := New()
		assert.False(t, step.IsAsync())
	})

	t.Run("has AI build type", func(t *testing.T) {
		step := New()
		buildType := step.BuildType()
		require.NotNil(t, buildType)
		assert.Equal(t, container.AI, *buildType)
	})
}

func TestMatches(t *testing.T) {
	t.Run("matches AI build type", func(t *testing.T) {
		t.Setenv("claude-test-key", "claude-123")
		b := container.Build{
			BuildType: container.AI,
			Custom: map[string][]string{
				"claude_api_key": {"claude-test-key"},
			},
		}
		assert.True(t, Matches(b))
	})

	t.Run("does not match non-AI build type", func(t *testing.T) {
		b := container.Build{
			BuildType: container.GoLang,
		}
		assert.False(t, Matches(b))
	})

	t.Run("does not match empty build type", func(t *testing.T) {
		b := container.Build{}
		assert.False(t, Matches(b))
	})
}

func TestClaudeImage(t *testing.T) {
	t.Run("returns valid image URI", func(t *testing.T) {
		b := container.Build{
			ContainifyRegistry: "registry.example.com",
		}
		image := ClaudeImage(b)
		assert.NotEmpty(t, image)
		assert.Contains(t, image, "claude")
	})

	t.Run("includes registry in image URI", func(t *testing.T) {
		b := container.Build{
			ContainifyRegistry: "myregistry.com",
		}
		image := ClaudeImage(b)
		assert.Contains(t, image, "myregistry.com")
	})
}

func TestClaudeImages(t *testing.T) {
	t.Run("returns node and claude images", func(t *testing.T) {
		b := container.Build{
			ContainifyRegistry: "registry.example.com",
		}
		images := ClaudeImages(b)
		assert.Len(t, images, 2)
		assert.Contains(t, images, "node:22-alpine")
		assert.Contains(t, images[1], "claude")
	})

	t.Run("first image is node:22-alpine", func(t *testing.T) {
		b := container.Build{}
		images := ClaudeImages(b)
		assert.Equal(t, "node:22-alpine", images[0])
	})
}

func TestNewContainer(t *testing.T) {
	t.Run("creates container with default max iterations", func(t *testing.T) {
		b := container.Build{
			Custom: map[string][]string{},
		}
		c := newContainer(b)
		assert.NotNil(t, c)
		assert.Equal(t, 5, c.MaxIter)
	})

	t.Run("creates container with custom max iterations", func(t *testing.T) {
		b := container.Build{
			Custom: map[string][]string{
				"max_iterations": {"10"},
			},
		}
		c := newContainer(b)
		assert.Equal(t, 10, c.MaxIter)
	})

	t.Run("extracts ai_prompt from custom fields", func(t *testing.T) {
		b := container.Build{
			Custom: map[string][]string{
				"ai_prompt": {"test prompt"},
			},
		}
		c := newContainer(b)
		assert.Equal(t, "test prompt", c.Prompt)
	})

	t.Run("extracts ai_context from custom fields", func(t *testing.T) {
		b := container.Build{
			Custom: map[string][]string{
				"ai_context": {"test context"},
			},
		}
		c := newContainer(b)
		assert.Equal(t, "test context", c.Context)
	})

	t.Run("extracts agent_mode from custom fields", func(t *testing.T) {
		b := container.Build{
			Custom: map[string][]string{
				"agent_mode": {"true"},
			},
		}
		c := newContainer(b)
		assert.True(t, c.AgentMode)
	})

	t.Run("extracts folder from build", func(t *testing.T) {
		b := container.Build{
			Folder: "/test/folder",
			Custom: map[string][]string{},
		}
		c := newContainer(b)
		assert.Equal(t, "/test/folder", c.Folder)
	})

	t.Run("extracts ai_role from custom fields", func(t *testing.T) {
		b := container.Build{
			Custom: map[string][]string{
				"ai_role": {"docker_expert"},
			},
		}
		c := newContainer(b)
		assert.Equal(t, "docker_expert", c.Role)
	})

	t.Run("role is empty when not provided", func(t *testing.T) {
		b := container.Build{
			Custom: map[string][]string{},
		}
		c := newContainer(b)
		assert.Empty(t, c.Role)
	})
}

func TestDockerFile(t *testing.T) {
	t.Run("returns custom dockerfile if provided", func(t *testing.T) {
		b := &container.Build{
			ContainerFiles: map[string]*protos2.ContainerFile{
				"build": {
					Name:    "custom-claude",
					Content: "FROM custom:image",
				},
			},
		}
		df, err := dockerFile(b)
		require.NoError(t, err)
		assert.Equal(t, "FROM custom:image", df.Content)
	})

	t.Run("returns generated dockerfile if no custom provided", func(t *testing.T) {
		b := &container.Build{
			ContainerFiles: map[string]*protos2.ContainerFile{},
		}
		df, err := dockerFile(b)
		require.NoError(t, err)
		assert.NotEmpty(t, df.Content)
		assert.Contains(t, df.Name, "claude")
	})

	t.Run("returns generated dockerfile if build is nil", func(t *testing.T) {
		df, err := dockerFile(nil)
		require.NoError(t, err)
		assert.NotEmpty(t, df.Content)
		assert.Contains(t, df.Name, "claude")
	})

	t.Run("returns generated dockerfile if no build containerfile", func(t *testing.T) {
		b := &container.Build{
			ContainerFiles: map[string]*protos2.ContainerFile{
				"other": {
					Content: "test",
				},
			},
		}
		df, err := dockerFile(b)
		require.NoError(t, err)
		assert.NotEmpty(t, df.Content)
	})
}

func TestSetValue(t *testing.T) {
	t.Run("handles empty host", func(t *testing.T) {
		err := setValue("", "auth", "key", "value")
		assert.Error(t, err)
	})

	t.Run("handles empty auth", func(t *testing.T) {
		err := setValue("host", "", "key", "value")
		// Will fail due to network but won't panic
		assert.Error(t, err)
	})
}

func TestClaudeContainerMethods(t *testing.T) {
	t.Run("ClaudeContainer has required fields", func(t *testing.T) {
		b := container.Build{
			Custom: map[string][]string{
				"ai_prompt":      {"test prompt"},
				"ai_context":     {"test context"},
				"max_iterations": {"3"},
			},
		}
		c := newContainer(b)

		assert.NotNil(t, c.Container)
		assert.Equal(t, "test prompt", c.Prompt)
		assert.Equal(t, "test context", c.Context)
		assert.Equal(t, 3, c.MaxIter)
	})
}

func TestGetDockerfileMetadata(t *testing.T) {
	t.Run("returns metadata for empty variant", func(t *testing.T) {
		version, checksum, content := GetDockerfileMetadata("")
		assert.NotEmpty(t, version)
		assert.NotEmpty(t, checksum)
		assert.NotEmpty(t, content)
	})

	t.Run("content contains node image", func(t *testing.T) {
		_, _, content := GetDockerfileMetadata("")
		assert.Contains(t, content, "node")
	})
}

func TestGetRoleTemplate(t *testing.T) {
	t.Run("returns template for valid role", func(t *testing.T) {
		template := getRoleTemplate("build-reviewer")
		assert.NotEmpty(t, template)
		assert.Contains(t, template, "Build Reviewer Role")
	})

	t.Run("returns empty for unknown role", func(t *testing.T) {
		template := getRoleTemplate("unknown_role")
		assert.Empty(t, template)
	})

	t.Run("returns empty for empty role", func(t *testing.T) {
		template := getRoleTemplate("")
		assert.Empty(t, template)
	})
}

func TestRolesFS(t *testing.T) {
	t.Run("all predefined role files exist", func(t *testing.T) {
		expectedRoles := []string{"build-reviewer", "code-reviewer", "test-strategist", "code-simplifier"}
		for _, role := range expectedRoles {
			template := getRoleTemplate(role)
			assert.NotEmpty(t, template, "missing or empty role file: %s.md", role)
		}
	})
}
