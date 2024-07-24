package types

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetContainerPlatform(t *testing.T) {
	tests := []struct {
		Host     string
		Expected string
	}{
		{
			Host:     "darwin/arm64",
			Expected: "linux/amd64",
		},
		{
			Host:     "darwin/amd64",
			Expected: "linux/amd64",
		},
		{
			Host:     "linux/amd64",
			Expected: "linux/amd64",
		},
	}

	for _, test := range tests {
		t.Run(test.Host, func(t *testing.T) {
			containerPlatform := GetContainerPlatform(ParsePlatform(test.Host))
			assert.Equal(t, test.Expected, containerPlatform.String())
		})
	}
}

func TestGetPlatforms(t *testing.T) {
	tests := []struct {
		Host     string
		Expected []string
	}{
		{
			Host:     "darwin/arm64",
			Expected: []string{"linux/arm64", "linux/amd64"},
		},
		{
			Host:     "darwin/amd64",
			Expected: []string{"linux/amd64", "linux/arm64"},
		},
		{
			Host:     "linux/arm64",
			Expected: []string{"linux/arm64"},
		},
		{
			Host:     "linux/amd64",
			Expected: []string{"linux/amd64"},
		},
	}

	for _, test := range tests {
		t.Run(test.Host, func(t *testing.T) {
			platform := GetPlatformSpec()
			platform.Host = ParsePlatform(test.Host)
			platforms := GetPlatforms(*platform)
			assert.Equal(t, test.Expected, platforms)
		})
	}
}
