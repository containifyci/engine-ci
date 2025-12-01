package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	dockerfile := []byte("FROM golang:1.23-alpine")
	p := New(dockerfile)

	require.NotNil(t, p)
	assert.Equal(t, dockerfile, p.dockerfile)
	assert.Nil(t, p.result)
}

func TestParser_parse_errors(t *testing.T) {
	testCases := []struct {
		name       string
		dockerfile string
	}{
		{"empty", ""},
		{"only comments", "# Comment only"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			p := New([]byte(tc.dockerfile))
			_, err := p.parse()
			assert.Error(t, err)
		})
	}
}

func TestParser_ParseFrom(t *testing.T) {
	tests := []struct {
		name       string
		dockerfile string
		want       []From
		wantErr    bool
	}{
		{
			name:       "simple FROM without stage",
			dockerfile: "FROM golang:1.23-alpine",
			want: []From{{
				BaseImage:   "golang:1.23-alpine",
				BaseVersion: "1.23-alpine",
				StageName:   "",
				Line:        1,
				Original:    "FROM golang:1.23-alpine"},
			},
		},
		{
			name:       "FROM with AS stage",
			dockerfile: "FROM golang:1.23-alpine AS builder",
			want: []From{{
				BaseImage:   "golang:1.23-alpine",
				BaseVersion: "1.23-alpine",
				StageName:   "builder",
				Line:        1,
				Original:    "FROM golang:1.23-alpine AS builder"},
			},
		},
		{
			name:       "FROM with scratch",
			dockerfile: "FROM scratch",
			want: []From{{
				BaseImage:   "scratch",
				BaseVersion: "latest",
				StageName:   "",
				Line:        1,
				Original:    "FROM scratch"},
			},
		},
		{
			name:       "FROM with platform",
			dockerfile: "FROM --platform=linux/amd64 golang:1.23-alpine",
			want: []From{{
				BaseImage:   "golang:1.23-alpine",
				BaseVersion: "1.23-alpine",
				StageName:   "",
				Line:        1,
				Original:    "FROM --platform=linux/amd64 golang:1.23-alpine"},
			},
		},
		{
			name: "multi-stage with named stages",
			dockerfile: `FROM golang:1.23-alpine AS builder
WORKDIR /app

FROM alpine:3.20
COPY --from=builder /app/myapp /myapp`,
			want: []From{{
				BaseImage:   "golang:1.23-alpine",
				BaseVersion: "1.23-alpine",
				StageName:   "builder",
				Line:        1,
				Original:    "FROM golang:1.23-alpine AS builder"}, {
				BaseImage:   "alpine:3.20",
				BaseVersion: "3.20",
				StageName:   "",
				Line:        4,
				Original:    "FROM alpine:3.20"},
			},
		},
		{
			name: "with comments and case variations",
			dockerfile: `# Build stage
from golang:1.23-alpine as builder
# Runtime stage
FROM alpine:3.20`,
			want: []From{{
				BaseImage:   "golang:1.23-alpine",
				BaseVersion: "1.23-alpine",
				StageName:   "builder",
				Line:        2,
				Original:    "from golang:1.23-alpine as builder"}, {
				BaseImage:   "alpine:3.20",
				BaseVersion: "3.20",
				StageName:   "",
				Line:        4,
				Original:    "FROM alpine:3.20"},
			},
		},
		{
			name:       "empty FROM statement",
			dockerfile: "FROM\nWORKDIR /app\nCOPY . .",
			wantErr:    true,
		},
		{
			name:       "no FROM statement",
			dockerfile: "WORKDIR /app\nCOPY . .",
			wantErr:    true,
		},
		{
			name:       "empty dockerfile",
			dockerfile: "",
			wantErr:    true,
		},
		{
			name:       "only comments",
			dockerfile: "# Comment\n# Another",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := New([]byte(tt.dockerfile))
			got, err := p.ParseFrom()

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestParser_parse_caching(t *testing.T) {
	dockerfile := "FROM golang:1.23-alpine\nWORKDIR /app"
	p := New([]byte(dockerfile))

	result1, err := p.parse()
	require.NoError(t, err)
	require.NotNil(t, result1)

	// Second call should return cached result
	result2, err := p.parse()
	require.NoError(t, err)
	assert.Same(t, result1, result2, "parse() should return cached result")
}

func TestParser_ExtractBaseVersion(t *testing.T) {
	tests := []struct {
		name      string
		baseImage string
		want      string
	}{
		{"with tag", "golang:1.23-alpine", "1.23-alpine"},
		{"with latest tag", "alpine:latest", "latest"},
		{"without tag", "ubuntu", "latest"},
		{"with digest", "myimage@sha256:abcdef", "abcdef"},
		{"complex tag", "myimage:2023.10.15-beta", "2023.10.15-beta"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractBaseVersion(tt.baseImage)
			assert.Equal(t, tt.want, got)
		})
	}
}
