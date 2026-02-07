package buildscript

import (
	"bufio"
	"bytes"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/containifyci/engine-ci/pkg/cri/types"
)

type CoverageMode string

type Image string

type GenerateMode string

const (
	GenerateModeAuto     GenerateMode = "auto"
	GenerateModeEnabled  GenerateMode = "enabled"
	GenerateModeDisabled GenerateMode = "disabled"
)

var excludedDirs = map[string]bool{
	"vendor":       true,
	"node_modules": true,
	"venv":         true,
	".git":         true,
	".cache":       true,
	"build":        true,
	"dist":         true,
	"bin":          true,
	"target":       true,
}

// detectGoGenerate scans the folder for //go:generate directives in .go files.
// It skips common excluded directories for performance and returns true on first match.
func detectGoGenerate(folder string) bool {
	found := false
	err := filepath.Walk(folder, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip files/dirs with errors
		}

		// Skip excluded directories
		if info.IsDir() {
			dirName := info.Name()
			// Skip hidden directories
			if strings.HasPrefix(dirName, ".") && dirName != "." {
				return filepath.SkipDir
			}
			// Skip excluded directories
			if excludedDirs[dirName] {
				return filepath.SkipDir
			}
			return nil
		}

		// Only process .go files
		if !strings.HasSuffix(path, ".go") {
			return nil
		}

		// Scan file for //go:generate directive
		file, err := os.Open(path)
		if err != nil {
			return nil // Skip files that can't be opened
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if strings.HasPrefix(line, "//go:generate") {
				found = true
				return filepath.SkipAll // Stop walking immediately
			}
		}

		return nil
	})

	if err != nil && err != filepath.SkipAll {
		slog.Debug("Error during go:generate detection", "error", err)
	}

	return found
}

type BuildScript struct {
	AppName         string
	MainFile        string
	Folder          string
	Output          string
	CoverageMode    CoverageMode
	FileName        string
	Tags            []string
	Platforms       []*types.PlatformSpec
	Artifacts       []string
	Verbose         bool
	NoCoverage      bool
	ShouldGenerate  bool
	CGOCrossCompile bool
}

func NewBuildScript(appName, mainfile string, folder string, tags []string, verbose bool, nocoverage bool, coverageMode CoverageMode, generateMode string, platforms ...*types.PlatformSpec) *BuildScript {
	filename := "{{.app}}-{{.os}}-{{.arch}}"
	if folder == "" {
		folder = "."
	}
	output := fmt.Sprintf("-o /src/%s", filename)
	if folder != "." {
		output = fmt.Sprintf("-o /src/%s/%s", folder, filename)
	}
	if mainfile == "" {
		mainfile = "./..."
		output = ""
		filename = ""
	}

	// Determine if go generate should run based on generateMode
	shouldGenerate := shouldGenerate(generateMode, folder)

	script := &BuildScript{
		AppName:        appName,
		CoverageMode:   coverageMode,
		Folder:         folder,
		MainFile:       mainfile,
		NoCoverage:     nocoverage,
		Output:         output,
		Platforms:      platforms,
		Tags:           tags,
		Verbose:        verbose,
		FileName:       filename,
		Artifacts:      []string{},
		ShouldGenerate: shouldGenerate,
	}
	return script
}

func NewCGOBuildScript(appName, mainfile string, folder string, tags []string, verbose bool, nocoverage bool, coverageMode CoverageMode, generateMode string, platforms ...*types.PlatformSpec) *BuildScript {
	script := NewBuildScript(appName, mainfile, folder, tags, verbose, nocoverage, coverageMode, generateMode, platforms...)
	script.CGOCrossCompile = true
	return script
}

func zigTarget(goos, goarch string) string {
	archMap := map[string]string{"arm64": "aarch64", "amd64": "x86_64"}
	osMap := map[string]string{"darwin": "macos", "linux": "linux-gnu"}
	return archMap[goarch] + "-" + osMap[goos]
}

func shouldGenerate(generateMode string, folder string) bool {
	switch GenerateMode(generateMode) {
	case GenerateModeEnabled:
		return true
	case GenerateModeDisabled:
		return false
	case GenerateModeAuto:
		fallthrough
	default:
		return detectGoGenerate(folder)
	}
}

// TODO: the -race flag needs CDO enabled for now https://github.com/golang/go/issues/6508
func (bs *BuildScript) String() string {
	return script(bs)
}

func script(bs *BuildScript) string {
	goBuildCmd := goBuildCmds(bs)
	generateCmd := ""
	if bs.ShouldGenerate {
		generateCmd = "go generate ./...\n"
	}
	script := fmt.Sprintf(`#!/bin/sh
set -xe
mkdir -p ~/.ssh
ssh-keyscan github.com >> ~/.ssh/known_hosts
git config --global url."ssh://git@github.com/.insteadOf" "https://github.com/"
cd %s
%s%s
`, bs.Folder, generateCmd, goBuildCmd)

	return script
}

func trim(str string, args ...any) string {
	s := strings.TrimSpace(str)
	if len(args) == 0 {
		return s
	}
	return fmt.Sprintf(s, args...)
}

func renderTestCommand(bs *BuildScript, m map[string]interface{}) string {
	var cmd string
	switch bs.CoverageMode {
	case "binary":
		cmd = "mkdir -p ${PWD}/.coverdata/unit\ngo test {{- .verbose }} -timeout 120s {{- .tags }} -cover ./... {{- .coverage}}"
	case "text":
		fallthrough
	default:
		cmd = "go test {{- .verbose }} -timeout 120s {{- .tags }} {{- .coverage}} ./..."
	}

	t := template.Must(template.New("").
		Parse(trim(cmd)))
	buf := new(bytes.Buffer)
	err := t.Execute(buf, m)
	if err != nil {
		slog.Error("Failed render go test cmd", "error", err)
		os.Exit(1)
	}
	return buf.String()
}

func (bs *BuildScript) renderCompileCommand(m map[string]interface{}) string {
	goos, _ := m["os"].(string)
	goarch, _ := m["arch"].(string)

	envPrefix := "env"
	if bs.CGOCrossCompile && goos == "darwin" {
		target := zigTarget(goos, goarch)
		envPrefix = fmt.Sprintf(`env CGO_ENABLED=1 CC="zig cc -target %s" CXX="zig c++ -target %s" CGO_LDFLAGS="" CGO_CFLAGS="" GOFLAGS="-ldflags=-w"`, target, target)
	}

	t := template.Must(template.New("").
		Parse(trim(`
%s GOOS={{.os}} GOARCH={{ .arch }} go build {{- .tags }} {{- .verbose }} %s {{.mainfile}}
`, envPrefix, bs.Output)))
	buf := new(bytes.Buffer)
	err := t.Execute(buf, m)
	if err != nil {
		slog.Error("Failed render go build cmd", "error", err)
		os.Exit(1)
	}

	if bs.FileName == "" {
		return buf.String()
	}
	t2 := template.Must(template.New("").
		Parse(bs.FileName))
	buf2 := new(bytes.Buffer)
	err2 := t2.Execute(buf2, m)
	if err2 != nil {
		slog.Error("Failed render go build cmd", "error", err)
		os.Exit(1)
	}

	bs.Artifacts = append(bs.Artifacts, buf2.String())
	return buf.String()
}

func coverage(nocoverage bool, coveragemode CoverageMode) string {
	if nocoverage {
		return ""
	}
	switch coveragemode {
	case "binary":
		return " -args -test.gocoverdir=${PWD}/.coverdata/unit"
	case "text":
		fallthrough
	default:
		return " -cover -coverprofile coverage.txt"

	}
}

func goBuildCmds(bs *BuildScript) string {
	var cmds []string
	for _, platform := range bs.Platforms {
		m := map[string]interface{}{"os": platform.OS, "arch": platform.Architecture, "app": bs.AppName, "mainfile": bs.MainFile, "verbose": "", "tags": ""}
		if bs.Verbose {
			m["verbose"] = " -x"
		}
		if len(bs.Tags) > 0 {
			m["tags"] = " -tags " + strings.Join(bs.Tags, ",")
		}
		cmds = append(cmds, bs.renderCompileCommand(m))
	}
	coverage := coverage(bs.NoCoverage, bs.CoverageMode)
	m := map[string]interface{}{"verbose": "", "tags": "", "cgo": 0, "coverage": coverage}
	if bs.Verbose {
		m["verbose"] = " -v"
	}
	if len(bs.Tags) > 0 {
		m["tags"] = " -tags " + strings.Join(bs.Tags, ",")
	}
	cmds = append(cmds, renderTestCommand(bs, m))
	return strings.Join(cmds, "\n")
}
