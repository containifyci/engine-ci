package cmd

import (
	_ "embed"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/containifyci/engine-ci/pkg/container"
	"github.com/containifyci/engine-ci/pkg/cri"
	"github.com/containifyci/engine-ci/pkg/cri/utils"
	"github.com/spf13/cobra"
)

// buildCmd represents the build command
var githubActionCmd = &cobra.Command{
	Use:   "github_actions",
	Short: "Command to generate Github Action file for gflip",
	Long: `Command to generate Github Action file for gflip.
`,
	RunE: RunGithubActionCmd,
}

func init() {
	rootCmd.AddCommand(githubActionCmd)
}

func RunGithubActionCmd(_ *cobra.Command, args []string) error {
	return RunGithubAction()
}

func RunGithubAction() error {

	// Define the script content
	var scriptContent string
	switch cri.DetectContainerRuntime() {
	case utils.Docker:
		slog.Info("Docker runtime detected")
		scriptContent = `#!/bin/bash
set -xe
sudo mv /etc/subgid /etc/subgid.orig
sudo mv /etc/subuid /etc/subuid.orig
echo "runneradmin:100000:65536" | sudo tee -a /etc/subgid
echo "runner:$(id -g runner):65536" | sudo tee -a /etc/subgid
echo "runneradmin:100000:65536" | sudo tee -a /etc/subuid
echo "runner:$(id -u runner):65536" | sudo tee -a /etc/subuid
# Check if the JSON file is empty or does not exist
if [ ! -s "/etc/docker/daemon.json" ]; then
  # If the file is empty or does not exist, initialize it with an array containing the new value
  echo "{\"userns-remap\": \"runner\"}" | sudo tee "/etc/docker/daemon.json"
else
	cat /etc/docker/daemon.json | sudo jq --arg value "runner" --arg key "userns-remap" '.[$key] = ($key as $k | (try $value))' | sudo tee "/etc/docker/daemon.json"
fi
sudo cat /etc/docker/daemon.json
# echo "{\"userns-remap\": \"runner\"}" | sudo tee -a /etc/docker/daemon.json
sudo systemctl restart docker
echo "CONTAINER_PRIVILGED=false" >> $GITHUB_ENV
`
	case utils.Podman:
		slog.Info("Podman runtime detected, Enable user podman socket")
		scriptContent = `#!/bin/bash
set -xe
systemctl --user restart podman.socket
ls -lha  /run/user/1001/podman/podman.sock
echo "CONTAINER_PRIVILGED=false" >> $GITHUB_ENV
`
	default:
		slog.Error("Unknown runtime", "runtime", container.GetBuild().Runtime)
		return fmt.Errorf("unknown runtime: %s", container.GetBuild().Runtime)
	}

	// Write the script to a temporary file
	scriptFile, err := writeTempScript(scriptContent)

	if err != nil {
		slog.Error("Error writing script.", "error", err)
		return err
	}

	command := exec.Command("bash", scriptFile)

	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	return command.Run()
}

func writeTempScript(content string) (string, error) {
	tempDir, err := os.MkdirTemp(os.TempDir(), "scripts")
	if err != nil {
		return "", err
	}

	// Define the script file name within the temp directory
	scriptFile := filepath.Join(tempDir, "script.sh")

	// Write the script content to the file
	err = os.WriteFile(scriptFile, []byte(content), 0755)
	if err != nil {
		return "", err
	}

	return scriptFile, nil
}
