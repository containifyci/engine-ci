package alpine

import (
	"bytes"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"text/template"

	"github.com/containifyci/engine-ci/pkg/cri/types"
)

type Image string

type BuildScript struct {
	Tags      []string
	AppName   string
	MainFile  string
	Platforms []*types.PlatformSpec
	Verbose   bool
}

func NewBuildScript(appName, mainfile string, tags []string, verbose bool, platforms ...*types.PlatformSpec) *BuildScript {
	script := &BuildScript{
		AppName:   appName,
		MainFile:  mainfile,
		Platforms: platforms,
		Tags:      tags,
		Verbose:   verbose,
	}
	return script
}

// TODO: the -race flag needs CDO enabled for now https://github.com/golang/go/issues/6508
func Script(bs *BuildScript) string {
	return script(bs)
}

func script(bs *BuildScript) string {
	goBuildCmd := goBuildCmds(bs)
	script := fmt.Sprintf(`#!/bin/sh
set -xe
mkdir -p ~/.ssh
ssh-keyscan github.com >> ~/.ssh/known_hosts
git config --global url."ssh://git@github.com/.insteadOf" "https://github.com/"
%s
`, goBuildCmd)

	return script
}

func trim(str string) string {
	return strings.TrimSpace(str)
}

func renderTestCommand(m map[string]interface{}) string {
	t := template.Must(template.New("").
		Parse(trim(`
go test {{- .verbose }} -timeout 120s {{- .tags }} -cover -coverprofile coverage.txt ./...
`)))
	buf := new(bytes.Buffer)
	err := t.Execute(buf, m)
	if err != nil {
		slog.Error("Failed render go test cmd", "error", err)
		os.Exit(1)
	}
	return buf.String()
}

func renderCompileCommand(m map[string]interface{}) string {
	t := template.Must(template.New("").
		Parse(trim(`
env GOOS={{.os}} GOARCH={{ .arch }} go build {{- .tags }} {{- .verbose }} -o /src/{{.app}}-{{.os}}-{{.arch}} {{.mainfile}}
`)))
	buf := new(bytes.Buffer)
	err := t.Execute(buf, m)
	if err != nil {
		slog.Error("Failed render go build cmd", "error", err)
		os.Exit(1)
	}
	return buf.String()
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
		cmds = append(cmds, renderCompileCommand(m))
	}
	m := map[string]interface{}{"verbose": "", "tags": ""}
	if bs.Verbose {
		m["verbose"] = " -v"
	}
	if len(bs.Tags) > 0 {
		m["tags"] = " -tags " + strings.Join(bs.Tags, ",")
	}
	cmds = append(cmds, renderTestCommand(m))
	return strings.Join(cmds, "\n")
}
