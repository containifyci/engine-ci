package protobuf

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestProtocScript(t *testing.T) {
	type args struct {
		bs *BuildScript
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "TestScript",
			args: args{
				bs: &BuildScript{
					Command:        "protoc",
					TargetPackages: []string{"pkg/protos", "pkg2/protos"},
					SourceFiles:    []string{"pkg/protos/file1.proto", "pkg2/protos/file2.proto"},
				},
			},
			want: `#!/bin/sh
set -x
protoc -I=/src/pkg/protos --go-grpc_out=/src/pkg/protos --plugin=grpc --go_out=/src/pkg/protos /src/pkg/protos/file1.proto

protoc -I=/src/pkg2/protos --go-grpc_out=/src/pkg2/protos --plugin=grpc --go_out=/src/pkg2/protos /src/pkg2/protos/file2.proto

`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Script(tt.args.bs)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestBufScript(t *testing.T) {
	type args struct {
		bs *BuildScript
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "TestScript",
			args: args{
				bs: &BuildScript{
					Command:        "buf",
					TargetPackages: []string{"pkg/protos", "pkg2/protos"},
					SourceFiles:    []string{"pkg/protos/file1.proto", "pkg2/protos/file2.proto"},
				},
			},
			want: `#!/bin/sh
set -x
buf generate
`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Script(tt.args.bs)
			assert.Equal(t, tt.want, got)
		})
	}
}
