package utils

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIDStore_Add(t *testing.T) {
	tests := []struct {
		name string
		ids  [][]string
		want []string
	}{
		{
			name: "add single ID",
			ids:  [][]string{{"id1"}},
			want: []string{"id1"},
		},
		{
			name: "add multiple IDs at once",
			ids:  [][]string{{"id1", "id2", "id3"}},
			want: []string{"id1", "id2", "id3"},
		},
		{
			name: "add IDs in multiple calls",
			ids:  [][]string{{"id1"}, {"id2"}, {"id3"}},
			want: []string{"id1", "id2", "id3"},
		},
		{
			name: "add empty",
			ids:  [][]string{{}},
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &IDStore{}
			for _, ids := range tt.ids {
				store.Add(ids...)
			}
			assert.Equal(t, tt.want, store.Get())
		})
	}
}

func TestIDStore_Get(t *testing.T) {
	tests := []struct {
		name     string
		initialIDs []string
		want     []string
	}{
		{
			name:     "empty store",
			initialIDs: nil,
			want:     nil,
		},
		{
			name:     "store with IDs",
			initialIDs: []string{"id1", "id2"},
			want:     []string{"id1", "id2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &IDStore{}
			if tt.initialIDs != nil {
				store.Add(tt.initialIDs...)
			}
			assert.Equal(t, tt.want, store.Get())
		})
	}
}

func TestIDStore_Concurrent(t *testing.T) {
	store := &IDStore{}
	var wg sync.WaitGroup

	// Concurrent writes
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id string) {
			defer wg.Done()
			store.Add(id)
		}(string(rune('a' + i)))
	}

	wg.Wait()

	// Verify all IDs were added
	ids := store.Get()
	assert.Len(t, ids, 10)
}
