package rust

import (
	"bytes"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"text/template"

	"github.com/containifyci/engine-ci/pkg/cri/types"
)

type BuildScript struct {
	Folder    string
	CacheDir  string
	Profile   string
	Target    string
	Features  []string
	Platforms []*types.PlatformSpec
	Verbose   bool
}

func NewBuildScript(folder string, profile string, target string, features []string, verbose bool, cacheDir string, platforms []*types.PlatformSpec) *BuildScript {
	if folder == "" {
		folder = "."
	}

	return &BuildScript{
		Folder:    folder,
		Profile:   profile,
		Target:    target,
		Features:  features,
		Platforms: platforms,
		Verbose:   verbose,
		CacheDir:  cacheDir,
	}
}

func (bs *BuildScript) Script() string {
	return script(bs)
}

func (bs *BuildScript) String() string {
	return script(bs)
}

func script(bs *BuildScript) string {
	buildCmds := cargoBuildCmds(bs)

	scriptTemplate := `#!/bin/sh
{{- if .Verbose }}
set -xe
{{- else }}
set -e
{{- end }}
{{- if .CacheDir }}
export CARGO_HOME={{.CacheDir}}
{{- end }}
{{.BuildCmds}}
`

	t := template.Must(template.New("cargobuild").Parse(strings.TrimSpace(scriptTemplate)))
	var buffer bytes.Buffer

	data := map[string]interface{}{
		"Verbose":   bs.Verbose,
		"CacheDir":  bs.CacheDir,
		"Folder":    bs.Folder,
		"BuildCmds": buildCmds,
	}

	err := t.Execute(&buffer, data)
	if err != nil {
		slog.Error("Failed to render Rust build script", "error", err)
		os.Exit(1)
	}

	return buffer.String()
}

func cargoBuildCmds(bs *BuildScript) string {
	var cmds []string

	// Change to project directory if specified
	if bs.Folder != "" && bs.Folder != "." {
		cmds = append(cmds, fmt.Sprintf("cd %s", bs.Folder))
	}

	// Build the cargo build command
	buildCmd := "cargo build --color never"

	// Add profile (release or debug)
	if bs.Profile == "release" {
		buildCmd += " --release"
	}

	// Add target if specified
	if bs.Target != "" {
		buildCmd += fmt.Sprintf(" --target %s", bs.Target)
	}

	// Add features if specified
	if len(bs.Features) > 0 {
		buildCmd += fmt.Sprintf(" --features %s", strings.Join(bs.Features, ","))
	}

	if bs.Verbose {
		buildCmd += " --verbose"
	}

	cmds = append(cmds, buildCmd)

	// Run tests
	testCmd := "cargo test --color never"
	if bs.Profile == "release" {
		testCmd += " --release"
	}
	if bs.Target != "" {
		testCmd += fmt.Sprintf(" --target %s", bs.Target)
	}
	if bs.Verbose {
		testCmd += " --verbose"
	}
	cmds = append(cmds, testCmd)

	return strings.Join(cmds, "\n")
}
