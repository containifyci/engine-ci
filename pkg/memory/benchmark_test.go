package memory

import (
	"testing"
)

// BenchmarkTarBufferStandard tests large buffer operations without pool
func BenchmarkTarBufferStandard(b *testing.B) {
	bufferData := make([]byte, 32*1024) // 32KB of data
	for i := range bufferData {
		bufferData[i] = byte(i % 256)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buffer := make([]byte, 64*1024) // Allocate 64KB buffer
		copy(buffer[:len(bufferData)], bufferData)
		_ = buffer
	}
}
