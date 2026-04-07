package container

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

type TestEnvValueResolver struct {
	envs map[string]string
}

func (te TestEnvValueResolver) GetValue(key string, _ string) string {
	v, ok := te.envs[key]
	if ok {
		return v
	}
	panic(fmt.Errorf("Expected env var %s not found", key))
}

func TestNewEnvValue(t *testing.T) {
	NewEnvValue("test_env")
}

func TestString(t *testing.T) {
	e := NewEnvValue("test_env")

	assert.Equal(t, "test_env", e.String())
}

func TestGet(t *testing.T) {
	te := &TestEnvValueResolver{
		map[string]string{
			"env:test_env": "test_env_value",
		},
	}
	resolver = te.GetValue

	e := NewEnvValue("env:test_env")

	assert.Equal(t, "test_env_value", e.Get())
}
