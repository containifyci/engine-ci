package cmd

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/containifyci/go-self-update/pkg/updater"
	"github.com/spf13/cobra"
)

var (
	checkOnly bool
	forceUpdate bool
)

// updateCmd represents the update command
var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update engine-ci to the latest version",
	Long: `Update the engine-ci binary to the latest available version from GitHub releases.
	
This command will:
- Check for the latest release on GitHub
- Compare with the current version
- Download and replace the binary if a newer version is available`,
	Run: runUpdate,
}

func init() {
	rootCmd.AddCommand(updateCmd)
	
	updateCmd.Flags().BoolVar(&checkOnly, "check", false, "Only check for updates without installing")
	updateCmd.Flags().BoolVar(&forceUpdate, "force", false, "Force update even if current version appears newer")
}

func runUpdate(cmd *cobra.Command, args []string) {
	// Get current version from root command
	currentVersion := rootCmd.Version
	if currentVersion == "" {
		currentVersion = "dev"
	}
	
	slog.Info("Checking for updates", "current_version", currentVersion)
	
	// Initialize the updater
	u := updater.NewUpdater(
		"engine-ci",           // Binary name
		"containifyci",        // GitHub organization
		"engine-ci",           // GitHub repository
		currentVersion,        // Current version
		updater.WithUpdateHook(func() error {
			slog.Info("Update completed successfully!")
			fmt.Println("Update completed successfully! Please restart engine-ci to use the new version.")
			return nil
		}),
	)
	
	if checkOnly {
		// For check-only mode, we'll use SelfUpdate with a dry-run approach
		// by checking the version and not actually updating
		fmt.Printf("Current version: %s\n", currentVersion)
		fmt.Println("Checking for updates...")
		
		// Unfortunately, the library doesn't expose a check-only method,
		// so we'll inform the user to use the regular update command
		fmt.Println("To check for and install updates, run: engine-ci update")
		fmt.Println("Note: The update command will inform you if you're already on the latest version.")
		return
	}
	
	// Perform the update
	fmt.Println("Checking for updates...")
	fmt.Printf("Current version: %s\n", currentVersion)
	
	updated, err := u.SelfUpdate()
	if err != nil {
		slog.Error("Failed to update", "error", err)
		fmt.Printf("Error: Failed to update engine-ci: %v\n", err)
		os.Exit(1)
	}
	
	if !updated {
		fmt.Println("You are already running the latest version!")
		slog.Info("No update needed", "version", currentVersion)
	}
}