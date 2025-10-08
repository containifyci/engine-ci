package copier

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/containifyci/engine-ci/pkg/container"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	step := New()
	assert.NotNil(t, step)
	assert.Equal(t, "copier", step.Name())
	assert.False(t, step.IsAsync())
}

func TestCopierImages(t *testing.T) {
	build := container.Build{}
	images := CopierImages(build)

	expected := []string{COPIER_IMAGE}
	assert.Equal(t, expected, images)
}

func TestMatches_WithCopierFile(t *testing.T) {
	// Create a temporary directory
	tempDir := t.TempDir()

	// Create a copier.yml file
	copierFile := filepath.Join(tempDir, COPIER_FILE)
	err := os.WriteFile(copierFile, []byte("# test copier file"), 0644)
	require.NoError(t, err)

	build := container.Build{
		Folder: tempDir,
	}

	assert.True(t, Matches(build))
}

func TestMatches_WithoutCopierFile(t *testing.T) {
	// Create a temporary directory without copier.yml
	tempDir := t.TempDir()

	build := container.Build{
		Folder: tempDir,
	}

	assert.False(t, Matches(build))
}

func TestMatches_EmptyFolder(t *testing.T) {
	build := container.Build{
		Folder: "", // Empty folder defaults to "./"
	}

	// This will check "./" + COPIER_FILE in the current working directory
	// Since we're in test context, it likely won't find one
	result := Matches(build)
	// We can't assert a specific value here since it depends on the current working directory
	// But we can verify the function doesn't crash
	assert.IsType(t, false, result) // Just check it returns a boolean
}

func TestExtractTemplateData(t *testing.T) {
	tests := []struct {
		custom   map[string][]string
		name     string
		expected []string
	}{
		{
			name: "all parameters provided",
			custom: map[string][]string{
				"data": {
					"service_name=test-service",
					"team=test-team",
					"domain=test-domain",
					"namespace=test-namespace",
					"business_domain_name=test-business-domain",
					"environments=shared,staging,production",
					"service_account=test-service@example.com",
					"pre_commit_entry=bash -c 'cd build/test-service && test'",
				},
			},
			expected: []string{
				"service_name=test-service",
				"team=test-team",
				"domain=test-domain",
				"namespace=test-namespace",
				"business_domain_name=test-business-domain",
				"environments=shared,staging,production",
				"service_account=test-service@example.com",
				"pre_commit_entry=bash -c 'cd build/test-service && test'",
			},
		},
		{
			name:     "no parameters provided",
			custom:   map[string][]string{},
			expected: []string{},
		},
		{
			name: "partial parameters provided",
			custom: map[string][]string{
				"data": {
					"service_name=test-service",
					"team=test-team",
				},
			},
			expected: []string{
				"service_name=test-service",
				"team=test-team",
			},
		},
		{
			name: "empty values",
			custom: map[string][]string{
				"data": {
					"service_name=",
					"team=",
				},
			},
			expected: []string{
				"service_name=",
				"team=",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			build := container.Build{
				Custom: tt.custom,
			}

			result := extractTemplateData(build)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractTemplatePath(t *testing.T) {
	tests := []struct {
		name     string
		custom   map[string][]string
		expected string
	}{
		{
			name: "template path provided",
			custom: map[string][]string{
				"template_path": {"/path/to/template"},
			},
			expected: "/path/to/template",
		},
		{
			name:     "no template path provided",
			custom:   map[string][]string{},
			expected: "",
		},
		{
			name: "empty template path",
			custom: map[string][]string{
				"template_path": {},
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			build := container.Build{
				Custom: tt.custom,
			}

			result := extractTemplatePath(build)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBuildCopierCommand(t *testing.T) {
	tests := []struct {
		name         string
		templateData []string
		templatePath string
		targetFolder string
		expected     []string
	}{
		{
			name: "full command with all parameters",
			templateData: []string{
				"service_name=test-service",
				"team=test-team",
				"domain=test-domain",
			},
			templatePath: "/path/to/template",
			targetFolder: "build/test-service",
			expected: []string{
				"copier", "copy",
				"--data", "service_name=test-service",
				"--data", "team=test-team",
				"--data", "domain=test-domain",
				"--defaults", "/path/to/template",
				"build/test-service",
			},
		},
		{
			name: "minimal command without template path",
			templateData: []string{
				"service_name=test-service",
			},
			templatePath: "",
			targetFolder: ".",
			expected: []string{
				"copier", "copy",
				"--data", "service_name=test-service",
				".",
			},
		},
		{
			name:         "no data parameters",
			templateData: []string{},
			templatePath: "/path/to/template",
			targetFolder: "target",
			expected: []string{
				"copier", "copy",
				"--defaults", "/path/to/template",
				"target",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			copierContainer := &CopierContainer{
				TemplateData: tt.templateData,
				TemplatePath: tt.templatePath,
				TargetFolder: tt.targetFolder,
			}

			result := copierContainer.buildCopierCommand()

			// Check that all expected elements are present
			for _, expected := range tt.expected {
				assert.Contains(t, result, expected, "Expected element %s not found in result %v", expected, result)
			}

			// Check the basic structure
			assert.Equal(t, "copier", result[0])
			assert.Equal(t, "copy", result[1])
			assert.Equal(t, tt.targetFolder, result[len(result)-1])
		})
	}
}

func TestNew_Constructor(t *testing.T) {
	build := container.Build{
		Folder: "./test",
		Custom: map[string][]string{
			"data": {
				"service_name=test-service",
			},
			"template_path": {"/path/to/template"},
			"output_path":   {"./test"},
		},
	}

	copierContainer := new(build)

	assert.NotNil(t, copierContainer.Container)
	assert.Equal(t, &build, copierContainer.Build)
	assert.Equal(t, "./test", copierContainer.SourceFolder)
	assert.Equal(t, "./test", copierContainer.TargetFolder)
	assert.Equal(t, "service_name=test-service", copierContainer.TemplateData[0])
	assert.Equal(t, "/path/to/template", copierContainer.TemplatePath)
}

func TestBuildCopierCommand_ParameterOrder(t *testing.T) {
	copierContainer := &CopierContainer{
		TemplateData: []string{
			"service_name=test-service",
			"team=test-team",
		},
		TemplatePath: "/template",
		TargetFolder: "target",
	}

	result := copierContainer.buildCopierCommand()

	// Verify the command structure
	assert.Equal(t, "copier", result[0])
	assert.Equal(t, "copy", result[1])

	// Find data parameters
	dataParams := []string{}
	for i := 2; i < len(result); i++ {
		if result[i] == "--data" && i+1 < len(result) {
			dataParams = append(dataParams, result[i+1])
		}
	}

	// Verify we have the expected data parameters
	assert.Len(t, dataParams, 2)
	assert.Contains(t, dataParams, "service_name=test-service")
	assert.Contains(t, dataParams, "team=test-team")

	// Verify defaults parameter
	defaultsIndex := -1
	for i, arg := range result {
		if arg == "--defaults" {
			defaultsIndex = i
			break
		}
	}
	assert.NotEqual(t, -1, defaultsIndex)
	assert.Equal(t, "/template", result[defaultsIndex+1])

	// Verify target is last
	assert.Equal(t, "target", result[len(result)-1])
}

func TestBuildCopierCommand_CommandString(t *testing.T) {
	copierContainer := &CopierContainer{
		TemplateData: []string{
			"service_name=some-service",
			"team=some-team",
		},
		TemplatePath: "/path/to/template",
		TargetFolder: "build/some-service",
	}

	result := copierContainer.buildCopierCommand()
	cmdString := strings.Join(result, " ")

	// Verify the command contains expected patterns
	assert.Contains(t, cmdString, "copier copy")
	assert.Contains(t, cmdString, "--data service_name=some-service")
	assert.Contains(t, cmdString, "--data team=some-team")
	assert.Contains(t, cmdString, "--defaults /path/to/template")
	assert.Contains(t, cmdString, "build/some-service")
}
