package cmd

import (
	_ "embed"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

//go:embed templates/github/release.yaml
var releaseWorkflow []byte

//go:embed templates/github/publish.yml
var publishWorkflow []byte

//go:embed templates/github/pull-request.yaml
var pullRequestWorkflow []byte

// WorkflowTemplate represents a GitHub Actions workflow template
type WorkflowTemplate struct {
	Name     string
	Filename string
	Content  []byte
}

// Available workflow templates
var workflowTemplates = map[string]WorkflowTemplate{
	"release": {
		Name:     "Release",
		Filename: "release.yml",
		Content:  releaseWorkflow,
	},
	"publish": {
		Name:     "Publish",
		Filename: "publish.yml",
		Content:  publishWorkflow,
	},
	"pull-request": {
		Name:     "Pull Request",
		Filename: "pull-request.yml",
		Content:  pullRequestWorkflow,
	},
}

// Command arguments structure
type githubCmdArgs struct {
	WorkflowType string
	OutputDir    string
	All          bool
	DryRun       bool
	Force        bool
}

var githubArgs = &githubCmdArgs{}

var githubCmd = &cobra.Command{
	Use:   "github",
	Short: "Generate GitHub Actions workflow files for engine-ci",
	Long: `Generate GitHub Actions workflow files for engine-ci.

This command creates GitHub Actions workflow files in the .github/workflows/ directory
based on predefined templates. You can generate individual workflows or all at once.

Available workflow types:
  - release: Creates release workflow for main branch pushes
  - publish: Creates publish workflow for manual releases
  - pull-request: Creates PR validation workflow
  - legacy: Creates the original engine-ci workflow template

Examples:
  # Generate all workflows
  engine-ci github --all
  
  # Generate specific workflow
  engine-ci github --type=release
  
  # Dry run to preview changes
  engine-ci github --dry-run --all
  
  # Force overwrite existing files
  engine-ci github --all --force`,
	RunE: RunGithub,
}

func init() {
	rootCmd.AddCommand(githubCmd)

	// Add flags
	githubCmd.Flags().StringVar(&githubArgs.WorkflowType, "type", "", "Workflow type to generate (release, publish, pull-request, legacy)")
	githubCmd.Flags().StringVarP(&githubArgs.OutputDir, "output", "o", ".github/workflows", "Output directory for workflow files")
	githubCmd.Flags().BoolVarP(&githubArgs.All, "all", "a", false, "Generate all workflow files")
	githubCmd.Flags().BoolVar(&githubArgs.DryRun, "dry-run", false, "Preview what files would be created without writing them")
	githubCmd.Flags().BoolVarP(&githubArgs.Force, "force", "f", false, "Overwrite existing files without prompting")
}

func RunGithub(cmd *cobra.Command, args []string) error {
	// Validate arguments
	if err := validateGithubArgs(); err != nil {
		return err
	}

	// Determine which workflows to generate
	workflowsToGenerate, err := getWorkflowsToGenerate()
	if err != nil {
		return err
	}

	if len(workflowsToGenerate) == 0 {
		return fmt.Errorf("no workflows specified. Use --type, --all, or see --help for usage")
	}

	// Create output directory if it doesn't exist (unless dry-run)
	if !githubArgs.DryRun {
		if err := ensureOutputDirectory(); err != nil {
			return fmt.Errorf("failed to create output directory: %w", err)
		}
	}

	// Generate workflows
	for _, workflowName := range workflowsToGenerate {
		if err := generateWorkflow(workflowName); err != nil {
			return fmt.Errorf("failed to generate %s workflow: %w", workflowName, err)
		}
	}

	slog.Info("GitHub workflows generation completed successfully")
	return nil
}

func validateGithubArgs() error {
	if githubArgs.WorkflowType != "" && githubArgs.All {
		return fmt.Errorf("cannot specify both --type and --all flags")
	}

	if githubArgs.WorkflowType != "" {
		if _, exists := workflowTemplates[githubArgs.WorkflowType]; !exists {
			validTypes := make([]string, 0, len(workflowTemplates))
			for k := range workflowTemplates {
				validTypes = append(validTypes, k)
			}
			return fmt.Errorf("invalid workflow type '%s'. Valid types: %s", githubArgs.WorkflowType, strings.Join(validTypes, ", "))
		}
	}

	return nil
}

func getWorkflowsToGenerate() ([]string, error) {
	if githubArgs.All {
		workflows := make([]string, 0, len(workflowTemplates))
		for name := range workflowTemplates {
			workflows = append(workflows, name)
		}
		return workflows, nil
	}

	if githubArgs.WorkflowType != "" {
		return []string{githubArgs.WorkflowType}, nil
	}

	return []string{}, nil
}

func ensureOutputDirectory() error {
	if err := os.MkdirAll(githubArgs.OutputDir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", githubArgs.OutputDir, err)
	}
	return nil
}

func generateWorkflow(workflowName string) error {
	template := workflowTemplates[workflowName]
	outputPath := filepath.Join(githubArgs.OutputDir, template.Filename)

	if githubArgs.DryRun {
		slog.Info("Would create workflow file", "type", workflowName, "file", outputPath)
		return nil
	}

	// Check if file exists and handle accordingly
	if _, err := os.Stat(outputPath); err == nil && !githubArgs.Force {
		// File exists and not forcing overwrite
		fmt.Printf("File %s already exists. Overwrite? (y/N): ", outputPath)
		var response string
		_, err := fmt.Scanln(&response)
		if err != nil {
			// If there's an error reading input, default to not overwriting
			slog.Info("Skipped workflow file due to input error", "type", workflowName, "file", outputPath, "error", err)
			return nil
		}
		if strings.ToLower(strings.TrimSpace(response)) != "y" {
			slog.Info("Skipped workflow file", "type", workflowName, "file", outputPath)
			return nil
		}
	}

	// Write the workflow file
	if err := os.WriteFile(outputPath, template.Content, 0644); err != nil {
		return fmt.Errorf("failed to write file %s: %w", outputPath, err)
	}

	slog.Info("Created workflow file", "type", workflowName, "file", outputPath)
	return nil
}
