package types

import (
	"strings"
)

// Function to compare error messages by prefix
func SameError(err1, err2 error) bool {
	if err1 == nil || err2 == nil {
		return err1 == err2
	}
	// Get error messages as strings
	msg1 := err1.Error()
	msg2 := err2.Error()

	// Compare the prefixes
	return strings.HasPrefix(msg1, msg2) || strings.HasPrefix(msg2, msg1)
}
