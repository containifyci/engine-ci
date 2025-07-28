package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetEnvWithDefault(t *testing.T) {
	t.Setenv("key", "value")
	tests := []struct {
		key  string
		def  func() string
		want string
	}{
		{
			key:  "key",
			def:  func() string { return "default" },
			want: "value",
		},
		{
			key:  "key",
			def:  nil,
			want: "value",
		},
		{
			key:  "key2",
			def:  func() string { return "default" },
			want: "default",
		},
		{
			key:  "key2",
			def:  nil,
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			value := GetEnvWithDefault(tt.key, tt.def)
			assert.Equal(t, tt.want, value)
		})
	}
}

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

func TestGetAllEnv(t *testing.T) {
	t.Setenv("key", "value")
	t.Setenv("key_env", "env:key")
	t.Setenv("key_cmd", "cmd:echo value")
	t.Setenv("key_cmd_default", "cmd:cat file || echo value")
	envs := GetAllEnvs([]string{"key", "key_env", "key_cmd", "key_cmd_default", "key_missing"}, "build")

	assert.Equal(t, "value", envs["key"])
	assert.Equal(t, "value", envs["key_env"])
	assert.Equal(t, "value", envs["key_cmd"])
	assert.Equal(t, "value", envs["key_cmd_default"])
	assert.Equal(t, "", envs["key_missing"])
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
