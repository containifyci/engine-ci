package zig

import (
	"bytes"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/containifyci/engine-ci/pkg/cri/types"
)

type BuildScript struct {
	Folder      string
	CacheDir    string
	Optimize    string
	Target      string
	Platforms   []*types.PlatformSpec
	Verbose     bool
	HasBuildZon bool
}

func NewBuildScript(folder string, optimize string, target string, verbose bool, cacheDir string, platform []*types.PlatformSpec) *BuildScript {
	if folder == "" {
		folder = "."
	}

	// Check if build.zig.zon exists
	hasBuildZon := false
	buildZonPath := filepath.Join(folder, "build.zig.zon")
	if _, err := os.Stat(buildZonPath); err == nil {
		hasBuildZon = true
	}

	return &BuildScript{
		Folder:      folder,
		Optimize:    optimize,
		Target:      target,
		Platforms:   platform,
		Verbose:     verbose,
		CacheDir:    cacheDir,
		HasBuildZon: hasBuildZon,
	}
}

func (bs *BuildScript) Script() string {
	return script(bs)
}

func (bs *BuildScript) String() string {
	return script(bs)
}

func script(bs *BuildScript) string {
	buildCmds := zigBuildCmds(bs)

	scriptTemplate := `#!/bin/sh
{{- if .Verbose }}
set -xe
{{- else }}
set -e
{{- end }}
{{- if .CacheDir }}
export ZIG_GLOBAL_CACHE_DIR={{.CacheDir}}
{{- end }}
{{.BuildCmds}}
`

	t := template.Must(template.New("zigbuild").Parse(strings.TrimSpace(scriptTemplate)))
	var buffer bytes.Buffer

	data := map[string]interface{}{
		"Verbose":   bs.Verbose,
		"CacheDir":  bs.CacheDir,
		"Folder":    bs.Folder,
		"BuildCmds": buildCmds,
	}

	err := t.Execute(&buffer, data)
	if err != nil {
		slog.Error("Failed to render Zig build script", "error", err)
		os.Exit(1)
	}

	return buffer.String()
}

func zigBuildCmds(bs *BuildScript) string {
	var cmds []string

	// Build the zig build command
	buildCmd := "zig build --color off --summary all"

	if bs.Optimize != "" {
		buildCmd += fmt.Sprintf(" -Doptimize=%s", bs.Optimize)
	}

	if bs.Target != "" {
		buildCmd += fmt.Sprintf(" -Dtarget=%s", bs.Target)
	}

	if bs.Verbose {
		buildCmd += " --verbose"
	}

	cmds = append(cmds, buildCmd)

	testCmd := "zig test "
	if bs.Folder != "" && bs.Folder != "." {
		testCmd += fmt.Sprintf("%s/*.zig", bs.Folder)
	} else {
		testCmd += "*"
	}

	if bs.Verbose {
		testCmd += " 2>&1 | cat"
	}
	cmds = append(cmds, testCmd)

	e2etestCmd := "zig build test --summary all"
	if bs.Target != "" {
		e2etestCmd += fmt.Sprintf(" -Dtarget=%s", bs.Target)
	}
	cmds = append(cmds, e2etestCmd)

	return strings.Join(cmds, "\n")
}
