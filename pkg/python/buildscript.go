package python

import (
	"fmt"
	"strings"

	"github.com/containifyci/engine-ci/pkg/container"
)

type Image string
type PrivateIndex string

func NewPrivateIndex(custom container.Custom) PrivateIndex {
	pi := custom.String("private_index")
	return PrivateIndex(pi)
}

func (pi PrivateIndex) String() string {
	return strings.ReplaceAll(strings.ToUpper(string(pi)), "-", "_")
}

func (pi PrivateIndex) Username() string {
	if pi == "" {
		return ""
	}
	return fmt.Sprintf("UV_INDEX_%s_USERNAME=oauth2accesstoken", pi.String())
}

func (pi PrivateIndex) Environ() string {
	if pi == "" {
		return ""
	}
	return fmt.Sprintf("export UV_INDEX_%s_PASSWORD=\"$(curl -fsS -H \"Authorization: Bearer ${CONTAINIFYCI_AUTH}\" \"${CONTAINIFYCI_HOST}/mem/accesstoken\")\"", pi.String())
}

type BuildScript struct {
	Folder       string
	Verbose      bool
	Commands     Commands
	PrivateIndex PrivateIndex
}

func NewBuildScript(folder string, verbose bool, privateIndex PrivateIndex, commands Commands) *BuildScript {
	return &BuildScript{
		Folder:       folder,
		Verbose:      verbose,
		Commands:     commands,
		PrivateIndex: privateIndex,
	}
}

func Script(bs *BuildScript) string {
	if bs.Verbose {
		return verboseScript(bs)
	}
	return simpleScript(bs)
}

func simpleScript(bs *BuildScript) string {
	cmd := bs.Commands.String()
	return fmt.Sprintf(`#!/bin/sh
set -e
%s
set -xe
cd %s
%s
`, bs.PrivateIndex.Environ(), bs.Folder, cmd)
}

func verboseScript(bs *BuildScript) string {
	cmd := bs.Commands.String()
	return fmt.Sprintf(`#!/bin/sh
set -e
%s
set -xe
cd %s
%s
`, bs.PrivateIndex.Environ(), bs.Folder, cmd)
}
