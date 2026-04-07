package container

import (
	"fmt"
	"sync"

	"github.com/containifyci/engine-ci/pkg/utils"
	"github.com/containifyci/engine-ci/protos2"
)

type BuildSecrets map[string]*BuildSecret

func (b BuildSecrets) Get(key string) *BuildSecret {
	v, ok := b[key]
	if ok {
		return v
	}
	return nil
}

func (b BuildSecrets) Add(secret *protos2.Secret) {
	b[secret.Key] = NewBuildSecret(secret)
}

func (b BuildSecrets) Available() (bool, []string) {
	if len(b) == 0 {
		return false, []string{}
	}
	required := []string{}
	for _, s := range b {
		if s.Value.Get() == "" {
			required = append(required, fmt.Sprintf("%s=%s", s.Key, s.Value.value))
		}
	}
	return len(required) == 0, required
}

type BuildSecret struct {
	Value *EnvValue
	Key   string
}

func NewBuildSecret(secret *protos2.Secret) *BuildSecret {
	return &BuildSecret{
		Key:   secret.Key,
		Value: NewEnvValue(secret.Value),
	}
}

type EnvValue struct {
	valueFnc func() string
	value    string
	// envType string //TODO: add support for real EnvType if really needed
}

type EnvValueResolver interface {
	GetValue(string, string) string
}

var resolver = utils.GetValue

func NewEnvValue(value string) *EnvValue {
	valueFnc := sync.OnceValue(func() string {
		return resolver(value, "build")
	})
	return &EnvValue{
		value:    value,
		valueFnc: valueFnc,
	}
}

func (e EnvValue) String() string {
	return e.value
}

func (e *EnvValue) Get() string {
	return e.valueFnc()
}
