package main

import (
	"fmt"
	"log"
	"os"

	"github.com/spf13/cobra"
)

var (
	version = "dev"
	commit  = "unknown"
	date    = "unknown"
)

var rootCmd = &cobra.Command{
	Use:   "sample-go-app",
	Short: "A sample Go application for testing",
	Long: `This is a sample Go application used for testing the engine-ci
build system. It demonstrates a typical Go CLI application structure
with dependencies, tests, and build configurations.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Sample Go App v%s\n", version)
		fmt.Printf("Commit: %s\n", commit)
		fmt.Printf("Built: %s\n", date)
		
		if len(args) > 0 {
			fmt.Printf("Arguments: %v\n", args)
		}
		
		// Demonstrate environment variable usage
		if env := os.Getenv("SAMPLE_ENV"); env != "" {
			fmt.Printf("Environment: %s\n", env)
		}
	},
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Version: %s\n", version)
		fmt.Printf("Commit: %s\n", commit)
		fmt.Printf("Date: %s\n", date)
	},
}

var echoCmd = &cobra.Command{
	Use:   "echo [message]",
	Short: "Echo a message",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		for i, arg := range args {
			if i > 0 {
				fmt.Print(" ")
			}
			fmt.Print(arg)
		}
		fmt.Println()
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(echoCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}