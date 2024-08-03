package main

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/containifyci/engine-ci/cmd"
)

var (
    version = "dev"
    commit  = "none"
    date    = "unknown"
		repo    = "github.com/containifyci/engine-ci"
)

func main() {
	v := cmd.SetVersionInfo(version, commit, date, repo)
	slog.Info("Version", "version", v)
	err := cmd.Execute()
	if err != nil {
		fmt.Printf("Main Error: %v", err)
		os.Exit(1)
	}
}
