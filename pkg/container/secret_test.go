package container

import (
	"fmt"
	"testing"

	"github.com/containifyci/engine-ci/protos2"
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
	origResolver := resolver
	defer func() {
		resolver = origResolver
	}()
	te := &TestEnvValueResolver{
		map[string]string{
			"env:test_env": "test_env_value",
		},
	}
	resolver = te.GetValue

	e := NewEnvValue("env:test_env")

	assert.Equal(t, "test_env_value", e.Get())
}

func TestGetNil(t *testing.T) {
	bs := make(BuildSecrets, 0)

	bs.Add(&protos2.Secret{
		Key:   "k",
		Value: "env:test_env",
	})
	res := bs.Get("k2")
	assert.Nil(t, res)
}

func TestAvailable(t *testing.T) {
	t.Setenv("test_env", "test_env_value")
	bs := make(BuildSecrets, 1)

	bs.Add(&protos2.Secret{
		Key:   "k",
		Value: "env:test_env",
	})
	assert.True(t, bs.Available())

	res := bs.Get("k").Value.Get()
	assert.Equal(t, "test_env_value", res)
}

func TestUnAvailable(t *testing.T) {
	bs := make(BuildSecrets, 0)

	bs.Add(&protos2.Secret{
		Key:   "k",
		Value: "env:test_env",
	})
	assert.False(t, bs.Available())
}

func TestUnAvailableNoSecretDefined(t *testing.T) {
	bs := make(BuildSecrets, 0)
	assert.False(t, bs.Available())
}
