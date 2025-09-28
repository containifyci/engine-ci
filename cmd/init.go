package cmd

import (
	"bytes"
	_ "embed"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"text/template"

	"github.com/containifyci/engine-ci/pkg/autodiscovery"
	"github.com/containifyci/engine-ci/pkg/container"
	"github.com/spf13/cobra"
)

// TemplateData holds the data passed to the containifyci.go template
type TemplateData struct {
	Groups container.BuildGroups
}

//go:embed containifyci.go.tmpl
var mage []byte

// initCmd represents the init command
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Command to generate containifyci.go file for containifyci usage",
	Long:  `Command to generate containifyci.go file for containifyci usage. Use --auto to generate based on auto-discovered Go projects.`,
	RunE:  RunInit,
}

func init() {
	rootCmd.AddCommand(initCmd)
	initCmd.Flags().BoolP("auto", "a", false, "Auto-discover Go projects and generate configuration")
}

// createContainifyCIFileWithProjects creates containifyci.go file using template with build groups
func createContainifyCIFileWithProjects(projects []autodiscovery.GoProject) error {
	// Generate build groups from discovered projects
	buildGroups := autodiscovery.GenerateBuildGroups(projects)

	if len(buildGroups) == 0 {
		slog.Warn("No valid build groups generated. Falling back to static template.")
		return createContainifyCIFile()
	}

	fileName := ".containifyci/containifyci.go"

	// Check if the file exists
	if _, err := os.Stat(fileName); err == nil {
		slog.Debug("File already exists", "file", fileName)
		return nil
	} else if !os.IsNotExist(err) {
		slog.Error("Error checking file", "error", err, "file", fileName)
		return err
	}

	var buf bytes.Buffer
	templateData := TemplateData{Groups: buildGroups}

	err := template.Must(template.New("containifyci-go").Parse(string(mage))).
		Execute(&buf, templateData)
	if err != nil {
		slog.Error("Failed to render containifyci go file with build groups", "error", err)
		return err
	}

	// Write content to the file
	err = os.WriteFile(fileName, buf.Bytes(), 0644)
	if err != nil {
		slog.Error("Failed to write containifyci go file", "error", err)
		return err
	}

	// Run go generate on the file
	cmd := exec.Command("go", "generate", "-tags", "mage", fileName)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		slog.Error("Failed to run go generate", "error", err)
		return err
	}

	slog.Info("Created .containifyci/containifyci.go file with auto-discovered projects", "file", fileName, "groupCount", len(buildGroups))
	return nil
}

// RunInit handles the init command with support for auto-discovery
func RunInit(cmd *cobra.Command, args []string) error {
	// Create .containifyci directory first
	err := createContainifyCIDir()
	if err != nil {
		slog.Error("Failed to create .containifyci directory", "error", err)
		return err
	}

	// Check if auto flag is set
	auto, err := cmd.Flags().GetBool("auto")
	if err != nil {
		slog.Error("Failed to get auto flag", "error", err)
		return err
	}

	if auto {
		// Use autodiscovery to generate containifyci.go
		slog.Info("Auto-discovering Go projects...")
		projects, err := autodiscovery.DiscoverGoProjects(".")
		if err != nil {
			slog.Error("Failed to discover Go projects", "error", err)
			return fmt.Errorf("autodiscovery failed: %w", err)
		}

		if len(projects) == 0 {
			slog.Warn("No Go projects discovered. Falling back to static template.")
			return createContainifyCIFile()
		}

		slog.Info("Discovered Go projects", "count", len(projects))
		for _, project := range projects {
			slog.Info("Found project", "name", project.AppName, "path", project.ModulePath, "isService", project.IsService)
		}

		// Create file with discovered projects using template
		return createContainifyCIFileWithProjects(projects)
	} else {
		// Use static template (existing behavior)
		return createContainifyCIFile()
	}
}

func RunMage(cmd *cobra.Command, args []string) {
	err := createContainifyCIDir()
	if err != nil {
		slog.Error("Failed to create .containifyci directory", "error", err)
		os.Exit(1)
	}

	err = createContainifyCIFile()
	if err != nil {
		slog.Error("Failed to create .containifyci/containifyci.go file", "error", err)
		os.Exit(1)
	}
}

func createContainifyCIFile() error {
	fileName := ".containifyci/containifyci.go"
	// Check if the file exists
	if _, err := os.Stat(fileName); err == nil {
		slog.Debug("File already exists", "file", fileName)
		return nil
	} else if !os.IsNotExist(err) {
		slog.Error("Error checking file", "error", err, "file", fileName)
		return err
	}

	var buf bytes.Buffer
	templateData := TemplateData{Groups: nil} // Empty groups for static template

	err := template.Must(template.New("containifyci-go").Parse(string(mage))).
		Execute(&buf, templateData)
	if err != nil {
		slog.Error("Failed to render mage go file", "error", err)
		return err
	}

	// Write content to the file
	err = os.WriteFile(fileName, buf.Bytes(), 0644)
	if err != nil {
		slog.Error("Failed to write mage go file", "error", err)
		return err
	}

	// Run go generate on the file
	cmd := exec.Command("go", "generate", "-tags", "mage", fileName)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		slog.Error("Failed to run go generate", "error", err)
		return err
	}

	slog.Info("Created .containifyci/containifyci.go file", "file", fileName)

	return nil
}

func createContainifyCIDir() error {
	dirPath := ".containifyci"

	// Check if the directory exists
	_, err := os.Stat(dirPath)
	if err == nil {
		return nil
	}

	if !os.IsNotExist(err) {
		slog.Error("Error checking directory", "error", err, "directory", dirPath)
		return err
	}

	// Directory does not exist, create it
	err = os.MkdirAll(dirPath, os.ModePerm)
	if err != nil {
		slog.Error("Error creating directory", "error", err, "directory", dirPath)
		return err
	}
	return nil
}
