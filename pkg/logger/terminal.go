package logger

import (
	"bufio"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/containifyci/engine-ci/pkg/memory"
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
	// LogEntry represents a single log entry with optimized memory usage and aligned fields
	LogEntry struct {
		// 64-bit aligned fields first for optimal memory layout
		startTime time.Time
		endTime   time.Time
		messages  []string // Pre-allocated slice for messages
		mu        sync.Mutex

		// Boolean fields grouped at end to minimize padding
		isDone   bool
		isFailed bool
	}

	// LogAggregator manages log aggregation with memory optimization and concurrency
	LogAggregator struct {
		entryPool      sync.Pool
		messagePool    sync.Pool
		shutdown       chan struct{}
		flushDone      chan struct{}
		logChannel     chan LogMessage
		batchProcessor *BatchProcessor
		logMap         sync.Map
		format         string
		routineOrder   []string
		workerWg       sync.WaitGroup
		batchTimeout   time.Duration
		batchSize      int
		processWorkers int
	}

	// LogMessage represents a single log message with optimized layout
	LogMessage struct {
		routineID string
		message   string
		// Boolean fields grouped together
		isDone   bool
		isFailed bool
	}

	// BatchProcessor handles batched log message processing for better performance
	BatchProcessor struct {
		inputChan    chan LogMessage
		processChan  chan []LogMessage
		aggregator   *LogAggregator
		batchSize    int
		batchTimeout time.Duration
		mu           sync.Mutex
		shutdown     bool
	}
)

// Singleton instance
var instance *LogAggregator
var once ResettableOnce

// addMessage adds a message to the log entry with memory optimization
func (le *LogEntry) addMessage(msg string) {
	le.mu.Lock()
	defer le.mu.Unlock()

	if len(le.messages) >= maxLogLines {
		// Maintain fixed size by removing the oldest entry
		// Use efficient slice operation to avoid reallocation
		if maxLogLines > 1 {
			copy(le.messages, le.messages[1:])
			le.messages = le.messages[:maxLogLines-1]
		} else {
			le.messages = le.messages[:0]
		}
	}

	le.messages = append(le.messages, msg)

	// Track memory allocation for the message more accurately
	// Account for both the string content and slice overhead
	stringMemory := int64(len(msg))
	sliceOverhead := int64(8) // Approximate pointer size in slice
	memory.TrackAllocation(stringMemory + sliceOverhead)
}

// NewLogAggregator returns the singleton instance of LogAggregator with memory optimization and concurrency
func NewLogAggregator(format string) *LogAggregator {
	once.Do(func() {
		instance = &LogAggregator{
			logChannel:     make(chan LogMessage, 1000), // Buffered channel for better performance
			flushDone:      make(chan struct{}),
			format:         format,
			processWorkers: 2,                     // Number of worker goroutines for processing
			batchSize:      10,                    // Process messages in batches
			batchTimeout:   50 * time.Millisecond, // Maximum time to wait for a batch
			shutdown:       make(chan struct{}),
		}

		// Initialize memory pool for LogEntry reuse
		instance.entryPool = sync.Pool{
			New: func() interface{} {
				return &LogEntry{
					messages: make([]string, 0, maxLogLines), // Pre-allocate with capacity
				}
			},
		}

		// Initialize message pool for string slice reuse
		instance.messagePool = sync.Pool{
			New: func() interface{} {
				return make([]string, 0, maxLogLines)
			},
		}

		// Initialize batch processor for concurrent message handling
		if format == "progress" {
			instance.batchProcessor = &BatchProcessor{
				batchSize:    instance.batchSize,
				batchTimeout: instance.batchTimeout,
				inputChan:    instance.logChannel,
				processChan:  make(chan []LogMessage, 100),
				aggregator:   instance,
			}

			// Start batch processor
			go instance.batchProcessor.start()
			go instance.startLogDisplayConcurrent()
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
		// Signal shutdown to batch processor
		close(la.shutdown)

		// Stop batch processor
		if la.batchProcessor != nil {
			la.batchProcessor.stop()
		}

		close(la.logChannel) // This will signal the flushing goroutine to finish
		// Wait for the display goroutine to signal completion
		<-la.flushDone
		close(la.flushDone)
		once.Reset()
	}
}

// BatchProcessor methods

// start begins the batch processing loop
func (bp *BatchProcessor) start() {
	batch := make([]LogMessage, 0, bp.batchSize)
	timer := time.NewTimer(bp.batchTimeout)
	timer.Stop() // Stop initially

	for {
		select {
		case msg, ok := <-bp.inputChan:
			if !ok {
				// Channel closed, flush remaining batch
				if len(batch) > 0 {
					bp.processChan <- batch
				}
				close(bp.processChan)
				return
			}

			batch = append(batch, msg)

			// Start timer if this is the first message in the batch
			if len(batch) == 1 {
				timer.Reset(bp.batchTimeout)
			}

			// Send batch if it's full
			if len(batch) >= bp.batchSize {
				bp.processChan <- batch
				batch = make([]LogMessage, 0, bp.batchSize)
				timer.Stop()
			}

		case <-timer.C:
			// Timeout reached, send current batch
			if len(batch) > 0 {
				bp.processChan <- batch
				batch = make([]LogMessage, 0, bp.batchSize)
			}

		case <-bp.aggregator.shutdown:
			// Shutdown signal, flush remaining batch
			if len(batch) > 0 {
				bp.processChan <- batch
			}
			close(bp.processChan)
			return
		}
	}
}

// stop stops the batch processor
func (bp *BatchProcessor) stop() {
	bp.mu.Lock()
	defer bp.mu.Unlock()
	bp.shutdown = true
}

// startLogDisplayConcurrent starts the concurrent log display system
func (la *LogAggregator) startLogDisplayConcurrent() {
	fmt.Println("Starting Real-Time Log Aggregation with Concurrency...")

	// Start worker goroutines for processing batches
	for i := 0; i < la.processWorkers; i++ {
		la.workerWg.Add(1)
		go la.processBatchWorker(i)
	}

	// Start display goroutine
	go la.displayLoop()

	// Wait for all workers to finish
	la.workerWg.Wait()
}

// processBatchWorker processes batches of log messages
func (la *LogAggregator) processBatchWorker(workerID int) {
	defer la.workerWg.Done()

	for batch := range la.batchProcessor.processChan {
		la.processBatch(batch, workerID)
	}
}

// processBatch processes a batch of log messages efficiently
func (la *LogAggregator) processBatch(batch []LogMessage, workerID int) {
	routineOrderMap := make(map[string]bool)

	for _, logMsg := range batch {
		// Process message similar to readFromChannel but in batch
		var logEntry *LogEntry
		if entry, loaded := la.logMap.Load(logMsg.routineID); loaded {
			logEntry = entry.(*LogEntry)
		} else {
			pooledEntry := la.entryPool.Get().(*LogEntry)
			pooledEntry.startTime = time.Now()
			pooledEntry.endTime = time.Time{}
			pooledEntry.messages = pooledEntry.messages[:0]
			pooledEntry.isDone = false
			pooledEntry.isFailed = false

			if actualEntry, entryLoaded := la.logMap.LoadOrStore(logMsg.routineID, pooledEntry); entryLoaded {
				la.entryPool.Put(pooledEntry)
				logEntry = actualEntry.(*LogEntry)
			} else {
				logEntry = pooledEntry
				memory.TrackBufferReuse()
			}
		}

		logEntry.addMessage(logMsg.message)

		if logMsg.isDone {
			logEntry.mu.Lock()
			logEntry.isDone = logMsg.isDone
			logEntry.isFailed = logMsg.isFailed
			logEntry.endTime = time.Now()
			logEntry.mu.Unlock()
		}

		routineOrderMap[logMsg.routineID] = true
	}

	// Update routine order (this needs to be synchronized)
	// For now, we'll use a simple approach but this could be optimized further
	for routineID := range routineOrderMap {
		found := false
		for _, existing := range la.routineOrder {
			if existing == routineID {
				found = true
				break
			}
		}
		if !found {
			la.routineOrder = append(la.routineOrder, routineID)
		}
	}
}

// displayLoop handles the display updates in a separate goroutine
func (la *LogAggregator) displayLoop() {
	ticker := time.NewTicker(100 * time.Millisecond) // Update display every 100ms
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			la.updateDisplay()
		case <-la.shutdown:
			la.updateDisplay() // Final update
			la.flushDone <- struct{}{}
			return
		}
	}
}

// updateDisplay updates the terminal display
func (la *LogAggregator) updateDisplay() {
	// Clear screen by printing new lines
	fmt.Print("\033[H\033[2J") // ANSI escape sequence to clear the screen
	fmt.Println("Real-Time Log Aggregation (Concurrent)")

	// Display completed log entries first
	for _, id := range la.routineOrder {
		value, ok := la.logMap.Load(id)
		if !ok {
			continue
		}

		logEntry := value.(*LogEntry)
		logEntry.mu.Lock()

		if logEntry.isDone {
			elapsed := logEntry.endTime.Sub(logEntry.startTime)
			if !logEntry.isFailed {
				logEntry.messages = []string{} // Remove the "Done" message
				fmt.Printf("%s%s (Completed in %v)%s\n", green, id, elapsed, reset)
			} else {
				displayMessages := last5Messages(logEntry.messages)
				fmt.Printf("%s%s (Failed in %v)%s\n", red, id, elapsed, reset)
				for _, msg := range displayMessages {
					fmt.Printf("   %s\n", msg)
				}
			}
		}

		logEntry.mu.Unlock()
	}

	// Display in-progress entries after completed ones
	for _, id := range la.routineOrder {
		value, ok := la.logMap.Load(id)
		if !ok {
			continue
		}

		logEntry := value.(*LogEntry)
		logEntry.mu.Lock()

		if !logEntry.isDone {
			elapsed := time.Since(logEntry.startTime)
			fmt.Printf("%s%s %v :%s\n", grayscale, id, elapsed, reset)
			for _, msg := range logEntry.messages {
				fmt.Printf("   %s\n", msg)
			}
		}

		logEntry.mu.Unlock()
	}
}
