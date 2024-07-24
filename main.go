package main

import (
	"fmt"
	"os"

	"github.com/containifyci/engine-ci/cmd"
)

func main() {
	err := cmd.Execute()
	if err != nil {
		fmt.Printf("Main Error: %v", err)
		os.Exit(1)
	}

}
