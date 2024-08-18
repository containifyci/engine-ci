package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetenv(t *testing.T) {
	t.Setenv("key", "value")
	t.Setenv("key_LOCAL", "value_LOCAL")
	tests := []struct {
		env  string
		key  string
		want string
	}{
		{
			env:  "local",
			key:  "key",
			want: "value_LOCAL",
		},
		{
			env:  "build",
			key:  "key",
			want: "value",
		},
		{
			env:  "production",
			key:  "key",
			want: "value",
		},
		{
			env:  "unknown",
			key:  "key",
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.env, func(t *testing.T) {
			value := Getenv(tt.key, tt.env)
			assert.Equal(t, tt.want, value)
		})
	}
}

func TestGetEnv(t *testing.T) {
	t.Setenv("key", "value")
	t.Setenv("key_env", "env:key")
	t.Setenv("key_cmd", "cmd:echo value")
	t.Setenv("key_cmd_default", "cmd:cat file || echo value")
	tests := []struct {
		key  string
		want string
	}{
		{
			key:  "key",
			want: "value",
		},
		{
			key:  "key_env",
			want: "value",
		},
		{
			key:  "key_cmd",
			want: "value",
		},
		{
			key:  "key_cmd_default",
			want: "value",
		},
	}
	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			value := GetEnv(tt.key, "build")
			assert.Equal(t, tt.want, value)
		})
	}
}

func TestGetenvs(t *testing.T) {
	t.Setenv("key", "value")
	t.Setenv("key_LOCAL", "value_LOCAL")

	v := GetEnvs([]string{"key2", "key"}, "local")
	assert.Equal(t, "value_LOCAL", v)

	v = GetEnvs([]string{"key2", "key"}, "build")
	assert.Equal(t, "value", v)

	v = GetEnvs([]string{"key2", "key1"}, "build")
	assert.Equal(t, "", v)
}
