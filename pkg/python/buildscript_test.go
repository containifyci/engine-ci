package python

import (
	"testing"

	"github.com/containifyci/engine-ci/pkg/container"
	"github.com/stretchr/testify/assert"
)

func TestPrivateIndexEmpty(t *testing.T) {
	custom := container.Custom{}
	pi := NewPrivateIndex(custom)

	assert.Equal(t, "", pi.String())
	assert.Equal(t, "", pi.Username())
	assert.Equal(t, "", pi.Environ())
}

func TestPrivateIndex(t *testing.T) {
	custom := container.Custom{
		"private_index": []string{"data-utils"},
	}
	pi := NewPrivateIndex(custom)

	assert.Equal(t, "DATA_UTILS", pi.String())
	assert.Equal(t, "UV_INDEX_DATA_UTILS_USERNAME=oauth2accesstoken", pi.Username())
	assert.Equal(t, `export UV_INDEX_DATA_UTILS_PASSWORD="$(curl -fsS -H "Authorization: Bearer ${CONTAINIFYCI_AUTH}" "${CONTAINIFYCI_HOST}/mem/accesstoken")"`, pi.Environ())
}
