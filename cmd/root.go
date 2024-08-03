package cmd

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/containifyci/engine-ci/pkg/container"

	"github.com/dusted-go/logging/prettylog"
	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "engine-ci",
	Short: "A brief description of your application",
	Long: `A longer description that spans multiple lines and likely contains
examples and usage of using your application. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	// Run: func(cmd *cobra.Command, args []string) { },
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() error {
	logOpts := slog.HandlerOptions{
		Level:       slog.LevelInfo,
		AddSource:   false,
		ReplaceAttr: nil,
	}

	prettyHandler := prettylog.NewHandler(&logOpts)
	logger := slog.New(prettyHandler)
	slog.SetDefault(logger)

	return rootCmd.Execute()
}

func init() {
	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	// rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.fuild.yaml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

func All(opts *container.Build) error {
	os.Args[1] = "build"
	os.Args = append(os.Args, opts.AsFlags()...)
	return Execute()
}

func SetVersionInfo(version, commit, date, repo string) {
	rootCmd.Version = fmt.Sprintf("%s (Built on %s from Git SHA %s of %s)", version, date, commit, repo)
}
