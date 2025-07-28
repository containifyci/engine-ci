package container

import (
	"context"
	"errors"
	"io"
	"sync"
	"time"

	"github.com/containifyci/engine-ci/pkg/cri"
	"github.com/containifyci/engine-ci/pkg/cri/types"
)

// Error definitions for concurrency operations
var (
	ErrPoolShutdown      = errors.New("worker pool is shutdown")
	ErrUnknownJobType    = errors.New("unknown job type")
	ErrInvalidJobPayload = errors.New("invalid job payload")
	ErrNoContainerClient = errors.New("no container client provided")
	ErrJobTimeout        = errors.New("job execution timeout")
	ErrTooManyRetries    = errors.New("maximum retries exceeded")
)

// Job payload types for different container operations

// PullImagePayload contains data for image pull operations
type PullImagePayload struct {
	Client     cri.ContainerManager
	Image      string
	AuthBase64 string
	Platform   string
}

// CreateContainerPayload contains data for container creation
type CreateContainerPayload struct {
	Client     cri.ContainerManager
	Config     *types.ContainerConfig
	AuthBase64 string
}

// StartContainerPayload contains data for starting containers
type StartContainerPayload struct {
	Client      cri.ContainerManager
	ContainerID string
}

// StopContainerPayload contains data for stopping containers
type StopContainerPayload struct {
	Client      cri.ContainerManager
	ContainerID string
	Signal      string
}

// BuildImagePayload contains data for image building
type BuildImagePayload struct {
	Client     cri.ContainerManager
	Dockerfile []byte
	ImageName  string
	Platform   string
}

// CopyFilesPayload contains data for file copy operations
type CopyFilesPayload struct {
	Client      cri.ContainerManager
	ContainerID string
	SrcPath     string
	DstPath     string
}

// ExecCommandPayload contains data for command execution
type ExecCommandPayload struct {
	Client       cri.ContainerManager
	ContainerID  string
	Command      []string
	AttachStdout bool
}

// InspectImagePayload contains data for image inspection
type InspectImagePayload struct {
	Client cri.ContainerManager
	Image  string
}

// RemoveContainerPayload contains data for container removal
type RemoveContainerPayload struct {
	Client      cri.ContainerManager
	ContainerID string
}

// WaitContainerPayload contains data for waiting on containers
type WaitContainerPayload struct {
	Client      cri.ContainerManager
	ContainerID string
	Condition   string
}

// ConcurrentImagePuller manages concurrent image pulling with rate limiting
type ConcurrentImagePuller struct {
	pool       *WorkerPool
	semaphore  chan struct{}
	maxRetries int
	retryDelay time.Duration
}

// NewConcurrentImagePuller creates a new concurrent image puller
func NewConcurrentImagePuller(maxConcurrent int, maxRetries int) *ConcurrentImagePuller {
	if maxConcurrent <= 0 {
		maxConcurrent = MaxConcurrentPulls
	}
	if maxRetries <= 0 {
		maxRetries = 3
	}

	return &ConcurrentImagePuller{
		pool:       NewWorkerPool(maxConcurrent),
		semaphore:  make(chan struct{}, maxConcurrent),
		maxRetries: maxRetries,
		retryDelay: 2 * time.Second,
	}
}

// Start starts the concurrent image puller
func (cip *ConcurrentImagePuller) Start() {
	cip.pool.Start()
}

// Stop stops the concurrent image puller
func (cip *ConcurrentImagePuller) Stop() {
	cip.pool.Stop()
}

// PullImages pulls multiple images concurrently with retry logic
func (cip *ConcurrentImagePuller) PullImages(ctx context.Context, client cri.ContainerManager, images []ImagePullRequest) error {
	if len(images) == 0 {
		return nil
	}

	// Use WaitGroup to wait for all pulls to complete
	var wg sync.WaitGroup
	errorChan := make(chan error, len(images))

	for _, img := range images {
		wg.Add(1)

		job := Job{
			ID:       "pull-" + img.Image,
			Type:     JobTypePullImage,
			Priority: img.Priority,
			Context:  ctx,
			Payload: &PullImagePayload{
				Client:     client,
				Image:      img.Image,
				AuthBase64: img.AuthBase64,
				Platform:   img.Platform,
			},
		}

		if err := cip.pool.Submit(job); err != nil {
			wg.Done()
			errorChan <- err
			continue
		}

		// Handle results asynchronously with timeout protection
		go func(expectedJobID string) {
			defer wg.Done()

			// Create timeout context for result waiting
			timeout := time.NewTimer(30 * time.Second)
			defer timeout.Stop()

			// Wait for this specific job's result with timeout
			for {
				select {
				case result, ok := <-cip.pool.Results():
					if !ok {
						// Channel closed, return
						return
					}
					if result.Job.ID == expectedJobID {
						if result.Error != nil {
							errorChan <- result.Error
						}
						return
					}
				case <-timeout.C:
					// Timeout waiting for result
					errorChan <- ErrJobTimeout
					return
				case <-ctx.Done():
					// Context cancelled
					errorChan <- ctx.Err()
					return
				}
			}
		}(job.ID)
	}

	// Wait for all pulls to complete
	go func() {
		wg.Wait()
		close(errorChan)
	}()

	// Collect any errors
	var errors []error
	for err := range errorChan {
		if err != nil {
			errors = append(errors, err)
		}
	}

	if len(errors) > 0 {
		return errors[0] // Return first error for simplicity
	}

	return nil
}

// ImagePullRequest represents a request to pull an image
type ImagePullRequest struct {
	Image      string
	AuthBase64 string
	Platform   string
	Priority   Priority
	Retries    int
}

// ConcurrentContainerManager wraps container operations with concurrency support
type ConcurrentContainerManager struct {
	client     cri.ContainerManager
	pool       *WorkerPool
	pullSem    chan struct{} // Semaphore for pull operations
	buildSem   chan struct{} // Semaphore for build operations
	maxRetries int
}

// NewConcurrentContainerManager creates a new concurrent container manager
func NewConcurrentContainerManager(client cri.ContainerManager, poolSize int) *ConcurrentContainerManager {
	if poolSize <= 0 {
		poolSize = DefaultWorkerPoolSize
	}

	return &ConcurrentContainerManager{
		client:     client,
		pool:       NewWorkerPool(poolSize),
		pullSem:    make(chan struct{}, MaxConcurrentPulls),
		buildSem:   make(chan struct{}, 2), // Limit builds to 2 concurrent
		maxRetries: 3,
	}
}

// Start starts the concurrent container manager
func (ccm *ConcurrentContainerManager) Start() {
	ccm.pool.Start()
}

// Stop stops the concurrent container manager
func (ccm *ConcurrentContainerManager) Stop() {
	ccm.pool.Stop()
}

// PullImagesParallel pulls multiple images in parallel with rate limiting
func (ccm *ConcurrentContainerManager) PullImagesParallel(ctx context.Context, requests []ImagePullRequest) <-chan ImagePullResult {
	resultChan := make(chan ImagePullResult, len(requests))

	go func() {
		defer close(resultChan)

		var wg sync.WaitGroup
		for _, req := range requests {
			wg.Add(1)

			go func(request ImagePullRequest) {
				defer wg.Done()

				// Acquire semaphore for rate limiting
				select {
				case ccm.pullSem <- struct{}{}:
					defer func() { <-ccm.pullSem }()
				case <-ctx.Done():
					resultChan <- ImagePullResult{
						Image: request.Image,
						Error: ctx.Err(),
					}
					return
				}

				// Pull the image with retries
				var reader io.ReadCloser
				var err error

				for attempt := 0; attempt <= ccm.maxRetries; attempt++ {
					reader, err = ccm.client.PullImage(ctx, request.Image, request.AuthBase64, request.Platform)
					if err == nil {
						break
					}

					if attempt < ccm.maxRetries {
						select {
						case <-time.After(time.Duration(attempt+1) * time.Second):
							// Continue to next attempt
						case <-ctx.Done():
							err = ctx.Err()
							break
						}
					}
				}

				resultChan <- ImagePullResult{
					Image:  request.Image,
					Reader: reader,
					Error:  err,
				}
			}(req)
		}

		wg.Wait()
	}()

	return resultChan
}

// CreateContainersParallel creates multiple containers in parallel
func (ccm *ConcurrentContainerManager) CreateContainersParallel(ctx context.Context, requests []ContainerCreateRequest) <-chan ContainerCreateResult {
	resultChan := make(chan ContainerCreateResult, len(requests))

	go func() {
		defer close(resultChan)

		var wg sync.WaitGroup
		for _, req := range requests {
			wg.Add(1)

			go func(request ContainerCreateRequest) {
				defer wg.Done()

				containerID, err := ccm.client.CreateContainer(ctx, request.Config, request.AuthBase64)

				resultChan <- ContainerCreateResult{
					Name:        request.Config.Name,
					ContainerID: containerID,
					Error:       err,
				}
			}(req)
		}

		wg.Wait()
	}()

	return resultChan
}

// StartContainersParallel starts multiple containers in parallel
func (ccm *ConcurrentContainerManager) StartContainersParallel(ctx context.Context, containerIDs []string) <-chan ContainerOperationResult {
	resultChan := make(chan ContainerOperationResult, len(containerIDs))

	go func() {
		defer close(resultChan)

		var wg sync.WaitGroup
		for _, id := range containerIDs {
			wg.Add(1)

			go func(containerID string) {
				defer wg.Done()

				err := ccm.client.StartContainer(ctx, containerID)

				resultChan <- ContainerOperationResult{
					ContainerID: containerID,
					Error:       err,
				}
			}(id)
		}

		wg.Wait()
	}()

	return resultChan
}

// StopContainersParallel stops multiple containers in parallel
func (ccm *ConcurrentContainerManager) StopContainersParallel(ctx context.Context, requests []ContainerStopRequest) <-chan ContainerOperationResult {
	resultChan := make(chan ContainerOperationResult, len(requests))

	go func() {
		defer close(resultChan)

		var wg sync.WaitGroup
		for _, req := range requests {
			wg.Add(1)

			go func(request ContainerStopRequest) {
				defer wg.Done()

				err := ccm.client.StopContainer(ctx, request.ContainerID, request.Signal)

				resultChan <- ContainerOperationResult{
					ContainerID: request.ContainerID,
					Error:       err,
				}
			}(req)
		}

		wg.Wait()
	}()

	return resultChan
}

// WaitContainersParallel waits for multiple containers in parallel
func (ccm *ConcurrentContainerManager) WaitContainersParallel(ctx context.Context, requests []ContainerWaitRequest) <-chan ContainerWaitResult {
	resultChan := make(chan ContainerWaitResult, len(requests))

	go func() {
		defer close(resultChan)

		var wg sync.WaitGroup
		for _, req := range requests {
			wg.Add(1)

			go func(request ContainerWaitRequest) {
				defer wg.Done()

				statusCode, err := ccm.client.WaitContainer(ctx, request.ContainerID, request.Condition)

				resultChan <- ContainerWaitResult{
					ContainerID: request.ContainerID,
					StatusCode:  statusCode,
					Error:       err,
				}
			}(req)
		}

		wg.Wait()
	}()

	return resultChan
}

// Result types for concurrent operations

// ImagePullResult represents the result of an image pull operation
type ImagePullResult struct {
	Image  string
	Reader io.ReadCloser
	Error  error
}

// ContainerCreateRequest represents a request to create a container
type ContainerCreateRequest struct {
	Config     *types.ContainerConfig
	AuthBase64 string
}

// ContainerCreateResult represents the result of a container creation
type ContainerCreateResult struct {
	Name        string
	ContainerID string
	Error       error
}

// ContainerStopRequest represents a request to stop a container
type ContainerStopRequest struct {
	ContainerID string
	Signal      string
}

// ContainerOperationResult represents the result of a container operation
type ContainerOperationResult struct {
	ContainerID string
	Error       error
}

// ContainerWaitRequest represents a request to wait for a container
type ContainerWaitRequest struct {
	ContainerID string
	Condition   string
}

// ContainerWaitResult represents the result of waiting for a container
type ContainerWaitResult struct {
	ContainerID string
	StatusCode  *int64
	Error       error
}

// BatchImageOperations provides batch operations for images
type BatchImageOperations struct {
	client cri.ContainerManager
	sem    chan struct{}
}

// NewBatchImageOperations creates a new batch image operations manager
func NewBatchImageOperations(client cri.ContainerManager, maxConcurrent int) *BatchImageOperations {
	if maxConcurrent <= 0 {
		maxConcurrent = DefaultWorkerPoolSize
	}

	return &BatchImageOperations{
		client: client,
		sem:    make(chan struct{}, maxConcurrent),
	}
}

// InspectImagesParallel inspects multiple images in parallel
func (bio *BatchImageOperations) InspectImagesParallel(ctx context.Context, images []string) <-chan ImageInspectResult {
	resultChan := make(chan ImageInspectResult, len(images))

	go func() {
		defer close(resultChan)

		var wg sync.WaitGroup
		for _, img := range images {
			wg.Add(1)

			go func(image string) {
				defer wg.Done()

				// Acquire semaphore for rate limiting
				select {
				case bio.sem <- struct{}{}:
					defer func() { <-bio.sem }()
				case <-ctx.Done():
					resultChan <- ImageInspectResult{
						Image: image,
						Error: ctx.Err(),
					}
					return
				}

				info, err := bio.client.InspectImage(ctx, image)

				resultChan <- ImageInspectResult{
					Image: image,
					Info:  info,
					Error: err,
				}
			}(img)
		}

		wg.Wait()
	}()

	return resultChan
}

// CheckImagesExistParallel checks if multiple images exist in parallel
func (bio *BatchImageOperations) CheckImagesExistParallel(ctx context.Context, images []string) <-chan ImageExistsResult {
	resultChan := make(chan ImageExistsResult, len(images))

	go func() {
		defer close(resultChan)

		var wg sync.WaitGroup
		for _, img := range images {
			wg.Add(1)

			go func(image string) {
				defer wg.Done()

				// Acquire semaphore for rate limiting
				select {
				case bio.sem <- struct{}{}:
					defer func() { <-bio.sem }()
				case <-ctx.Done():
					resultChan <- ImageExistsResult{
						Image: image,
						Error: ctx.Err(),
					}
					return
				}

				images, err := bio.client.ListImage(ctx, image)
				exists := len(images) > 0 && err == nil

				resultChan <- ImageExistsResult{
					Image:  image,
					Exists: exists,
					Error:  err,
				}
			}(img)
		}

		wg.Wait()
	}()

	return resultChan
}

// ImageInspectResult represents the result of an image inspection
type ImageInspectResult struct {
	Image string
	Info  *types.ImageInfo
	Error error
}

// ImageExistsResult represents the result of checking if an image exists
type ImageExistsResult struct {
	Image  string
	Exists bool
	Error  error
}

// Semaphore provides a simple semaphore implementation for rate limiting
type Semaphore struct {
	ch chan struct{}
}

// NewSemaphore creates a new semaphore with the given capacity
func NewSemaphore(capacity int) *Semaphore {
	return &Semaphore{
		ch: make(chan struct{}, capacity),
	}
}

// Acquire acquires a permit from the semaphore
func (s *Semaphore) Acquire(ctx context.Context) error {
	select {
	case s.ch <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Release releases a permit back to the semaphore
func (s *Semaphore) Release() {
	select {
	case <-s.ch:
	default:
		// Should not happen if used correctly
	}
}

// TryAcquire tries to acquire a permit without blocking
func (s *Semaphore) TryAcquire() bool {
	select {
	case s.ch <- struct{}{}:
		return true
	default:
		return false
	}
}

// Available returns the number of available permits
func (s *Semaphore) Available() int {
	return cap(s.ch) - len(s.ch)
}
