package container

import (
	"sync"

	"github.com/containifyci/engine-ci/pkg/utils"
)

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
