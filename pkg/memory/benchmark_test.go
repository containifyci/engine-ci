package memory

import (
	"testing"
)

// BenchmarkTarBufferWithPool tests large buffer operations with pool
func BenchmarkTarBufferWithPool(b *testing.B) {
	// Use buffer size that fits within TarBufferSize (64KB)
	bufferData := make([]byte, 32*1024) // 32KB of data
	for i := range bufferData {
		bufferData[i] = byte(i % 256)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		WithBuffer(TarBuffer, func(buffer []byte) {
			// Simulate copying data within buffer capacity
			copy(buffer[:len(bufferData)], bufferData)
		})
	}
}

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

// BenchmarkTarBufferPoolContention simulates concurrent TAR buffer usage
func BenchmarkTarBufferPoolContention(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			// Simulate TAR operations that would happen during container builds
			WithBuffer(TarBuffer, func(buffer []byte) {
				// Simulate copying file data
				copy(buffer[:1024], make([]byte, 1024))
			})
		}
	})
}
