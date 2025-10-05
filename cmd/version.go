package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:         "version",
	Short:       "Show the current version of engine-ci",
	Long:        `Display the current version information for the engine-ci binary.`,
	Run:         runVersion,
	Annotations: map[string]string{skipRootHooks: "true"},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}

func runVersion(cmd *cobra.Command, _ []string) {
	b, _ := json.MarshalIndent(RootArgs.version, "", "  ")
	fmt.Println(string(b))
}
