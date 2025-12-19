package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/containifyci/engine-ci/pkg/doctor"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

type doctorArgs struct {
	Categories         []string
	Timeout            time.Duration
	Verbose            bool
	JSONOutput         bool
	NoColor            bool
	Parallel           bool
	KeepTestContainers bool
}

var doctorCmdArgs = &doctorArgs{}

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check engine-ci environment health and configuration",
	Long: `The doctor command runs diagnostic checks to verify that your
environment is properly configured for engine-ci.

It checks:
  - Container runtime detection and connectivity
  - Build tool availability (BuildKit, etc.)
  - Network connectivity to registries
  - System resources (memory, disk)
  - User permissions and group memberships
  - GitHub Actions configuration (when applicable)

Use --verbose for detailed diagnostic output.
Use --json for machine-readable output.`,
	Example: `  # Run all checks
  engine-ci doctor

  # Run with detailed output
  engine-ci doctor --verbose

  # Output as JSON
  engine-ci doctor --json

  # Run only specific categories
  engine-ci doctor --category "Container Runtime" --category "Network Access"`,
	Annotations: map[string]string{
		skipRootHooks: "true", // Skip root initialization for doctor
	},
	RunE: runDoctor,
}

func init() {
	rootCmd.AddCommand(doctorCmd)

	doctorCmd.Flags().BoolVarP(&doctorCmdArgs.Verbose, "verbose", "v", true,
		"Show detailed diagnostic information")
	doctorCmd.Flags().BoolVar(&doctorCmdArgs.JSONOutput, "json", false,
		"Output results as JSON")
	doctorCmd.Flags().BoolVar(&doctorCmdArgs.NoColor, "no-color", false,
		"Disable colored output")
	doctorCmd.Flags().BoolVar(&doctorCmdArgs.Parallel, "parallel", true,
		"Run checks in parallel")
	doctorCmd.Flags().StringSliceVar(&doctorCmdArgs.Categories, "category", nil,
		"Run only specific categories")
	doctorCmd.Flags().DurationVar(&doctorCmdArgs.Timeout, "timeout", 30*time.Second,
		"Timeout for individual checks")
}

func runDoctor(_ *cobra.Command, _ []string) error {
	// Setup logging
	if doctorCmdArgs.Verbose {
		slog.SetLogLoggerLevel(slog.LevelDebug)
	}

	// Parse categories if specified
	var categories []doctor.CheckCategory
	if len(doctorCmdArgs.Categories) > 0 {
		categories = make([]doctor.CheckCategory, len(doctorCmdArgs.Categories))
		for i, cat := range doctorCmdArgs.Categories {
			categories[i] = doctor.CheckCategory(cat)
		}
	}

	// Create doctor with options
	opts := doctor.DoctorOptions{
		Verbose:    doctorCmdArgs.Verbose,
		JSONOutput: doctorCmdArgs.JSONOutput,
		Parallel:   doctorCmdArgs.Parallel,
		Categories: categories,
	}
	d := doctor.NewDoctor(opts)

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(),
		doctorCmdArgs.Timeout*5) // Total timeout is 5x individual timeout
	defer cancel()

	// Run checks
	slog.Debug("Running diagnostic checks")
	results := d.RunChecks(ctx)

	// Format and output results
	useColor := !doctorCmdArgs.NoColor && isTerminal(os.Stdout)
	formatter := doctor.NewResultFormatter(os.Stdout,
		doctorCmdArgs.Verbose,
		doctorCmdArgs.JSONOutput,
		useColor)

	if err := formatter.FormatResults(results); err != nil {
		return fmt.Errorf("failed to format results: %w", err)
	}

	// Exit with error code if critical failures
	for _, result := range results {
		if result.Status == doctor.StatusFail &&
			result.Severity == doctor.SeverityCritical {
			return fmt.Errorf("critical environment issues detected")
		}
	}

	return nil
}

// isTerminal checks if output is a terminal
func isTerminal(f *os.File) bool {
	return term.IsTerminal(int(f.Fd()))
}
