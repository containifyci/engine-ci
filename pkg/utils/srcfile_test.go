package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSrcFile_IsEmpty(t *testing.T) {
	tests := []struct {
		name string
		s    SrcFile
		want bool
	}{
		{
			name: "empty string",
			s:    SrcFile(""),
			want: true,
		},
		{
			name: "non-empty string",
			s:    SrcFile("file.go"),
			want: false,
		},
		{
			name: "whitespace only",
			s:    SrcFile(" "),
			want: false,
		},
		{
			name: "path string",
			s:    SrcFile("/src/main.go"),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.s.IsEmpty())
		})
	}
}

func TestSrcFile_Container(t *testing.T) {
	tests := []struct {
		name string
		s    SrcFile
		want string
	}{
		{
			name: "empty string",
			s:    SrcFile(""),
			want: "",
		},
		{
			name: "special case /src/main.go",
			s:    NewSrcFile("/src", "main.go"),
			want: "/src/main.go",
		},
		{
			name: "regular file",
			s:    NewSrcFile("", "file.go"),
			want: "/src/file.go",
		},
		{
			name: "regular file",
			s:    NewSrcFile("go-app", "file.go"),
			want: "/src/go-app/file.go",
		},
		{
			name: "file with path",
			s:    NewSrcFile("pkg/utils", "helper.go"),
			want: "/src/pkg/utils/helper.go",
		},
		{
			name: "absolute path not /src/main.go",
			s:    NewSrcFile("/home/user/", "file.go"),
			want: "/src//home/user/file.go",
		},
		{
			name: "relative path with dots",
			s:    NewSrcFile("..", "file.go"),
			want: "/src/../file.go",
		},
		{
			name: "file with spaces",
			s:    NewSrcFile("", "my file.go"),
			want: "/src/my file.go",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.s.Container())
		})
	}
}

func TestSrcFile_Host(t *testing.T) {
	tests := []struct {
		name string
		s    SrcFile
		want string
	}{
		{
			name: "empty string",
			s:    SrcFile(""),
			want: "",
		},
		{
			name: "regular file",
			s:    SrcFile("file.go"),
			want: "file.go",
		},
		{
			name: "file with path",
			s:    SrcFile("pkg/utils/helper.go"),
			want: "pkg/utils/helper.go",
		},
		{
			name: "absolute path",
			s:    SrcFile("/home/user/file.go"),
			want: "/home/user/file.go",
		},
		{
			name: "special case /src/main.go",
			s:    SrcFile("/src/main.go"),
			want: "/src/main.go",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.s.Host())
		})
	}
}
