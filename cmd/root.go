package cmd

import (
	"fmt"
	"log/slog"

	"github.com/containifyci/engine-ci/pkg/logger"

	"github.com/spf13/cobra"
)

type rootCmdArgs struct {
	Progress string
	Target   string
	Verbose  bool
}

var RootArgs = &rootCmdArgs{}

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "engine-ci",
	Short: "A brief description of your application",
	Long: `A longer description that spans multiple lines and likely contains
examples and usage of using your application. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		logOpts := slog.HandlerOptions{
			Level:       slog.LevelInfo,
			AddSource:   false,
			ReplaceAttr: nil,
		}

		if RootArgs.Verbose {
			logOpts.Level = slog.LevelDebug
			logOpts.AddSource = true
		}
		logger := slog.New(logger.New(RootArgs.Progress, logOpts))
		slog.SetDefault(logger)
		slog.Info("Progress logging format", "format", RootArgs.Progress)
	},
	PersistentPostRun: func(cmd *cobra.Command, args []string) {
		slog.Info("Flushing logs")
		logger.GetLogAggregator().Flush()
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	logOpts := slog.HandlerOptions{
		Level:       slog.LevelInfo,
		AddSource:   false,
		ReplaceAttr: nil,
	}

	slogger := slog.New(logger.NewRootLog(logOpts))
	slog.SetDefault(slogger)
	rootCmd.PersistentFlags().BoolVarP(&RootArgs.Verbose, "verbose", "v", false, "Enable verbose logging")
	rootCmd.PersistentFlags().StringVarP(&RootArgs.Target, "target", "t", "all", "The build target to run")
	rootCmd.PersistentFlags().StringVar(&RootArgs.Progress, "progress", "plain", "The progress logging format to use. Options are: progress, plain")
}

// func All(opts *container.Build) error {
// 	os.Args[1] = "build"
// 	os.Args = append(os.Args, opts.AsFlags()...)
// 	slog.Info("Running build command", "args", opts.AsFlags())
// 	return Execute()
// }

func SetVersionInfo(version, commit, date, repo string) string {
	rootCmd.Version = fmt.Sprintf("%s (Built on %s from Git SHA %s of %s)", version, date, commit, repo)
	return rootCmd.Version
}
