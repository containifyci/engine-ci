# Fix Test Timeout Issue - Investigation Plan

## Problem Statement
- `make test` hangs and times out after 2 minutes
- Tests were passing individually but now there's a blocking test
- Need systematic identification and resolution of the hanging test

## Investigation Strategy

### Phase 1: Test Isolation and Identification (15 minutes)

#### 1.1 Package-Level Test Isolation
```bash
# Test each package individually with short timeout to identify problem area
go test -timeout=30s ./cmd/...
go test -timeout=30s ./internal/models/...
go test -timeout=30s ./internal/registry/...
go test -timeout=30s ./internal/storage/...
go test -timeout=30s ./internal/cache/...
go test -timeout=30s ./internal/runner/...
go test -timeout=30s ./test/integration/...
```

#### 1.2 Individual Test Identification
```bash
# Run with verbose output to see exactly where it hangs
go test -v -timeout=30s ./path/to/suspect/package

# List all tests and run one by one if needed
go test -list . ./suspect/package
go test -run ^TestSpecificTest$ -timeout=30s ./suspect/package
```

#### 1.3 Concurrency Analysis
```bash
# Check for race conditions and goroutine leaks
go test -race -timeout=30s ./...
go test -v -timeout=30s -count=1 ./... # Disable test caching
```

### Phase 2: Root Cause Analysis (20 minutes)

#### 2.1 Likely Culprits Based on Recent WebSocket Implementation

**WebSocket Tests:**
- Connection cleanup issues
- Channel deadlocks in subscriber management
- Goroutine leaks in message handling

**LogBuffer Tests:**
- Subscriber map access without proper locking
- WebSocket upgrade hanging
- Event loop not terminating

**Claude Manager Tests:**
- Process management not cleaning up
- Command execution hanging
- Signal handling issues

#### 2.2 Diagnostic Code Insertion
Add timeout and debug output to suspect tests:

```go
func TestSuspiciousFunction(t *testing.T) {
    // Add test timeout
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()
    
    // Add debug checkpoints
    t.Log("Starting test phase 1")
    // ... test code
    t.Log("Completed test phase 1")
    
    // Use context for operations
    select {
    case result := <-resultChan:
        // Handle result
    case <-ctx.Done():
        t.Fatal("Test timed out at specific phase")
    }
}
```

### Phase 3: Targeted Investigation (15 minutes)

#### 3.1 WebSocket-Specific Debugging
Check for:
- Unclosed WebSocket connections
- Goroutines waiting on channels that never receive
- HTTP server not shutting down properly
- Event loops not terminating

#### 3.2 Process Management Debugging
Check for:
- Child processes not being cleaned up
- Signal handlers not working
- Context cancellation not propagating
- Command execution hanging

#### 3.3 Channel and Goroutine Analysis
```bash
# Add GODEBUG for detailed goroutine info
GODEBUG=schedtrace=1000 go test -timeout=30s ./suspect/package

# Profile to see where goroutines are stuck
go test -timeout=30s -cpuprofile=cpu.prof -memprofile=mem.prof ./suspect/package
```

### Phase 4: Fix Implementation (20 minutes)

Based on root cause, implement one of these fixes:

#### 4.1 WebSocket Connection Cleanup
```go
func (s *Server) Shutdown() error {
    // Ensure all WebSocket connections are closed
    s.mu.Lock()
    for conn := range s.connections {
        conn.Close()
    }
    s.connections = make(map[*websocket.Conn]bool)
    s.mu.Unlock()
    
    return s.httpServer.Shutdown(context.Background())
}
```

#### 4.2 Goroutine Management
```go
func TestWithProperCleanup(t *testing.T) {
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel() // Ensure all goroutines receive cancellation
    
    var wg sync.WaitGroup
    
    wg.Add(1)
    go func() {
        defer wg.Done()
        select {
        case <-ctx.Done():
            return
        case <-someChan:
            // Handle
        }
    }()
    
    // Cleanup
    cancel()
    wg.Wait()
}
```

#### 4.3 Test Cleanup Patterns
```go
func setupTest(t *testing.T) func() {
    // Setup resources
    server := startTestServer()
    
    return func() {
        // Cleanup function
        server.Shutdown()
        // Wait for cleanup to complete
        time.Sleep(100 * time.Millisecond)
    }
}

func TestExample(t *testing.T) {
    cleanup := setupTest(t)
    defer cleanup()
    
    // Test implementation
}
```

### Phase 5: Verification (10 minutes)

#### 5.1 Comprehensive Test Run
```bash
# Test the fix multiple times to ensure consistency
for i in {1..5}; do
    echo "Test run $i"
    go test -timeout=30s ./...
    if [ $? -ne 0 ]; then
        echo "Test failed on run $i"
        break
    fi
done
```

#### 5.2 Resource Leak Detection
```bash
# Check for goroutine leaks
go test -v -timeout=30s ./... 2>&1 | grep -i "goroutine\|leak"

# Check for file descriptor leaks
lsof -p $$ | wc -l  # Before tests
go test ./...
lsof -p $$ | wc -l  # After tests
```

## Execution Checklist

- [ ] **Phase 1**: Run package-level isolation tests
- [ ] **Phase 1**: Identify hanging package/test
- [ ] **Phase 2**: Add diagnostic logging to suspect test
- [ ] **Phase 2**: Identify specific hanging point
- [ ] **Phase 3**: Analyze root cause (WebSocket/Process/Channel)
- [ ] **Phase 4**: Implement targeted fix
- [ ] **Phase 4**: Add proper cleanup to test
- [ ] **Phase 5**: Verify fix with multiple test runs
- [ ] **Phase 5**: Confirm no resource leaks

## Success Criteria

1. All tests complete within 30 seconds
2. No hanging or timeout issues
3. No goroutine leaks
4. All existing functionality preserved
5. `make fmt lint test` passes completely

## Risk Assessment

- **Low Risk**: Adding timeouts and debug logging
- **Medium Risk**: Modifying test cleanup logic
- **High Risk**: Changes to core WebSocket or process management

## Timeline

- **Total Estimated Time**: 80 minutes
- **Critical Path**: Identify hanging test → Root cause → Targeted fix
- **Fallback**: If complex, implement test-level timeouts as temporary fix

## Root Cause Analysis - IDENTIFIED

**Problem**: `TestWebSocketHandshake` in `internal/server/websocket_test.go` is hanging at line 536 in handlers.go

**Root Cause**: The test sets up a WebSocket handler with a mock connection, but the handler runs in a goroutine that waits indefinitely on `for logLine := range logChan`. The mock connection doesn't properly simulate connection closure, so the goroutine never exits.

**Specific Issue**: 
- Line 42: `server.handleWebSocketLogs(recorder, req)` starts the handler
- Handler reaches line 536: `for logLine := range logChan` and blocks forever
- Mock connection doesn't trigger cleanup/close, so channel never closes
- Test completes but goroutine keeps running until timeout

**Fix Strategy**: Modify the test to properly simulate connection lifecycle or add timeout context to the handler.

## Next Steps

1. ✅ **Phase 1**: Identified problematic test - `TestWebSocketHandshake` 
2. ✅ **Phase 2**: Root cause identified - goroutine blocking on channel range
3. ✅ **Phase 3**: Implement fix for test
4. ✅ **Phase 4**: Verify solution with comprehensive testing

## RESOLUTION - COMPLETED SUCCESSFULLY

**Problem Solved**: The test timeout issue has been completely resolved.

### What Was Fixed

1. **Primary Issue**: `TestWebSocketHandshake` was hanging indefinitely because the WebSocket handler goroutine was blocking on `for logLine := range logChan` with no way to exit.

2. **Race Condition**: The test had a data race between reading/writing the `hijacked` boolean field concurrently.

### Changes Made

#### 1. **Test Goroutine Management**
- Modified `TestWebSocketHandshake` to run the handler in a separate goroutine
- Added proper timeout handling (100ms) to allow handshake completion without infinite blocking
- Added graceful cleanup and verification logic

#### 2. **Race Condition Fix**
- Changed `hijacked` field from `bool` to `chan bool` to eliminate race conditions
- Added `isHijacked()` method for thread-safe checking
- Used buffered channel to prevent blocking during hijack operation

#### 3. **Improved Mock Connection**
- Enhanced `mockConn` to properly simulate connection lifecycle
- Added proper close/read behavior that follows real network connection patterns
- Implemented connection state management with channels

### Verification Results

✅ **Individual Test**: `TestWebSocketHandshake` passes consistently  
✅ **Package Tests**: All `internal/server` tests pass  
✅ **Race Detection**: No race conditions detected with `-race` flag  
✅ **Full Test Suite**: All tests pass with `make test`  
✅ **Performance**: Tests complete in <2 seconds (was timing out at 2 minutes)  

### Quality Impact

- **Zero Breaking Changes**: All existing functionality preserved
- **Thread Safety**: Eliminated race conditions in test code
- **Test Reliability**: Test now runs consistently and quickly
- **Maintainability**: Cleaner test patterns that can be reused

**Status**: ✅ COMPLETE - All tests pass, no timeouts, no race conditions