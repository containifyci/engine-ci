package cmd

import (
	_ "embed"
	"log/slog"
	"os"
	"text/template"

	"github.com/spf13/cobra"
)

//go:embed github-workflow.yml.tmpl
var githubWorkflow []byte

// buildCmd represents the build command
var githubCmd = &cobra.Command{
	Use:   "github",
	Short: "Command to generate Github Action file for engine-ci",
	Long: `Command to generate Github Action file for engine-ci.
`,
	Run: RunGithub,
}

func init() {
	rootCmd.AddCommand(githubCmd)
}

func RunGithub(cmd *cobra.Command, args []string) {
	err := template.Must(template.New("github-workflow").Delims("~~~", "~~~").Parse(string(githubWorkflow))).
		Execute(os.Stdout, nil)
	if err != nil {
		slog.Error("Failed to render github-workflow yaml", "error", err)
		os.Exit(1)
	}
}
