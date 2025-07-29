package memory

import (
	"crypto/sha256"
	"strings"
	"testing"
)

// BenchmarkStringBuilderWithPool tests the performance of the string builder pool
func BenchmarkStringBuilderWithPool(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result := WithStringBuilder(func(sb *strings.Builder) string {
			sb.WriteString("container-image")
			sb.WriteByte(':')
			sb.WriteString("latest")
			sb.WriteByte('-')
			sb.WriteString("sha256")
			sb.WriteByte(':')
			sb.WriteString("abcdef123456")
			return sb.String()
		})
		_ = result
	}
}

// BenchmarkStringBuilderStandard tests standard string builder without pool
func BenchmarkStringBuilderStandard(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var sb strings.Builder
		sb.WriteString("container-image")
		sb.WriteByte(':')
		sb.WriteString("latest")
		sb.WriteByte('-')
		sb.WriteString("sha256")
		sb.WriteByte(':')
		sb.WriteString("abcdef123456")
		result := sb.String()
		_ = result
	}
}

// BenchmarkHashBufferWithPool tests hash operations with pooled buffers
func BenchmarkHashBufferWithPool(b *testing.B) {
	data := make([]byte, 1024) // 1KB of data
	for i := range data {
		data[i] = byte(i % 256)
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result := WithBufferReturn(HashBuffer, func(buffer []byte) []byte {
			hasher := sha256.New()
			hasher.Write(data)
			return hasher.Sum(buffer[:0])
		})
		_ = result
	}
}

// BenchmarkHashBufferStandard tests hash operations with standard byte slices
func BenchmarkHashBufferStandard(b *testing.B) {
	data := make([]byte, 1024) // 1KB of data
	for i := range data {
		data[i] = byte(i % 256)
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		hasher := sha256.New()
		hasher.Write(data)
		result := hasher.Sum(nil)
		_ = result
	}
}

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

// BenchmarkPoolContentionSimulation simulates concurrent pool usage
func BenchmarkPoolContentionSimulation(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			// Mix of operations that would happen during container builds
			WithStringBuilder(func(sb *strings.Builder) string {
				sb.WriteString("image:tag")
				return sb.String()
			})
			
			WithBuffer(HashBuffer, func(buffer []byte) {
				hasher := sha256.New()
				hasher.Write([]byte("test data"))
				hasher.Sum(buffer[:0])
			})
		}
	})
}