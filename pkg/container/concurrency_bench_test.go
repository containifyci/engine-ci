package container

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/containifyci/engine-ci/pkg/cri/types"
	"github.com/containifyci/engine-ci/pkg/memory"
)

// MockContainerManager for benchmarking
type MockContainerManager struct {
	delay        time.Duration
	failureRate  float64 // 0.0 to 1.0
	callCount    int64
	mu           sync.Mutex
	operationLog []string
}

func NewMockContainerManager(delay time.Duration, failureRate float64) *MockContainerManager {
	// Reduce delays in CI environment
	if isCI() {
		delay = delay / 10 // 10x faster in CI
		if delay < time.Millisecond {
			delay = time.Millisecond
		}
	}
	
	return &MockContainerManager{
		delay:       delay,
		failureRate: failureRate,
		callCount:   0,
	}
}

// isCI detects if we're running in a CI environment
func isCI() bool {
	// Check common CI environment variables
	ciVars := []string{"CI", "CONTINUOUS_INTEGRATION", "GITHUB_ACTIONS", "GITLAB_CI", "JENKINS_URL"}
	for _, env := range ciVars {
		if os.Getenv(env) != "" {
			return true
		}
	}
	return false
}

func (m *MockContainerManager) logOperation(op string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.operationLog = append(m.operationLog, fmt.Sprintf("%s@%v", op, time.Now().UnixNano()))
	m.callCount++
}

func (m *MockContainerManager) simulateWork() error {
	if m.delay > 0 {
		time.Sleep(m.delay)
	}

	// Simulate random failures
	if m.failureRate > 0 {
		m.mu.Lock()
		shouldFail := float64(m.callCount%100)/100.0 < m.failureRate
		m.mu.Unlock()

		if shouldFail {
			return fmt.Errorf("simulated failure")
		}
	}

	return nil
}

// Implement required interface methods for benchmarking
func (m *MockContainerManager) PullImage(ctx context.Context, image string, authBase64 string, platform string) (io.ReadCloser, error) {
	m.logOperation("PullImage")
	if err := m.simulateWork(); err != nil {
		return nil, err
	}
	return io.NopCloser(strings.NewReader(fmt.Sprintf("Pulling %s", image))), nil
}

func (m *MockContainerManager) ListImage(ctx context.Context, image string) ([]string, error) {
	m.logOperation("ListImage")
	if err := m.simulateWork(); err != nil {
		return nil, err
	}
	return []string{image}, nil
}

func (m *MockContainerManager) InspectImage(ctx context.Context, image string) (*types.ImageInfo, error) {
	m.logOperation("InspectImage")
	if err := m.simulateWork(); err != nil {
		return nil, err
	}
	platform := &types.PlatformSpec{OS: "linux", Architecture: "amd64"}
	return &types.ImageInfo{
		Platform: platform,
	}, nil
}

func (m *MockContainerManager) CreateContainer(ctx context.Context, opts *types.ContainerConfig, authBase64 string) (string, error) {
	m.logOperation("CreateContainer")
	if err := m.simulateWork(); err != nil {
		return "", err
	}
	return "container-id-" + opts.Name, nil
}

func (m *MockContainerManager) StartContainer(ctx context.Context, id string) error {
	m.logOperation("StartContainer")
	return m.simulateWork()
}

func (m *MockContainerManager) StopContainer(ctx context.Context, id string, signal string) error {
	m.logOperation("StopContainer")
	return m.simulateWork()
}

func (m *MockContainerManager) WaitContainer(ctx context.Context, id string, waitCondition string) (*int64, error) {
	m.logOperation("WaitContainer")
	if err := m.simulateWork(); err != nil {
		return nil, err
	}
	code := int64(0)
	return &code, nil
}

func (m *MockContainerManager) RemoveContainer(ctx context.Context, containerID string) error {
	m.logOperation("RemoveContainer")
	return m.simulateWork()
}

// Stub implementations for interface completeness
func (m *MockContainerManager) CommitContainer(ctx context.Context, containerID string, opts types.CommitOptions) (string, error) {
	return "image-id", nil
}
func (m *MockContainerManager) ContainerList(ctx context.Context, all bool) ([]*types.Container, error) {
	return nil, nil
}
func (m *MockContainerManager) ContainerLogs(ctx context.Context, id string, ShowStdout bool, ShowStderr bool, Follow bool) (io.ReadCloser, error) {
	return nil, nil
}
func (m *MockContainerManager) CopyContentToContainer(ctx context.Context, id, content, dest string) error {
	return nil
}
func (m *MockContainerManager) CopyDirectorToContainer(ctx context.Context, id, srcPath, dstPath string) error {
	return nil
}
func (m *MockContainerManager) CopyToContainer(ctx context.Context, id, srcPath, dstPath string) error {
	return nil
}
func (m *MockContainerManager) CopyFileFromContainer(ctx context.Context, id string, srcPath string) (string, error) {
	return "", nil
}
func (m *MockContainerManager) ExecContainer(ctx context.Context, id string, cmd []string, attachStdOut bool) (io.Reader, error) {
	return nil, nil
}
func (m *MockContainerManager) InspectContainer(ctx context.Context, id string) (*types.ContainerConfig, error) {
	return nil, nil
}
func (m *MockContainerManager) BuildImage(ctx context.Context, dockerfile []byte, imageName string, platform string) (io.ReadCloser, error) {
	return nil, nil
}
func (m *MockContainerManager) BuildMultiArchImage(ctx context.Context, dockerfile []byte, dockerCtx *bytes.Buffer, imageName string, platforms []string, authBase64 string) (io.ReadCloser, []string, error) {
	return nil, nil, nil
}
func (m *MockContainerManager) TagImage(ctx context.Context, source, target string) error {
	return nil
}
func (m *MockContainerManager) PushImage(ctx context.Context, target string, authBase64 string) (io.ReadCloser, error) {
	return nil, nil
}
func (m *MockContainerManager) RemoveImage(ctx context.Context, target string) error {
	return nil
}
func (m *MockContainerManager) Name() string {
	return "mock"
}

// Benchmark concurrent image pulling
func BenchmarkConcurrentImagePulling(b *testing.B) {
	// Reset memory tracking
	memory.ResetBufferPoolMetrics()
	memory.ResetPoolMetrics()

	testCases := []struct {
		name        string
		imageCount  int
		delay       time.Duration
		concurrency int
	}{
		{"3Images_1ms_Seq", 3, 1 * time.Millisecond, 1},
		{"3Images_1ms_Con2", 3, 1 * time.Millisecond, 2},
		{"5Images_1ms_Seq", 5, 1 * time.Millisecond, 1},
		{"5Images_1ms_Con3", 5, 1 * time.Millisecond, 3},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			// Create test images
			images := make([]ImagePullRequest, tc.imageCount)
			for i := 0; i < tc.imageCount; i++ {
				images[i] = ImagePullRequest{
					Image:      fmt.Sprintf("test-image-%d:latest", i),
					AuthBase64: "",
					Platform:   "linux/amd64",
					Priority:   PriorityNormal,
				}
			}

			mockClient := NewMockContainerManager(tc.delay, 0.0) // No failures for benchmarking

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				// Create context with timeout for benchmark safety
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()

				if tc.concurrency == 1 {
					// Sequential pulling
					for _, img := range images {
						_, err := mockClient.PullImage(ctx, img.Image, img.AuthBase64, img.Platform)
						if err != nil {
							b.Fatalf("Failed to pull image: %v", err)
						}
					}
				} else {
					// Concurrent pulling
					ccm := NewConcurrentContainerManager(mockClient, tc.concurrency)
					ccm.Start()
					defer ccm.Stop()

					results := ccm.PullImagesParallel(ctx, images)
					// Properly consume all results with timeout protection
					count := 0
					for result := range results {
						_ = result // Consume result
						count++
						if count >= len(images) {
							break
						}
					}
				}
			}

			// Report metrics
			bufferMetrics := memory.GetBufferPoolMetrics()
			poolMetrics := memory.GetPoolMetrics()

			b.ReportMetric(float64(mockClient.callCount)/float64(b.N), "calls/op")
			b.ReportMetric(bufferMetrics.HitRate()*100, "buffer_hit_%")
			b.ReportMetric(poolMetrics.HitRate()*100, "pool_hit_%")
		})
	}
}

// Benchmark worker pool performance
func BenchmarkWorkerPool(b *testing.B) {
	testCases := []struct {
		name    string
		workers int
		jobs    int
		jobTime time.Duration
	}{
		{"1Worker_5Jobs_1ms", 1, 5, 1 * time.Millisecond},
		{"2Workers_5Jobs_1ms", 2, 5, 1 * time.Millisecond},
		{"3Workers_10Jobs_1ms", 3, 10, 1 * time.Millisecond},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			mockClient := NewMockContainerManager(tc.jobTime, 0.0)

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				// Create context with timeout for benchmark safety
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()

				pool := NewWorkerPool(tc.workers)
				pool.Start()
				defer pool.Stop()

				// Submit jobs
				for j := 0; j < tc.jobs; j++ {
					job := Job{
						ID:      fmt.Sprintf("job-%d", j),
						Type:    JobTypePullImage,
						Context: ctx,
						Payload: &PullImagePayload{
							Client: mockClient,
							Image:  fmt.Sprintf("image-%d", j),
						},
					}

					if err := pool.Submit(job); err != nil {
						b.Fatalf("Failed to submit job: %v", err)
					}
				}

				// Collect results with timeout protection
				resultsCollected := 0
				for resultsCollected < tc.jobs {
					select {
					case <-pool.Results():
						resultsCollected++
					case <-ctx.Done():
						b.Fatalf("Timeout waiting for results")
					}
				}

				// Report metrics for this run
				metrics := pool.GetMetrics()
				b.ReportMetric(float64(metrics.JobsCompleted), "jobs_completed")
				b.ReportMetric(float64(metrics.JobsFailed), "jobs_failed")
			}
		})
	}
}

// Benchmark batch image operations
func BenchmarkBatchImageOperations(b *testing.B) {
	testCases := []struct {
		name          string
		imageCount    int
		concurrency   int
		operationTime time.Duration
	}{
		{"5Images_1Worker_1ms", 5, 1, 1 * time.Millisecond},
		{"5Images_2Workers_1ms", 5, 2, 1 * time.Millisecond},
		{"10Images_3Workers_1ms", 10, 3, 1 * time.Millisecond},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			mockClient := NewMockContainerManager(tc.operationTime, 0.0)
			batchOps := NewBatchImageOperations(mockClient, tc.concurrency)

			// Create test images
			images := make([]string, tc.imageCount)
			for i := 0; i < tc.imageCount; i++ {
				images[i] = fmt.Sprintf("test-image-%d:latest", i)
			}

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				// Create context with timeout for benchmark safety
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()

				// Test image existence checking
				results := batchOps.CheckImagesExistParallel(ctx, images)
				// Properly consume all results with count check
				count := 0
				for result := range results {
					_ = result // Consume result
					count++
					if count >= len(images) {
						break
					}
				}
			}

			b.ReportMetric(float64(mockClient.callCount)/float64(b.N), "calls/op")
		})
	}
}

// Benchmark semaphore performance
func BenchmarkSemaphore(b *testing.B) {
	testCases := []struct {
		name     string
		capacity int
		workers  int
	}{
		{"Cap1_1Worker", 1, 1},
		{"Cap2_2Workers", 2, 2},
		{"Cap3_5Workers", 3, 5},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			sem := NewSemaphore(tc.capacity)

			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					// Create context with timeout for benchmark safety
					ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
					if err := sem.Acquire(ctx); err != nil {
						cancel()
						if err == context.DeadlineExceeded {
							return // Skip this iteration on timeout
						}
						b.Fatalf("Failed to acquire semaphore: %v", err)
					}
					cancel()

					// Minimal work simulation
					// time.Sleep removed to speed up benchmark

					sem.Release()
				}
			})
		})
	}
}

// Benchmark container lifecycle operations
func BenchmarkContainerLifecycle(b *testing.B) {
	testCases := []struct {
		name        string
		containers  int
		concurrency int
		delay       time.Duration
	}{
		{"3Containers_Sequential_1ms", 3, 1, 1 * time.Millisecond},
		{"3Containers_Concurrent2_1ms", 3, 2, 1 * time.Millisecond},
		{"5Containers_Concurrent3_1ms", 5, 3, 1 * time.Millisecond},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			mockClient := NewMockContainerManager(tc.delay, 0.0)

			// Create test container requests
			createRequests := make([]ContainerCreateRequest, tc.containers)
			containerIDs := make([]string, tc.containers)
			stopRequests := make([]ContainerStopRequest, tc.containers)

			for i := 0; i < tc.containers; i++ {
				createRequests[i] = ContainerCreateRequest{
					Config: &types.ContainerConfig{
						Name:  fmt.Sprintf("test-container-%d", i),
						Image: fmt.Sprintf("test-image-%d", i),
					},
				}
				containerIDs[i] = fmt.Sprintf("container-id-%d", i)
				stopRequests[i] = ContainerStopRequest{
					ContainerID: containerIDs[i],
					Signal:      "SIGTERM",
				}
			}

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				// Create context with timeout for benchmark safety
				ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
				defer cancel()

				if tc.concurrency == 1 {
					// Sequential operations
					for j := 0; j < tc.containers; j++ {
						// Create
						_, err := mockClient.CreateContainer(ctx, createRequests[j].Config, "")
						if err != nil {
							b.Fatalf("Failed to create container: %v", err)
						}

						// Start
						err = mockClient.StartContainer(ctx, containerIDs[j])
						if err != nil {
							b.Fatalf("Failed to start container: %v", err)
						}

						// Stop
						err = mockClient.StopContainer(ctx, containerIDs[j], "SIGTERM")
						if err != nil {
							b.Fatalf("Failed to stop container: %v", err)
						}
					}
				} else {
					// Concurrent operations
					ccm := NewConcurrentContainerManager(mockClient, tc.concurrency)
					ccm.Start()
					defer ccm.Stop()

					// Create containers
					createResults := ccm.CreateContainersParallel(ctx, createRequests)
					count := 0
					for result := range createResults {
						_ = result // Consume result
						count++
						if count >= len(createRequests) {
							break
						}
					}

					// Start containers
					startResults := ccm.StartContainersParallel(ctx, containerIDs)
					count = 0
					for result := range startResults {
						_ = result // Consume result
						count++
						if count >= len(containerIDs) {
							break
						}
					}

					// Stop containers
					stopResults := ccm.StopContainersParallel(ctx, stopRequests)
					count = 0
					for result := range stopResults {
						_ = result // Consume result
						count++
						if count >= len(stopRequests) {
							break
						}
					}
				}
			}

			b.ReportMetric(float64(mockClient.callCount)/float64(b.N), "calls/op")
		})
	}
}
