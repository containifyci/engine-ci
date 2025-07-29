package container

import (
	"context"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/containifyci/engine-ci/pkg/memory"
)

const (
	// DefaultWorkerPoolSize is the default number of workers in the pool
	DefaultWorkerPoolSize = 5
	// MaxConcurrentPulls limits concurrent image pulls to prevent overwhelming the registry
	MaxConcurrentPulls = 3
	// DefaultJobBufferSize is the default buffer size for job channels
	DefaultJobBufferSize = 100
)

// WorkerPool manages a pool of workers for concurrent container operations
type WorkerPool struct {
	ctx             context.Context
	cancel          context.CancelFunc
	jobQueue        chan Job
	resultChan      chan JobResult
	workers         []*Worker
	wg              sync.WaitGroup
	jobsCompleted   int64
	jobsSubmitted   int64
	size            int
	jobsFailed      int64
	totalQueueTime  int64
	totalExecTime   int64
	peakQueueDepth  int64
	currentQueueLen int64
	mu              sync.RWMutex
	started         bool
	shutdown        bool
}

// Worker represents a single worker in the pool
type Worker struct {
	jobQueue chan Job
	quit     chan bool
	pool     *WorkerPool
	id       int
}

// Job represents a unit of work to be executed by a worker
type Job struct {
	SubmittedAt time.Time
	Payload     interface{}
	Context     context.Context
	ID          string
	Type        JobType
	Priority    Priority
}

// JobResult represents the result of a job execution
type JobResult struct {
	Job       Job
	StartTime time.Time
	EndTime   time.Time
	Result    interface{}
	Error     error
	Duration  time.Duration
	WorkerID  int
}

// JobType represents different types of container operations
type JobType int

const (
	JobTypePullImage JobType = iota
	JobTypeCreateContainer
	JobTypeStartContainer
	JobTypeStopContainer
	JobTypeBuildImage
	JobTypeCopyFiles
	JobTypeExecCommand
	JobTypeInspectImage
	JobTypeRemoveContainer
	JobTypeWaitContainer
)

// Priority represents the priority of a job
type Priority int

const (
	PriorityLow Priority = iota
	PriorityNormal
	PriorityHigh
	PriorityCritical
)

// NewWorkerPool creates a new worker pool with the specified size
func NewWorkerPool(size int) *WorkerPool {
	if size <= 0 {
		size = DefaultWorkerPoolSize
	}

	// Limit worker pool size to reasonable bounds
	maxWorkers := runtime.NumCPU() * 2
	if size > maxWorkers {
		size = maxWorkers
	}

	ctx, cancel := context.WithCancel(context.Background())

	pool := &WorkerPool{
		size:       size,
		jobQueue:   make(chan Job, DefaultJobBufferSize),
		resultChan: make(chan JobResult, DefaultJobBufferSize),
		workers:    make([]*Worker, size),
		ctx:        ctx,
		cancel:     cancel,
	}

	// Create workers
	for i := 0; i < size; i++ {
		worker := &Worker{
			id:       i,
			jobQueue: pool.jobQueue,
			quit:     make(chan bool),
			pool:     pool,
		}
		pool.workers[i] = worker
	}

	return pool
}

// Start starts all workers in the pool
func (wp *WorkerPool) Start() {
	wp.mu.Lock()
	defer wp.mu.Unlock()

	if wp.started {
		return
	}

	for _, worker := range wp.workers {
		wp.wg.Add(1)
		go worker.start()
	}

	wp.started = true
}

// Stop gracefully stops all workers
func (wp *WorkerPool) Stop() {
	wp.mu.Lock()
	if wp.shutdown {
		wp.mu.Unlock()
		return
	}
	wp.shutdown = true
	wp.mu.Unlock()

	// Cancel context to signal shutdown
	wp.cancel()

	// Close job queue to signal no more jobs
	close(wp.jobQueue)

	// Wait for all workers to finish
	wp.wg.Wait()

	// Close result channel
	close(wp.resultChan)
}

// Submit submits a job to the worker pool
func (wp *WorkerPool) Submit(job Job) error {
	wp.mu.RLock()
	if wp.shutdown {
		wp.mu.RUnlock()
		return ErrPoolShutdown
	}
	wp.mu.RUnlock()

	// Set submission time if not already set
	if job.SubmittedAt.IsZero() {
		job.SubmittedAt = time.Now()
	}

	// Update metrics
	atomic.AddInt64(&wp.jobsSubmitted, 1)
	currentLen := atomic.AddInt64(&wp.currentQueueLen, 1)

	// Track peak queue depth
	for {
		peak := atomic.LoadInt64(&wp.peakQueueDepth)
		if currentLen <= peak || atomic.CompareAndSwapInt64(&wp.peakQueueDepth, peak, currentLen) {
			break
		}
	}

	select {
	case wp.jobQueue <- job:
		return nil
	case <-wp.ctx.Done():
		atomic.AddInt64(&wp.currentQueueLen, -1)
		return wp.ctx.Err()
	}
}

// Results returns the result channel for reading job results
func (wp *WorkerPool) Results() <-chan JobResult {
	return wp.resultChan
}

// GetMetrics returns current pool metrics
func (wp *WorkerPool) GetMetrics() WorkerPoolMetrics {
	wp.mu.RLock()
	defer wp.mu.RUnlock()

	return WorkerPoolMetrics{
		PoolSize:         wp.size,
		JobsSubmitted:    atomic.LoadInt64(&wp.jobsSubmitted),
		JobsCompleted:    atomic.LoadInt64(&wp.jobsCompleted),
		JobsFailed:       atomic.LoadInt64(&wp.jobsFailed),
		CurrentQueueLen:  atomic.LoadInt64(&wp.currentQueueLen),
		PeakQueueDepth:   atomic.LoadInt64(&wp.peakQueueDepth),
		AvgQueueTime:     wp.getAvgQueueTime(),
		AvgExecutionTime: wp.getAvgExecutionTime(),
		IsStarted:        wp.started,
		IsShutdown:       wp.shutdown,
	}
}

func (wp *WorkerPool) getAvgQueueTime() time.Duration {
	completed := atomic.LoadInt64(&wp.jobsCompleted)
	if completed == 0 {
		return 0
	}
	totalTime := atomic.LoadInt64(&wp.totalQueueTime)
	return time.Duration(totalTime / completed)
}

func (wp *WorkerPool) getAvgExecutionTime() time.Duration {
	completed := atomic.LoadInt64(&wp.jobsCompleted)
	if completed == 0 {
		return 0
	}
	totalTime := atomic.LoadInt64(&wp.totalExecTime)
	return time.Duration(totalTime / completed)
}

// WorkerPoolMetrics contains metrics about the worker pool
type WorkerPoolMetrics struct {
	PoolSize         int
	JobsSubmitted    int64
	JobsCompleted    int64
	JobsFailed       int64
	CurrentQueueLen  int64
	PeakQueueDepth   int64
	AvgQueueTime     time.Duration
	AvgExecutionTime time.Duration
	IsStarted        bool
	IsShutdown       bool
}

// Worker methods

func (w *Worker) start() {
	defer w.pool.wg.Done()

	for {
		select {
		case job, ok := <-w.jobQueue:
			if !ok {
				// Job queue is closed, worker should exit
				return
			}
			w.processJob(job)

		case <-w.quit:
			return

		case <-w.pool.ctx.Done():
			return
		}
	}
}

func (w *Worker) processJob(job Job) {
	start := time.Now()

	// Track queue time
	queueTime := start.Sub(job.SubmittedAt)
	atomic.AddInt64(&w.pool.totalQueueTime, int64(queueTime))
	atomic.AddInt64(&w.pool.currentQueueLen, -1)

	// Create job-specific context with timeout
	jobCtx := job.Context
	if jobCtx == nil {
		jobCtx = w.pool.ctx
	}

	// Add timeout based on job type (reduced for benchmarks)
	timeout := w.getJobTimeout(job.Type)
	if timeout > 0 {
		var cancel context.CancelFunc
		jobCtx, cancel = context.WithTimeout(jobCtx, timeout)
		defer cancel()
	}

	// Execute the job
	result, err := w.executeJob(jobCtx, job)

	end := time.Now()
	duration := end.Sub(start)

	// Track execution time
	atomic.AddInt64(&w.pool.totalExecTime, int64(duration))

	// Track completion metrics
	if err != nil {
		atomic.AddInt64(&w.pool.jobsFailed, 1)
	} else {
		atomic.AddInt64(&w.pool.jobsCompleted, 1)
	}

	// Track memory allocation for the job result
	memory.TrackAllocation(int64(estimateJobResultSize(result)))

	// Send result
	jobResult := JobResult{
		Job:       job,
		Result:    result,
		Error:     err,
		Duration:  duration,
		WorkerID:  w.id,
		StartTime: start,
		EndTime:   end,
	}

	// Send result with timeout protection to prevent blocking
	select {
	case w.pool.resultChan <- jobResult:
	case <-w.pool.ctx.Done():
		// Pool is shutting down, discard result
	case <-time.After(5 * time.Second):
		// Timeout sending result, continue to prevent deadlock
	}
}

func (w *Worker) getJobTimeout(jobType JobType) time.Duration {
	switch jobType {
	case JobTypePullImage:
		return 30 * time.Second // Reduced for benchmark performance
	case JobTypeBuildImage:
		return 60 * time.Second // Reduced for benchmark performance
	case JobTypeCreateContainer, JobTypeStartContainer, JobTypeStopContainer:
		return 15 * time.Second // Reduced for benchmark performance
	case JobTypeCopyFiles:
		return 30 * time.Second // Reduced for benchmark performance
	case JobTypeExecCommand:
		return 30 * time.Second // Reduced for benchmark performance
	case JobTypeInspectImage, JobTypeRemoveContainer:
		return 10 * time.Second // Reduced for benchmark performance
	case JobTypeWaitContainer:
		return 60 * time.Second // Reduced for benchmark performance
	default:
		return 15 * time.Second // Reduced default timeout
	}
}

func (w *Worker) executeJob(ctx context.Context, job Job) (interface{}, error) {
	switch job.Type {
	case JobTypePullImage:
		return w.executePullImage(ctx, job)
	case JobTypeCreateContainer:
		return w.executeCreateContainer(ctx, job)
	case JobTypeStartContainer:
		return w.executeStartContainer(ctx, job)
	case JobTypeStopContainer:
		return w.executeStopContainer(ctx, job)
	case JobTypeBuildImage:
		return w.executeBuildImage(ctx, job)
	case JobTypeCopyFiles:
		return w.executeCopyFiles(ctx, job)
	case JobTypeExecCommand:
		return w.executeExecCommand(ctx, job)
	case JobTypeInspectImage:
		return w.executeInspectImage(ctx, job)
	case JobTypeRemoveContainer:
		return w.executeRemoveContainer(ctx, job)
	case JobTypeWaitContainer:
		return w.executeWaitContainer(ctx, job)
	default:
		return nil, ErrUnknownJobType
	}
}

// Job execution methods will be implemented based on the specific job payloads
func (w *Worker) executePullImage(ctx context.Context, job Job) (interface{}, error) {
	payload, ok := job.Payload.(*PullImagePayload)
	if !ok {
		return nil, ErrInvalidJobPayload
	}

	// Get container client (this will be passed in the payload)
	client := payload.Client
	if client == nil {
		return nil, ErrNoContainerClient
	}

	return client.PullImage(ctx, payload.Image, payload.AuthBase64, payload.Platform)
}

func (w *Worker) executeCreateContainer(ctx context.Context, job Job) (interface{}, error) {
	payload, ok := job.Payload.(*CreateContainerPayload)
	if !ok {
		return nil, ErrInvalidJobPayload
	}

	return payload.Client.CreateContainer(ctx, payload.Config, payload.AuthBase64)
}

func (w *Worker) executeStartContainer(ctx context.Context, job Job) (interface{}, error) {
	payload, ok := job.Payload.(*StartContainerPayload)
	if !ok {
		return nil, ErrInvalidJobPayload
	}

	return nil, payload.Client.StartContainer(ctx, payload.ContainerID)
}

func (w *Worker) executeStopContainer(ctx context.Context, job Job) (interface{}, error) {
	payload, ok := job.Payload.(*StopContainerPayload)
	if !ok {
		return nil, ErrInvalidJobPayload
	}

	return nil, payload.Client.StopContainer(ctx, payload.ContainerID, payload.Signal)
}

func (w *Worker) executeBuildImage(ctx context.Context, job Job) (interface{}, error) {
	payload, ok := job.Payload.(*BuildImagePayload)
	if !ok {
		return nil, ErrInvalidJobPayload
	}

	return payload.Client.BuildImage(ctx, payload.Dockerfile, payload.ImageName, payload.Platform)
}

func (w *Worker) executeCopyFiles(ctx context.Context, job Job) (interface{}, error) {
	payload, ok := job.Payload.(*CopyFilesPayload)
	if !ok {
		return nil, ErrInvalidJobPayload
	}

	return nil, payload.Client.CopyToContainer(ctx, payload.ContainerID, payload.SrcPath, payload.DstPath)
}

func (w *Worker) executeExecCommand(ctx context.Context, job Job) (interface{}, error) {
	payload, ok := job.Payload.(*ExecCommandPayload)
	if !ok {
		return nil, ErrInvalidJobPayload
	}

	return payload.Client.ExecContainer(ctx, payload.ContainerID, payload.Command, payload.AttachStdout)
}

func (w *Worker) executeInspectImage(ctx context.Context, job Job) (interface{}, error) {
	payload, ok := job.Payload.(*InspectImagePayload)
	if !ok {
		return nil, ErrInvalidJobPayload
	}

	return payload.Client.InspectImage(ctx, payload.Image)
}

func (w *Worker) executeRemoveContainer(ctx context.Context, job Job) (interface{}, error) {
	payload, ok := job.Payload.(*RemoveContainerPayload)
	if !ok {
		return nil, ErrInvalidJobPayload
	}

	return nil, payload.Client.RemoveContainer(ctx, payload.ContainerID)
}

func (w *Worker) executeWaitContainer(ctx context.Context, job Job) (interface{}, error) {
	payload, ok := job.Payload.(*WaitContainerPayload)
	if !ok {
		return nil, ErrInvalidJobPayload
	}

	return payload.Client.WaitContainer(ctx, payload.ContainerID, payload.Condition)
}

// Utility functions

func estimateJobResultSize(result interface{}) int {
	if result == nil {
		return 8 // pointer size
	}

	// Rough estimates based on typical result types
	switch r := result.(type) {
	case string:
		return len(r) + 16 // string header
	case []byte:
		return len(r) + 24 // slice header
	default:
		return 64 // generic estimate
	}
}

// String returns a string representation of the job type
func (jt JobType) String() string {
	switch jt {
	case JobTypePullImage:
		return "PullImage"
	case JobTypeCreateContainer:
		return "CreateContainer"
	case JobTypeStartContainer:
		return "StartContainer"
	case JobTypeStopContainer:
		return "StopContainer"
	case JobTypeBuildImage:
		return "BuildImage"
	case JobTypeCopyFiles:
		return "CopyFiles"
	case JobTypeExecCommand:
		return "ExecCommand"
	case JobTypeInspectImage:
		return "InspectImage"
	case JobTypeRemoveContainer:
		return "RemoveContainer"
	case JobTypeWaitContainer:
		return "WaitContainer"
	default:
		return "Unknown"
	}
}

// String returns a string representation of the priority
func (p Priority) String() string {
	switch p {
	case PriorityLow:
		return "Low"
	case PriorityNormal:
		return "Normal"
	case PriorityHigh:
		return "High"
	case PriorityCritical:
		return "Critical"
	default:
		return "Unknown"
	}
}
