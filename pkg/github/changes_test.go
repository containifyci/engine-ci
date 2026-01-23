package github

import (
	"os"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
)

// mockExecCommand creates a mock exec.Command that runs a test helper
func mockExecCommand(output string, exitCode int) func(string, ...string) *exec.Cmd {
	return func(name string, args ...string) *exec.Cmd {
		cs := []string{"-test.run=TestHelperProcess", "--", name}
		cs = append(cs, args...)
		cmd := exec.Command(os.Args[0], cs...)
		cmd.Env = []string{
			"GO_WANT_HELPER_PROCESS=1",
			"MOCK_OUTPUT=" + output,
			"MOCK_EXIT_CODE=" + string(rune('0'+exitCode)),
		}
		return cmd
	}
}

// TestHelperProcess is a test helper that is invoked as a subprocess
// It outputs the MOCK_OUTPUT environment variable and exits with MOCK_EXIT_CODE
func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	output := os.Getenv("MOCK_OUTPUT")
	exitCode := os.Getenv("MOCK_EXIT_CODE")

	if output != "" {
		os.Stdout.WriteString(output)
	}

	if exitCode != "0" {
		os.Exit(1)
	}
	os.Exit(0)
}

func TestHasUncommittedChanges(t *testing.T) {
	// Save original and restore after test
	originalExecCommand := execCommand
	defer func() { execCommand = originalExecCommand }()

	t.Run("returns true when output is non-empty", func(t *testing.T) {
		execCommand = mockExecCommand("?? untracked.txt\n", 0)
		assert.True(t, HasUncommittedChanges())
	})

	t.Run("returns true for staged files", func(t *testing.T) {
		execCommand = mockExecCommand("A  staged.txt\n", 0)
		assert.True(t, HasUncommittedChanges())
	})

	t.Run("returns true for modified files", func(t *testing.T) {
		execCommand = mockExecCommand(" M modified.txt\n", 0)
		assert.True(t, HasUncommittedChanges())
	})

	t.Run("returns false when output is empty", func(t *testing.T) {
		execCommand = mockExecCommand("", 0)
		assert.False(t, HasUncommittedChanges())
	})

	t.Run("returns false when output is whitespace only", func(t *testing.T) {
		execCommand = mockExecCommand("   \n", 0)
		assert.False(t, HasUncommittedChanges())
	})

	t.Run("returns false when command fails", func(t *testing.T) {
		execCommand = mockExecCommand("", 1)
		assert.False(t, HasUncommittedChanges())
	})
}

func TestGetChangedFiles(t *testing.T) {
	// Save original and restore after test
	originalExecCommand := execCommand
	defer func() { execCommand = originalExecCommand }()

	t.Run("returns nil when output is empty", func(t *testing.T) {
		execCommand = mockExecCommand("", 0)
		files := GetChangedFiles()
		assert.Nil(t, files)
	})

	t.Run("returns nil when command fails", func(t *testing.T) {
		execCommand = mockExecCommand("", 1)
		files := GetChangedFiles()
		assert.Nil(t, files)
	})

	t.Run("returns untracked files", func(t *testing.T) {
		execCommand = mockExecCommand("?? file1.txt\n?? file2.txt", 0)
		files := GetChangedFiles()
		assert.Len(t, files, 2)
		assert.Contains(t, files, "file1.txt")
		assert.Contains(t, files, "file2.txt")
	})

	t.Run("returns staged files", func(t *testing.T) {
		execCommand = mockExecCommand("A  staged.txt", 0)
		files := GetChangedFiles()
		assert.Len(t, files, 1)
		assert.Contains(t, files, "staged.txt")
	})

	t.Run("returns modified files", func(t *testing.T) {
		execCommand = mockExecCommand(" M modified.txt", 0)
		files := GetChangedFiles()
		assert.Len(t, files, 1)
		assert.Contains(t, files, "modified.txt")
	})

	t.Run("handles files in subdirectories", func(t *testing.T) {
		execCommand = mockExecCommand("?? subdir/nested.txt", 0)
		files := GetChangedFiles()
		assert.Len(t, files, 1)
		assert.Contains(t, files, "subdir/nested.txt")
	})

	t.Run("handles mixed status types", func(t *testing.T) {
		execCommand = mockExecCommand("A  added.txt\n M modified.txt\n?? untracked.txt", 0)
		files := GetChangedFiles()
		assert.Len(t, files, 3)
		assert.Contains(t, files, "added.txt")
		assert.Contains(t, files, "modified.txt")
		assert.Contains(t, files, "untracked.txt")
	})

	t.Run("skips lines shorter than 3 characters", func(t *testing.T) {
		// Edge case: lines that are too short to have a filename
		execCommand = mockExecCommand("?? valid.txt\nAB\n", 0)
		files := GetChangedFiles()
		assert.Len(t, files, 1)
		assert.Contains(t, files, "valid.txt")
	})

	t.Run("handles renamed files", func(t *testing.T) {
		// Git shows renamed files as "R  old -> new"
		execCommand = mockExecCommand("R  old.txt -> new.txt", 0)
		files := GetChangedFiles()
		assert.Len(t, files, 1)
		assert.Contains(t, files, "old.txt -> new.txt")
	})
}

func TestGenerateFallbackMessage(t *testing.T) {
	tests := []struct {
		name  string
		want  string
		files []string
	}{
		{
			name:  "no files",
			files: []string{},
			want:  "chore: automated changes",
		},
		{
			name:  "nil files",
			files: nil,
			want:  "chore: automated changes",
		},
		{
			name:  "single file",
			files: []string{"main.go"},
			want:  "chore: update main.go",
		},
		{
			name:  "multiple files",
			files: []string{"main.go", "go.mod", "README.md"},
			want:  "chore: update 3 files",
		},
		{
			name:  "two files",
			files: []string{"a.go", "b.go"},
			want:  "chore: update 2 files",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GenerateFallbackMessage(tt.files)
			assert.Equal(t, tt.want, got)
		})
	}
}
