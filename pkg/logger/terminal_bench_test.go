package logger

import (
	"bufio"
	"io"
	"strings"
	"sync"
	"testing"
	"time"
)

// BenchmarkLogAggregation benchmarks the log aggregation system
func BenchmarkLogAggregation(b *testing.B) {
	b.Run("LogAggregator Creation", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			// Reset the singleton for each iteration
			instance = nil
			once.Reset()

			aggregator := NewLogAggregator("standard")
			_ = aggregator
		}
	})

	b.Run("Single Routine Logging", func(b *testing.B) {
		// Reset singleton
		instance = nil
		once.Reset()

		aggregator := NewLogAggregator("standard")
		routineID := "test-routine"
		message := "This is a test log message for benchmarking purposes"

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			aggregator.LogMessage(routineID, message)
		}
	})

	b.Run("Multiple Routine Logging", func(b *testing.B) {
		// Reset singleton
		instance = nil
		once.Reset()

		aggregator := NewLogAggregator("standard")
		routines := []string{"routine-1", "routine-2", "routine-3", "routine-4", "routine-5"}
		message := "Concurrent log message from multiple routines"

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			routineID := routines[i%len(routines)]
			aggregator.LogMessage(routineID, message)
		}
	})

	b.Run("Progress Format Logging", func(b *testing.B) {
		// Reset singleton
		instance = nil
		once.Reset()

		aggregator := NewLogAggregator("progress")
		routineID := "progress-routine"
		message := "Progress message for real-time display"

		// Allow some time for the display goroutine to start
		time.Sleep(10 * time.Millisecond)

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			aggregator.LogMessage(routineID, message)
		}

		// Clean up
		aggregator.Flush()
	})
}

// BenchmarkLogEntry benchmarks LogEntry operations
func BenchmarkLogEntry(b *testing.B) {
	b.Run("Add Message to LogEntry", func(b *testing.B) {
		entry := &LogEntry{
			messages: make([]string, 0, maxLogLines),
		}
		message := "Test log message for LogEntry benchmarking"

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			entry.addMessage(message)
		}
	})

	b.Run("LogEntry with Overflow", func(b *testing.B) {
		entry := &LogEntry{
			messages: make([]string, 0, maxLogLines),
		}

		// Pre-fill to capacity
		for i := 0; i < maxLogLines; i++ {
			entry.addMessage("initial message")
		}

		message := "Overflow message"

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			entry.addMessage(message)
		}
	})

	b.Run("Last5Messages Function", func(b *testing.B) {
		// Create test message slices of various sizes
		testCases := [][]string{
			make([]string, 3),   // Less than 5
			make([]string, 5),   // Exactly 5
			make([]string, 10),  // More than 5
			make([]string, 100), // Much more than 5
		}

		// Fill with test data
		for i, messages := range testCases {
			for j := range messages {
				messages[j] = "message-" + string(rune('0'+i)) + "-" + string(rune('0'+j%10))
			}
		}

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			for _, messages := range testCases {
				result := last5Messages(messages)
				_ = result
			}
		}
	})
}

// BenchmarkConcurrentLogging benchmarks concurrent logging scenarios
func BenchmarkConcurrentLogging(b *testing.B) {
	b.Run("Concurrent Single Routine", func(b *testing.B) {
		// Reset singleton
		instance = nil
		once.Reset()

		aggregator := NewLogAggregator("standard")
		routineID := "concurrent-routine"
		message := "Concurrent message from same routine"

		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				aggregator.LogMessage(routineID, message)
			}
		})
	})

	b.Run("Concurrent Multiple Routines", func(b *testing.B) {
		// Reset singleton
		instance = nil
		once.Reset()

		aggregator := NewLogAggregator("standard")
		message := "Concurrent message from different routines"

		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			routineCounter := 0
			for pb.Next() {
				routineID := "routine-" + string(rune('A'+routineCounter%26))
				aggregator.LogMessage(routineID, message)
				routineCounter++
			}
		})
	})

	b.Run("Concurrent LogEntry Access", func(b *testing.B) {
		entry := &LogEntry{
			messages: make([]string, 0, maxLogLines),
		}
		message := "Concurrent message for LogEntry"

		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				entry.addMessage(message)
			}
		})
	})

	b.Run("Mixed Read Write Operations", func(b *testing.B) {
		// Reset singleton
		instance = nil
		once.Reset()

		aggregator := NewLogAggregator("standard")
		routineID := "mixed-routine"
		message := "Mixed operation message"

		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				// Simulate mixed operations
				aggregator.LogMessage(routineID, message)
				aggregator.SuccessMessage(routineID, "Success")
				aggregator.FailedMessage(routineID, "Failed")
			}
		})
	})
}

// BenchmarkStringOperations benchmarks string processing in logger
func BenchmarkStringOperations(b *testing.B) {
	b.Run("String Trimming", func(b *testing.B) {
		testStrings := []string{
			"message\n",
			"message with newline\n",
			"no newline",
			"multiple\nlines\nhere\n",
			strings.Repeat("x", 1000) + "\n",
		}

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			for _, str := range testStrings {
				result := strings.TrimSuffix(str, "\n")
				_ = result
			}
		}
	})

	b.Run("Error Detail Detection", func(b *testing.B) {
		testLines := []string{
			"normal log line",
			"another normal line",
			"errorDetail: something went wrong",
			"line with errorDetail in middle",
			"ERROR: not an errorDetail",
			strings.Repeat("x", 1000) + "errorDetail" + strings.Repeat("y", 1000),
		}

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			for _, line := range testLines {
				contains := strings.Contains(line, "errorDetail")
				_ = contains
			}
		}
	})
}

// BenchmarkIOOperations benchmarks I/O operations in logger
func BenchmarkIOOperations(b *testing.B) {
	b.Run("Write Operation", func(b *testing.B) {
		// Reset singleton
		instance = nil
		once.Reset()

		aggregator := NewLogAggregator("standard")
		testData := []byte("This is test data for write operation benchmarking\n")

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			n, err := aggregator.Write(testData)
			_ = n
			_ = err
		}
	})

	b.Run("Copy Operation Small", func(b *testing.B) {
		// Reset singleton
		instance = nil
		once.Reset()

		aggregator := NewLogAggregator("standard")

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			// Create a small test reader
			data := "line1\nline2\nline3\n"
			reader := io.NopCloser(strings.NewReader(data))

			n, err := aggregator.Copy(reader)
			_ = n
			_ = err
		}
	})

	b.Run("Copy Operation Large", func(b *testing.B) {
		// Reset singleton
		instance = nil
		once.Reset()

		aggregator := NewLogAggregator("standard")

		// Create large test data
		var largeData strings.Builder
		for i := 0; i < 1000; i++ {
			largeData.WriteString("This is line number ")
			largeData.WriteString(string(rune('0' + i%10)))
			largeData.WriteString(" for testing large copy operations\n")
		}

		testData := largeData.String()

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			reader := io.NopCloser(strings.NewReader(testData))
			n, err := aggregator.Copy(reader)
			_ = n
			_ = err
		}
	})

	b.Run("Scanner Operations", func(b *testing.B) {
		testData := strings.Repeat("line\n", 1000)

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			scanner := bufio.NewScanner(strings.NewReader(testData))
			lineCount := 0

			for scanner.Scan() {
				line := scanner.Text()
				_ = line
				lineCount++
			}

			_ = lineCount
		}
	})
}

// BenchmarkMemoryUsage benchmarks memory allocation patterns
func BenchmarkMemoryUsage(b *testing.B) {
	b.Run("LogMessage Struct Creation", func(b *testing.B) {
		routineID := "test-routine"
		message := "test message"

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			logMsg := LogMessage{
				routineID: routineID,
				message:   message,
				isDone:    false,
				isFailed:  false,
			}
			_ = logMsg
		}
	})

	b.Run("LogEntry Slice Growth", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			entry := &LogEntry{
				messages: make([]string, 0, maxLogLines),
			}

			// Add messages up to capacity
			for j := 0; j < maxLogLines*2; j++ {
				entry.addMessage("message")
			}
		}
	})

	b.Run("Sync Map Operations", func(b *testing.B) {
		var logMap sync.Map
		routineIDs := []string{"routine-1", "routine-2", "routine-3", "routine-4", "routine-5"}

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			routineID := routineIDs[i%len(routineIDs)]
			entry, _ := logMap.LoadOrStore(routineID, &LogEntry{
				messages:  make([]string, 0, maxLogLines),
				startTime: time.Now(),
			})
			_ = entry
		}
	})
}

// BenchmarkTimeOperations benchmarks time-related operations
func BenchmarkTimeOperations(b *testing.B) {
	b.Run("Time.Now Calls", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			now := time.Now()
			_ = now
		}
	})

	b.Run("Time.Since Calculations", func(b *testing.B) {
		startTime := time.Now()

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			elapsed := time.Since(startTime)
			_ = elapsed
		}
	})

	b.Run("Time.Sub Operations", func(b *testing.B) {
		startTime := time.Now()
		time.Sleep(1 * time.Millisecond) // Small delay
		endTime := time.Now()

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			elapsed := endTime.Sub(startTime)
			_ = elapsed
		}
	})
}
