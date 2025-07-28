package cmd

import (
	"bytes"
	_ "embed"
	"log/slog"
	"os"
	"os/exec"
	"text/template"

	"github.com/spf13/cobra"
)

//go:embed containifyci.go.tmpl
var mage []byte

// buildCmd represents the build command
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Command to generate containifyci.go file for containifyci usage",
	Long:  `Command to generate containifyci.go file for containifyci usage`,
	Run:   RunMage,
}

func init() {
	rootCmd.AddCommand(initCmd)
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

	err := template.Must(template.New("containifyci-go").Delims("~~~", "~~~").Parse(string(mage))).
		Execute(&buf, nil)
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
