package github

import (
	"fmt"
	"os/exec"
	"strings"
)

// execCommand is a variable for testing - allows mocking exec.Command
var execCommand = exec.Command

// HasUncommittedChanges returns true if there are uncommitted changes in the git repository
func HasUncommittedChanges() bool {
	cmd := execCommand("git", "status", "--porcelain")
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	return len(strings.TrimSpace(string(output))) > 0
}

// GetChangedFiles returns a list of changed files in the git repository
func GetChangedFiles() []string {
	cmd := execCommand("git", "status", "--porcelain")
	output, err := cmd.Output()
	if err != nil {
		return nil
	}

	outputStr := strings.TrimSuffix(string(output), "\n")
	if outputStr == "" {
		return nil
	}
	lines := strings.Split(outputStr, "\n")
	var files []string
	for _, line := range lines {
		if len(line) > 3 {
			// Format is "XY filename" where XY is status and filename starts at position 3
			files = append(files, strings.TrimSpace(line[3:]))
		}
	}
	return files
}

// GenerateFallbackMessage generates a commit message based on changed files
func GenerateFallbackMessage(files []string) string {
	if len(files) == 0 {
		return "chore: automated changes"
	}

	if len(files) == 1 {
		return fmt.Sprintf("chore: update %s", files[0])
	}

	return fmt.Sprintf("chore: update %d files", len(files))
}
