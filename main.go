package main

import (
	"fmt"
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
	cmd.SetVersionInfo(version, commit, date, repo)
	err := cmd.Execute()
	if err != nil {
		fmt.Printf("Main Error: %v", err)
		os.Exit(1)
	}
}
