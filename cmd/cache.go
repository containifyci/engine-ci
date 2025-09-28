package cmd

import (
	_ "embed"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"sync"

	"github.com/containifyci/engine-ci/pkg/cri/utils"
	"github.com/spf13/cobra"
)

// buildCmd represents the build command
var cacheCmd = &cobra.Command{
	Use:   "cache",
	Short: "Command to generate Github Action file for gflip",
	Long: `Command to generate Github Action file for gflip.
`,
}

var cacheSaveCmd = &cobra.Command{
	Use:   "save",
	Short: "Command to generate Github Action file for gflip",
	Long: `Command to generate Github Action file for gflip.
`,
	Run: func(cmd *cobra.Command, args []string) {
		err := SaveCache()
		if err != nil {
			slog.Error("Error saving cache", "error", err)
			os.Exit(1)
		}
	},
}

var cacheLoadCmd = &cobra.Command{
	Use:   "load",
	Short: "Command to generate Github Action file for gflip",
	Long: `Command to generate Github Action file for gflip.
`,
	Run: func(cmd *cobra.Command, args []string) {
		err := LoadCache()
		if err != nil {
			slog.Error("Error loading cache", "error", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(cacheCmd)
	cacheCmd.AddCommand(cacheSaveCmd)
	cacheCmd.AddCommand(cacheLoadCmd)
}

func SaveCache() error {
	args := GetBuild(false) // Use plugin system for cache operations
	_, bs := Pre(args[0].Builds[0], nil)
	images := bs.Images()
	if len(images) == 0 {
		return nil
	}

	// Channel to collect errors
	errs := make(chan error, 10)

	var wg sync.WaitGroup

	// Run each command in a separate goroutine
	for _, image := range images {
		wg.Add(1)
		info, err := utils.ParseDockerImage(image)
		if err != nil {
			slog.Error("Error parsing image", "error", err)
			os.Exit(1)
		}
		cmd := fmt.Sprintf(`
set -x
docker pull %s
docker save -o ~/image-cache/%s.tar %s
`, image, info.Image, image)
		go runCommand(&wg, errs, "sh", []string{"-c", cmd}...)
	}

	// Wait for all commands to complete
	wg.Wait()
	close(errs)
	// Check for any errors
	for err := range errs {
		// errors.
		slog.Warn("Error pull image", "error", err)
	}
	return nil
}

func LoadCache() error {
	args := GetBuild(false) // Use plugin system for cache operations
	arg, bs := Pre(args[0].Builds[0], nil)

	images := bs.Images()
	if len(images) == 0 {
		return nil
	}

	// Channel to collect errors
	errs := make(chan error, 10)

	var wg sync.WaitGroup

	// Run each command in a separate goroutine
	for _, image := range images {
		wg.Add(1)
		info, err := utils.ParseDockerImage(image)
		if err != nil {
			slog.Error("Error parsing image", "error", err)
			os.Exit(1)
		}
		// nolint:staticcheck
		if arg.Runtime == utils.Docker {
			cmd := fmt.Sprintf(`
set -x
docker load -i ~/image-cache/%s.tar
`, info.Image)
			go runCommand(&wg, errs, "sh", []string{"-c", cmd}...)
		} else if arg.Runtime == utils.Podman {
			cmd := fmt.Sprintf(`
set -x
podman load -i ~/image-cache/%s.tar
`, info.Image)
			runCommand(&wg, errs, "sh", []string{"-c", cmd}...)
		}
	}

	// Wait for all commands to complete
	wg.Wait()
	close(errs)
	// Check for any errors
	for err := range errs {
		// errors.
		slog.Warn("Error loading image", "error", err)
	}
	return nil
}

func runCommand(wg *sync.WaitGroup, errors chan<- error, cmd string, args ...string) {
	defer wg.Done()

	// Create the command
	command := exec.Command(cmd, args...)

	command.Stdout = os.Stdout
	command.Stderr = os.Stderr

	// Run the command
	err := command.Run()
	if err != nil {
		errors <- fmt.Errorf("error running command: error %s, cmd %s, args %s", err, cmd, args)
	}
}
