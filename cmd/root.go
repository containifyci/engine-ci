package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	_ "net/http/pprof"
	"os"
	"runtime"
	"runtime/pprof"
	"time"

	"github.com/containifyci/engine-ci/pkg/logger"

	"github.com/spf13/cobra"
)

type rootCmdArgs struct {
	cpuProfileFile *os.File
	httpSrv        *http.Server
	version        VersionInfo
	CPUProfile     string
	MemProfile     string
	Progress       string
	Target         string
	PProfPort      int
	Auto           bool
	PProfHTTP      bool
	Verbose        bool
}

type VersionInfo struct {
	Version string `json:"version"`
	Commit  string `json:"commit"`
	Date    string `json:"date"`
	Repo    string `json:"repo"`
}

const skipRootHooks = "skipRootHooks"

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
	SilenceUsage:  true,
	SilenceErrors: true,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if cmd.Annotations[skipRootHooks] == "true" {
			return nil
		}
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
		slog.Info("Version", "version", RootArgs.version)

		// Enable CPU profiling if requested
		if RootArgs.CPUProfile != "" {
			f, err := os.Create(RootArgs.CPUProfile)
			if err != nil {
				return fmt.Errorf("could not create CPU profile: %w", err)
			}
			if err := pprof.StartCPUProfile(f); err != nil {
				_ = f.Close()
				return fmt.Errorf("could not start CPU profile: %w", err)
			}
			RootArgs.cpuProfileFile = f
			slog.Info("CPU profiling started", "file", RootArgs.CPUProfile)
		}

		// Enable HTTP pprof endpoint if requested
		if RootArgs.PProfHTTP {
			addr := fmt.Sprintf("localhost:%d", RootArgs.PProfPort)
			RootArgs.httpSrv = &http.Server{Addr: addr}
			go func() {
				slog.Info("Starting pprof HTTP server", "addr", addr)
				if err := RootArgs.httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
					slog.Error("pprof server failed", "error", err)
				}
			}()
		}
		return nil
	},
	PersistentPostRunE: func(cmd *cobra.Command, args []string) error {
		if cmd.Annotations[skipRootHooks] == "true" {
			return nil
		}
		slog.Info("Flushing logs")
		logger.GetLogAggregator().Flush()

		// Stop CPU profiling if it was started
		if RootArgs.CPUProfile != "" {
			pprof.StopCPUProfile()
			slog.Info("CPU profiling stopped", "file", RootArgs.CPUProfile)
			if RootArgs.cpuProfileFile != nil {
				if err := RootArgs.cpuProfileFile.Close(); err != nil {
					slog.Warn("Failed to close CPU profile file", "error", err)
				}
			}
		}

		// Write memory profile if requested
		if RootArgs.MemProfile != "" {
			f, err := os.Create(RootArgs.MemProfile)
			if err != nil {
				return fmt.Errorf("could not create memory profile: %w", err)
			}
			defer f.Close()

			runtime.GC() // get up-to-date statistics
			if err := pprof.WriteHeapProfile(f); err != nil {
				return fmt.Errorf("could not write memory profile: %w", err)
			}
			slog.Info("Memory profile written", "file", RootArgs.MemProfile)
		}

		// Gracefully shutdown pprof HTTP server if enabled
		if RootArgs.httpSrv != nil {
			ctx := cmd.Context()
			shutdownCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
			defer cancel()
			if err := RootArgs.httpSrv.Shutdown(shutdownCtx); err != nil {
				slog.Warn("Failed to shutdown pprof server", "error", err)
			}
		}
		return nil
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
	rootCmd.PersistentFlags().BoolVarP(&RootArgs.Auto, "auto", "a", false, "The build target to run")
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
	RootArgs.version = VersionInfo{
		Version: version,
		Commit:  commit,
		Date:    date,
		Repo:    repo,
	}
	return rootCmd.Version
}

func RootCmd() *cobra.Command {
	return rootCmd
}
