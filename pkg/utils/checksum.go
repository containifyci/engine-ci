package utils

import (
	"crypto/sha256"
	"encoding/hex"
)

// ShortChecksum returns the SHA-256 checksum as a hex string.
func ShortChecksum(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}
