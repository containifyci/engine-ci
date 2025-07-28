package logger

import (
	"bufio"
	"fmt"
	"io"
	"slices"
	"strings"
	"sync"
	"time"
)

const (
	reset       = "\033[0m"  // Reset to default color
	green       = "\033[32m" // Green text
	red         = "\033[31m" // Red text
	grayscale   = "\033[90m" // Grayscale text
	maxLogLines = 5          // Maximum lines per routine
)

type ResettableOnce struct {
	mu       sync.Mutex
	executed bool
}

func (ro *ResettableOnce) Do(f func()) {
	ro.mu.Lock()
	defer ro.mu.Unlock()
	if !ro.executed {
		f()
		ro.executed = true
	}
}

func (ro *ResettableOnce) Reset() {
	ro.mu.Lock()
	defer ro.mu.Unlock()
	ro.executed = false
}

type (
	LogEntry struct {
		startTime time.Time
		endTime   time.Time
		messages  []string
		mu        sync.Mutex
		isDone    bool
		isFailed  bool
	}

	LogAggregator struct {
		logChannel   chan LogMessage
		flushDone    chan struct{}
		logMap       sync.Map
		format       string
		routineOrder []string
	}

	LogMessage struct {
		routineID string
		message   string
		isDone    bool
		isFailed  bool
	}
)

// Singleton instance
var instance *LogAggregator
var once ResettableOnce

func (le *LogEntry) addMessage(msg string) {
	le.mu.Lock()
	defer le.mu.Unlock()
	if len(le.messages) >= maxLogLines {
		// Maintain fixed size by removing the oldest entry
		le.messages = le.messages[1:]
	}
	le.messages = append(le.messages, msg)
}

// GetLogAggregator returns the singleton instance of LogAggregator.
func NewLogAggregator(format string) *LogAggregator {
	once.Do(func() {
		instance = &LogAggregator{
			logChannel: make(chan LogMessage),
			flushDone:  make(chan struct{}),
			format:     format,
		}
		if format == "progress" {
			go instance.startLogDisplay()
		}
	})
	return instance
}

func GetLogAggregator() *LogAggregator {
	if instance == nil {
		panic("LogAggregator is not initialized")
	}
	return instance
}

func last5Messages(messages []string) []string {
	if len(messages) <= 5 {
		return messages
	}
	return messages[len(messages)-5:]
}

func readFromChannel(la *LogAggregator) []string {
	logMsg, ok := <-la.logChannel
	if !ok {
		// Channel is closed, break the loop to finish
		la.flushDone <- struct{}{}
		return nil
	}
	entry, _ := la.logMap.LoadOrStore(logMsg.routineID, &LogEntry{messages: make([]string, 0, maxLogLines), startTime: time.Now()})
	logEntry := entry.(*LogEntry)

	logEntry.addMessage(logMsg.message)

	// If the routine is done, mark it
	if logMsg.isDone {
		logEntry.mu.Lock()
		logEntry.isDone = logMsg.isDone
		logEntry.isFailed = logMsg.isFailed
		logEntry.endTime = time.Now()
		logEntry.mu.Unlock()
	}
	if !slices.Contains(la.routineOrder, logMsg.routineID) {
		la.routineOrder = append(la.routineOrder, logMsg.routineID)
	}
	return la.routineOrder
}

func (la *LogAggregator) startLogDisplay() {
	fmt.Println("Starting Real-Time Log Aggregation...")

	// Continuously update the console with the current log state
	for {

		routineOrder := readFromChannel(la)
		if routineOrder == nil {
			break // Exit if the channel is closed
		}
		// Clear screen by printing new lines
		fmt.Print("\033[H\033[2J") // ANSI escape sequence to clear the screen
		fmt.Println("Real-Time Log Aggregation")

		// Display completed log entries first
		for _, id := range routineOrder {
			value, ok := la.logMap.Load(id)
			if !ok {
				continue
			}

			logEntry := value.(*LogEntry)

			// Display recent log lines with indentation
			logEntry.mu.Lock()
			if logEntry.isDone {
				elapsed := logEntry.endTime.Sub(logEntry.startTime)
				if !logEntry.isFailed {
					logEntry.messages = []string{} // Remove the "Done" message
					fmt.Printf("%s%s (Completed in %v)%s\n", green, id, elapsed, reset)
				} else {
					logEntry.messages = last5Messages(logEntry.messages[:len(logEntry.messages)])
					fmt.Printf("%s%s (Failed in in %v)%s\n", red, id, elapsed, reset)
				}
			} else {
				logEntry.mu.Unlock()
				continue
			}
			for _, msg := range logEntry.messages {
				fmt.Printf("   %s\n", msg)
			}
			logEntry.mu.Unlock()
		}

		// Display in-progress entries after completed ones
		for _, id := range routineOrder {
			value, ok := la.logMap.Load(id)
			if !ok {
				continue
			}

			logEntry := value.(*LogEntry)

			logEntry.mu.Lock()
			elapsed := time.Since(logEntry.startTime)

			if !logEntry.isDone {
				fmt.Printf("%s%s %v :%s\n", grayscale, id, elapsed, reset)
				for _, msg := range logEntry.messages {
					fmt.Printf("   %s\n", msg)
				}
			}
			logEntry.mu.Unlock()
		}
	}
}

func (la *LogAggregator) LogMessage(routineID string, msg string) {
	la.logMessage(routineID, msg, false, false)
}

func (la *LogAggregator) logMessage(routineID string, msg string, isDone bool, isFailed bool) {
	if la.format == "progress" {
		la.logChannel <- LogMessage{routineID: routineID, message: msg, isDone: isDone, isFailed: isFailed}
	} else {
		fmt.Printf("%s%s %s%s\n", grayscale, routineID, reset, msg)
	}
}

func (la *LogAggregator) Write(p []byte) (n int, err error) {
	msg := string(p)
	msg = strings.TrimSuffix(msg, "\n")
	la.logMessage("[engine-ci]", msg, false, false)
	return len(p), nil
}

func (la *LogAggregator) Copy(r io.ReadCloser) (n int, err error) {
	scanner := bufio.NewScanner(r)

	i := 0
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "errorDetail") {
			la.logMessage("[engine-ci]", line, true, true)
			return i, fmt.Errorf("errorDetail: %s", line)
		} else {
			la.logMessage("[engine-ci]", line, false, false)
		}
		i++
	}
	return i, nil
}

func (la *LogAggregator) SuccessMessage(routineID string, msg string) {
	la.logMessage(routineID, msg, true, false)
}

func (la *LogAggregator) FailedMessage(routineID string, msg string) {
	la.logMessage(routineID, msg, true, true)
}

// Flush will close the log channel and wait for all messages to be processed.
func (la *LogAggregator) Flush() {
	if la.format == "progress" {
		close(la.logChannel) // This will signal the flushing goroutine to finish
		// Wait for the display goroutine to signal completion
		<-la.flushDone
		close(la.flushDone)
		once.Reset()
	}
}
