package cmd

import (
	"fmt"
	"log/slog"
	"net/http"
	_ "net/http/pprof"
	"os"
	"runtime"
	"runtime/pprof"

	"github.com/containifyci/engine-ci/pkg/logger"

	"github.com/spf13/cobra"
)

type rootCmdArgs struct {
	Progress   string
	Target     string
	Verbose    bool
	// Profiling options
	CPUProfile string
	MemProfile string
	PProfHTTP  bool
	PProfPort  int
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
		
		// Enable CPU profiling if requested
		if RootArgs.CPUProfile != "" {
			f, err := os.Create(RootArgs.CPUProfile)
			if err != nil {
				slog.Error("Could not create CPU profile", "error", err)
				os.Exit(1)
			}
			if err := pprof.StartCPUProfile(f); err != nil {
				slog.Error("Could not start CPU profile", "error", err)
				f.Close()
				os.Exit(1)
			}
			slog.Info("CPU profiling started", "file", RootArgs.CPUProfile)
		}
		
		// Enable HTTP pprof endpoint if requested
		if RootArgs.PProfHTTP {
			go func() {
				addr := fmt.Sprintf("localhost:%d", RootArgs.PProfPort)
				slog.Info("Starting pprof HTTP server", "addr", addr)
				if err := http.ListenAndServe(addr, nil); err != nil {
					slog.Error("pprof server failed", "error", err)
				}
			}()
		}
	},
	PersistentPostRun: func(cmd *cobra.Command, args []string) {
		slog.Info("Flushing logs")
		logger.GetLogAggregator().Flush()
		
		// Stop CPU profiling if it was started
		if RootArgs.CPUProfile != "" {
			pprof.StopCPUProfile()
			slog.Info("CPU profiling stopped", "file", RootArgs.CPUProfile)
		}
		
		// Write memory profile if requested
		if RootArgs.MemProfile != "" {
			f, err := os.Create(RootArgs.MemProfile)
			if err != nil {
				slog.Error("Could not create memory profile", "error", err)
				return
			}
			defer f.Close()
			
			runtime.GC() // get up-to-date statistics
			if err := pprof.WriteHeapProfile(f); err != nil {
				slog.Error("Could not write memory profile", "error", err)
			} else {
				slog.Info("Memory profile written", "file", RootArgs.MemProfile)
			}
		}
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
	
	// Profiling flags
	rootCmd.PersistentFlags().StringVar(&RootArgs.CPUProfile, "cpuprofile", "", "write cpu profile to file")
	rootCmd.PersistentFlags().StringVar(&RootArgs.MemProfile, "memprofile", "", "write memory profile to file")
	rootCmd.PersistentFlags().BoolVar(&RootArgs.PProfHTTP, "pprof-http", false, "enable HTTP pprof endpoint")
	rootCmd.PersistentFlags().IntVar(&RootArgs.PProfPort, "pprof-port", 6060, "HTTP pprof port")
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
