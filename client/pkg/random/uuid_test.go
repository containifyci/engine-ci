package random

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewUUID(t *testing.T) {
	t.Parallel()
	t.Run("generates valid UUID", func(t *testing.T) {
		t.Parallel()
		uuid, err := NewUUID()
		require.NoError(t, err)
		assert.NotEmpty(t, uuid)

		// Check format: 8-4-4-4-12 hexadecimal characters
		assert.Len(t, uuid, 36) // 32 hex chars + 4 hyphens
		assert.Contains(t, uuid, "-")

		// Check that it's properly formatted
		parts := []int{8, 13, 18, 23} // positions of hyphens
		for _, pos := range parts {
			assert.Equal(t, "-", string(uuid[pos]))
		}
	})

	t.Run("generates unique UUIDs", func(t *testing.T) {
		t.Parallel()
		uuid1, err := NewUUID()
		require.NoError(t, err)

		uuid2, err := NewUUID()
		require.NoError(t, err)

		assert.NotEqual(t, uuid1, uuid2)
	})

	t.Run("version 4 UUID", func(t *testing.T) {
		t.Parallel()
		uuid, err := NewUUID()
		require.NoError(t, err)

		// Version 4 UUID should have '4' as the first character of the 3rd group
		assert.Equal(t, "4", string(uuid[14]))
	})

	t.Run("variant RFC 4122", func(t *testing.T) {
		t.Parallel()
		uuid, err := NewUUID()
		require.NoError(t, err)

		// Variant bits should be 10xx in binary (8, 9, a, or b in hex)
		variantChar := uuid[19]
		assert.Contains(t, "89ab", string(variantChar))
	})

	t.Run("generates multiple unique UUIDs", func(t *testing.T) {
		t.Parallel()
		uuids := make(map[string]bool)
		count := 100

		for i := 0; i < count; i++ {
			uuid, err := NewUUID()
			require.NoError(t, err)
			assert.False(t, uuids[uuid], "UUID collision detected")
			uuids[uuid] = true
		}

		assert.Len(t, uuids, count)
	})
}
