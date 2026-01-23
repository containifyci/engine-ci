package github

import (
	"testing"
)

func TestGenerateFallbackMessage(t *testing.T) {
	tests := []struct {
		name  string
		want  string
		files []string
	}{
		{
			name:  "no files",
			files: []string{},
			want:  "chore: automated changes",
		},
		{
			name:  "nil files",
			files: nil,
			want:  "chore: automated changes",
		},
		{
			name:  "single file",
			files: []string{"main.go"},
			want:  "chore: update main.go",
		},
		{
			name:  "multiple files",
			files: []string{"main.go", "go.mod", "README.md"},
			want:  "chore: update 3 files",
		},
		{
			name:  "two files",
			files: []string{"a.go", "b.go"},
			want:  "chore: update 2 files",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GenerateFallbackMessage(tt.files)
			if got != tt.want {
				t.Errorf("GenerateFallbackMessage() = %v, want %v", got, tt.want)
			}
		})
	}
}
