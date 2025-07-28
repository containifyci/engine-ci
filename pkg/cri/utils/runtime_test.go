// utils/runtime_type_test.go

package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRuntimeTypeConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant RuntimeType
		expected string
	}{
		{"Docker", Docker, "docker"},
		{"Podman", Podman, "podman"},
		{"Test", Test, "test"},
		{"Host", Host, "host"},
		{"Unknown", Unknown, "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, string(tt.constant))
		})
	}
}
