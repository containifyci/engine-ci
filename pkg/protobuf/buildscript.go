package protobuf

import (
	"bytes"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

type BuildScript struct {
	Command        string
	TargetPackages []string
	SourceFiles    []string
	WithHttp       bool
	WithTag        bool
}

func NewBuildScript(Command string, SourcePackages, SourceFiles []string, WithHttp, WithTag bool) *BuildScript {
	return &BuildScript{
		WithHttp:       WithHttp,
		WithTag:        WithTag,
		Command:        Command,
		TargetPackages: SourcePackages,
		SourceFiles:    SourceFiles,
	}
}

func Script(bs *BuildScript) string {
	return script(bs)
}

func script(bs *BuildScript) string {
	// p := "--gohttp_out=/src/{{.package}}"
	// t := "protoc -I=/src/{{.source}} --go-grpc_out=/src/{{.package}} --plugin=grpc --gotag_out=outdir=\"./{{.package}}\":./ /src/{{.file}}"
	cmds := []string{}
	// nolint:staticcheck
	if bs.Command == "protoc" {
		for i, pkg := range bs.TargetPackages {
			m := map[string]interface{}{"source": filepath.Dir(bs.SourceFiles[i]), "package": pkg, "file": bs.SourceFiles[i], "WithTag": bs.WithTag, "WithHttp": bs.WithHttp}
			slog.Debug("Rendering protobuf cmd", "m", m)
			t := template.Must(template.New("").Parse(
				`protoc -I=/src/{{.source}} --go-grpc_out=/src/{{.package}} --plugin=grpc --go_out=/src/{{.package}} {{- if .WithHttp }} --gohttp_out=/src/{{.package}} {{- end }} /src/{{.file}}
{{ if .WithTag -}} protoc -I=/src/{{.source}} --go-grpc_out=/src/{{.package}} --plugin=grpc --gotag_out=outdir=./{{.package}}:./ /src/{{.file}} {{ end -}}`))
			buf := new(bytes.Buffer)
			err := t.Execute(buf, m)
			if err != nil {
				slog.Error("Failed render protobuf cmd", "error", err)
				os.Exit(1)
			}
			cmds = append(cmds, buf.String())
		}
	} else if bs.Command == "buf" {
		cmds = append(cmds, "buf generate")
	} else {
		slog.Error("Unknown protobuf command", "command", bs.Command)
		os.Exit(1)
	}
	cmd := strings.Join(cmds, "\n")
	script := fmt.Sprintf(`#!/bin/sh
set -x
%s
`, cmd)

	return script
}
