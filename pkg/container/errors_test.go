package container

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestHasSamePrefix tests the hasSamePrefix function
func TestHasSamePrefix(t *testing.T) {
	// Test cases
	tests := []struct {
		err1 error
		err2 error
		want bool
	}{
		{nil, nil, true},
		{NewError("error"), NewError("error"), true},
		{NewError("error"), NewError("error message"), true},
		{NewError("error message"), NewError("error"), true},
		{NewError("error message"), NewError("error message"), true},
		{NewError("error"), nil, false},
		{nil, NewError("error"), false},
		{NewError("error"), nil, false},
		{NewError("error"), NewError("message"), false},
		{NewError("message"), NewError("error"), false},
	}

	// Run tests
	for _, tt := range tests {
		same := sameError(tt.err1, tt.err2)
		assert.Equal(t, tt.want, same)
	}
}

func NewError(message string) error {
	return fmt.Errorf(message)
}
