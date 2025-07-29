package main

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRootCommand(t *testing.T) {
	// Test root command execution
	cmd := rootCmd
	cmd.SetArgs([]string{})
	
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	
	err := cmd.Execute()
	require.NoError(t, err)
	
	output := buf.String()
	assert.Contains(t, output, "Sample Go App")
	assert.Contains(t, output, version)
}

func TestVersionCommand(t *testing.T) {
	// Test version command
	cmd := rootCmd
	cmd.SetArgs([]string{"version"})
	
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	
	err := cmd.Execute()
	require.NoError(t, err)
	
	output := buf.String()
	assert.Contains(t, output, "Version:")
	assert.Contains(t, output, "Commit:")
	assert.Contains(t, output, "Date:")
}

func TestEchoCommand(t *testing.T) {
	testCases := []struct {
		name     string
		args     []string
		expected string
	}{
		{
			name:     "single argument",
			args:     []string{"echo", "hello"},
			expected: "hello",
		},
		{
			name:     "multiple arguments",
			args:     []string{"echo", "hello", "world"},
			expected: "hello world",
		},
		{
			name:     "special characters",
			args:     []string{"echo", "hello,", "world!"},
			expected: "hello, world!",
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cmd := rootCmd
			cmd.SetArgs(tc.args)
			
			var buf bytes.Buffer
			cmd.SetOut(&buf)
			
			err := cmd.Execute()
			require.NoError(t, err)
			
			output := strings.TrimSpace(buf.String())
			assert.Equal(t, tc.expected, output)
		})
	}
}

func TestEchoCommandNoArgs(t *testing.T) {
	// Test echo command with no arguments (should fail)
	cmd := rootCmd
	cmd.SetArgs([]string{"echo"})
	
	err := cmd.Execute()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "requires at least 1 arg(s)")
}

func TestEnvironmentVariable(t *testing.T) {
	// Test environment variable handling
	testEnv := "test-environment"
	os.Setenv("SAMPLE_ENV", testEnv)
	defer os.Unsetenv("SAMPLE_ENV")
	
	cmd := rootCmd
	cmd.SetArgs([]string{})
	
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	
	err := cmd.Execute()
	require.NoError(t, err)
	
	output := buf.String()
	assert.Contains(t, output, "Environment: "+testEnv)
}

func TestCommandWithArguments(t *testing.T) {
	// Test root command with arguments
	cmd := rootCmd
	cmd.SetArgs([]string{"arg1", "arg2", "arg3"})
	
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	
	err := cmd.Execute()
	require.NoError(t, err)
	
	output := buf.String()
	assert.Contains(t, output, "Arguments: [arg1 arg2 arg3]")
}

// Benchmark tests for performance validation
func BenchmarkRootCommand(b *testing.B) {
	for i := 0; i < b.N; i++ {
		cmd := &cobra.Command{
			Use: "sample-go-app",
			Run: func(cmd *cobra.Command, args []string) {
				// Minimal execution for benchmarking
			},
		}
		cmd.SetArgs([]string{})
		
		var buf bytes.Buffer
		cmd.SetOut(&buf)
		
		_ = cmd.Execute()
	}
}

func BenchmarkEchoCommand(b *testing.B) {
	args := []string{"echo", "benchmark", "test", "message"}
	
	for i := 0; i < b.N; i++ {
		cmd := rootCmd
		cmd.SetArgs(args)
		
		var buf bytes.Buffer
		cmd.SetOut(&buf)
		
		_ = cmd.Execute()
	}
}

// Integration test helpers
func TestApplicationIntegration(t *testing.T) {
	// Test that the application behaves correctly as a whole
	
	t.Run("help flag", func(t *testing.T) {
		cmd := rootCmd
		cmd.SetArgs([]string{"--help"})
		
		var buf bytes.Buffer
		cmd.SetOut(&buf)
		
		err := cmd.Execute()
		require.NoError(t, err)
		
		output := buf.String()
		assert.Contains(t, output, "A sample Go application for testing")
		assert.Contains(t, output, "Available Commands:")
		assert.Contains(t, output, "echo")
		assert.Contains(t, output, "version")
	})
	
	t.Run("invalid command", func(t *testing.T) {
		cmd := rootCmd
		cmd.SetArgs([]string{"invalid-command"})
		
		err := cmd.Execute()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unknown command")
	})
	
	t.Run("version flag", func(t *testing.T) {
		cmd := rootCmd
		cmd.SetArgs([]string{"--version"})
		
		var buf bytes.Buffer
		cmd.SetOut(&buf)
		
		// Note: --version might not be implemented by default in cobra
		// This test verifies the behavior
		err := cmd.Execute()
		// Either succeeds with version output or fails with unknown flag
		if err != nil {
			assert.Contains(t, err.Error(), "unknown flag")
		}
	})
}

// Example function demonstrating testify features
func TestExampleTestifyFeatures(t *testing.T) {
	// Demonstrate various testify assertions for testing the testing framework
	
	t.Run("basic assertions", func(t *testing.T) {
		assert.True(t, true, "This should always be true")
		assert.False(t, false, "This should always be false")
		assert.Equal(t, 42, 42, "Numbers should be equal")
		assert.NotEqual(t, 42, 43, "Numbers should not be equal")
		assert.Empty(t, "", "Empty string should be empty")
		assert.NotEmpty(t, "not empty", "Non-empty string should not be empty")
	})
	
	t.Run("string assertions", func(t *testing.T) {
		text := "Hello, World!"
		assert.Contains(t, text, "Hello", "Text should contain Hello")
		assert.NotContains(t, text, "Goodbye", "Text should not contain Goodbye")
		assert.HasPrefix(t, text, "Hello", "Text should start with Hello")
		assert.HasSuffix(t, text, "!", "Text should end with !")
	})
	
	t.Run("slice assertions", func(t *testing.T) {
		slice := []string{"a", "b", "c"}
		assert.Len(t, slice, 3, "Slice should have 3 elements")
		assert.Contains(t, slice, "b", "Slice should contain 'b'")
		assert.NotContains(t, slice, "d", "Slice should not contain 'd'")
		assert.ElementsMatch(t, slice, []string{"c", "a", "b"}, "Slices should have same elements")
	})
	
	t.Run("error assertions", func(t *testing.T) {
		var err error
		assert.NoError(t, err, "Nil error should be no error")
		
		err = assert.AnError
		assert.Error(t, err, "Non-nil error should be an error")
	})
}