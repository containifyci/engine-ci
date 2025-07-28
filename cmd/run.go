package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

// RunCommand is the command to run the service
var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Command to start the containifyci pipeline execution",
	Long:  `Command to start the containifyci pipeline execution`,
	RunE:  RunCommand,
}

func init() {
	rootCmd.AddCommand(runCmd)
}

func RunCommand(cmd *cobra.Command, args []string) error {
	os.Setenv("CONTAINIFYCI_FILE", ".containifyci/containifyci.go")
	return Engine(cmd, args)
}
