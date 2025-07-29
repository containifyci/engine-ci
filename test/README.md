# Engine-CI Test Suite

This directory contains comprehensive tests for the engine-ci project's new architecture, focusing on the LanguageBuilder interface system and centralized configuration.

## Test Organization

### Integration Tests
- **pkg/builder/integration_test.go** - Builder system integration tests
- **pkg/config/integration_test.go** - Configuration system integration tests  
- **pkg/golang/integration_test.go** - Golang-specific integration tests

### End-to-End Tests
- **test/e2e/build_workflows_test.go** - Complete build workflow testing
- **test/e2e/backward_compatibility_test.go** - Backward compatibility validation

### Performance Tests
- **test/performance/regression_test.go** - Performance regression testing

### Test Fixtures
- **test/fixtures/sample_configs.yaml** - Sample configuration files for testing
- **test/fixtures/projects/sample-go-app/** - Sample Go project for testing

## Running Tests

### All Tests
```bash
# Run all tests
go test ./...

# Run tests with verbose output
go test -v ./...

# Run tests with coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Integration Tests Only
```bash
# Run all integration tests
go test ./pkg/builder ./pkg/config ./pkg/golang -run "Integration"

# Run specific integration test suite
go test ./pkg/builder -run "TestBuilderIntegration"
go test ./pkg/config -run "TestConfigurationIntegration"
go test ./pkg/golang -run "TestGoBuilderIntegration"
```

### End-to-End Tests
```bash
# Run E2E tests (requires more setup)
go test ./test/e2e

# Run specific E2E test suite
go test ./test/e2e -run "TestBuildWorkflows"
go test ./test/e2e -run "TestBackwardCompatibility"
```

### Performance Tests
```bash
# Run performance regression tests
go test ./test/performance

# Run benchmarks
go test -bench=. ./test/performance
go test -bench=BenchmarkConfigurationOperations ./test/performance
go test -bench=BenchmarkBuilderOperations ./test/performance
```

### Short Tests (Skip Long-Running Tests)
```bash
# Skip long-running tests (useful for CI)
go test -short ./...
```

## Test Categories

### 1. Builder System Integration Tests
Tests the complete LanguageBuilder interface system:
- Interface compliance for all builders
- Factory system registration and creation
- BaseBuilder common functionality
- Common utilities and shared functions
- Builder registry operations
- Performance and concurrency testing

**Key Test Files:**
- `pkg/builder/integration_test.go`

**Coverage Areas:**
- LanguageBuilder interface implementations
- Factory pattern with builder registration
- BaseBuilder functionality and configuration
- Common utilities like ComputeChecksum
- Performance benchmarks and memory usage

### 2. Configuration System Integration Tests
Tests the centralized configuration system:
- Hierarchical configuration loading (CLI > env > file > defaults)
- Environment variable processing (280+ variables)
- Configuration validation and error handling
- Thread safety and concurrent access
- Backward compatibility with existing methods

**Key Test Files:**
- `pkg/config/integration_test.go`

**Coverage Areas:**
- Configuration hierarchy and precedence
- Environment variable parsing and validation
- Configuration file loading (YAML)
- Global configuration management
- Configuration merging and defaults
- Performance and memory usage

### 3. Golang Package Integration Tests
Tests Golang-specific functionality with new architecture:
- All variant support (Alpine, Debian, DebianCGO)
- Build operations (Build, Lint, Prod, Pull)
- Configuration integration
- Container system integration
- Factory pattern implementation

**Key Test Files:**
- `pkg/golang/integration_test.go`

**Coverage Areas:**
- Variant-specific functionality
- Configuration integration
- Container build integration
- Factory system usage
- Backward compatibility functions
- Performance and concurrency

### 4. End-to-End Workflow Tests
Tests complete build workflows:
- Complete Golang build workflows
- Configuration-driven builds
- Multi-variant workflows
- Factory-based workflows
- Environment integration

**Key Test Files:**
- `test/e2e/build_workflows_test.go`

**Coverage Areas:**
- Complete build workflows from start to finish
- Configuration file integration
- Multi-variant build support
- Factory pattern usage in real workflows
- Environment variable integration

### 5. Backward Compatibility Tests
Ensures existing APIs continue to work:
- Legacy Golang package functions
- Existing container workflows
- Configuration methods
- Builder functions and patterns

**Key Test Files:**
- `test/e2e/backward_compatibility_test.go`

**Coverage Areas:**
- Legacy golang.New(), golang.NewDebian(), golang.NewCGO()
- Legacy utility functions
- Existing container.Build integration
- Configuration compatibility
- Performance comparison with new architecture

### 6. Performance Regression Tests
Validates performance hasn't degraded:
- Configuration loading performance
- Builder creation performance
- Method call performance
- Memory usage validation
- Concurrency performance

**Key Test Files:**
- `test/performance/regression_test.go`

**Coverage Areas:**
- Performance baselines and thresholds
- Memory usage and leak detection
- Concurrent operation performance
- Scalability testing

## Test Data and Fixtures

### Configuration Fixtures
The `test/fixtures/sample_configs.yaml` file contains various configuration examples:
- **development** - Local development configuration
- **production** - Production-ready configuration with security
- **ci** - CI/CD build configuration
- **minimal** - Minimal configuration for testing
- **edge_case** - Edge cases and unusual values

### Project Fixtures
The `test/fixtures/projects/sample-go-app/` directory contains a complete Go project:
- `go.mod` - Go module definition with dependencies
- `main.go` - Sample CLI application using Cobra
- `main_test.go` - Comprehensive tests using testify

## Performance Baselines

The test suite includes performance baselines to prevent regressions:

### Configuration Performance
- Default config loading: < 10ms
- Environment config loading: < 50ms
- File config loading: < 20ms
- Config validation: < 5ms

### Builder Performance
- Builder creation: < 5ms
- Factory creation: < 1ms
- Method calls: < 1μs
- Intermediate image caching: < 100ns (cached)

### Memory Usage
- Configuration memory: < 100KB per config
- Builder memory: < 50KB per builder
- Memory leak detection: < 1MB increase over 1000 operations

## CI/CD Integration

### GitHub Actions Integration
Tests are designed to work in GitHub Actions environments:
- No external dependencies required for unit/integration tests
- Docker-dependent tests are skipped in CI (marked with build tags)
- Performance tests have appropriate timeouts
- Coverage reports are generated

### Make Target Integration
Tests integrate with existing build system:
```bash
# Run tests as part of quality gates
make test

# Run tests with formatting and linting
make fmt lint test

# Generate coverage reports
make test-coverage
```

## Test Best Practices

### Test Structure
- Use table-driven tests for multiple scenarios
- Use testify for assertions and mocking
- Create temporary directories for file system tests
- Mock external dependencies for reliability

### Test Isolation
- Tests are independent and can run in parallel
- Use t.TempDir() for temporary directories
- Clean up resources in defer statements
- Avoid global state modifications

### Error Testing
- Test both positive and negative cases
- Validate error messages are helpful
- Test edge cases and boundary conditions
- Ensure graceful degradation

### Performance Testing
- Include benchmarks for critical paths
- Test concurrent access patterns
- Monitor memory usage and leaks
- Validate against performance baselines

## Troubleshooting

### Common Issues

#### Test Failures in CI
- Ensure tests don't depend on local file system paths
- Use appropriate timeouts for CI environments
- Mock external dependencies
- Check for race conditions in concurrent tests

#### Performance Test Failures
- Performance tests may be sensitive to system load
- Run multiple times to get consistent results
- Adjust thresholds for different environments
- Monitor for memory leaks in long-running tests

#### Configuration Test Failures
- Environment variables may persist between tests
- Use defer statements to clean up environment
- Test configuration hierarchy carefully
- Validate file permissions for config files

### Debugging Tips

#### Verbose Test Output
```bash
# Run with verbose output to see detailed logs
go test -v ./...

# Run specific test with debug output
go test -v ./pkg/config -run "TestConfigurationHierarchy"
```

#### Test Coverage Analysis
```bash
# Generate detailed coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Check coverage for specific packages
go test -coverprofile=config_coverage.out ./pkg/config
go tool cover -func=config_coverage.out
```

#### Memory Profiling
```bash
# Run tests with memory profiling
go test -memprofile=mem.prof ./test/performance
go tool pprof mem.prof
```

#### Race Condition Detection
```bash
# Run tests with race detection
go test -race ./...
```

## Contributing to Tests

### Adding New Tests
1. Follow existing test patterns and naming conventions
2. Use testify for assertions and mocking
3. Include both positive and negative test cases
4. Add performance tests for new critical paths
5. Update this documentation

### Test Naming Conventions
- Test functions: `TestFeatureName`
- Subtests: `t.Run("specific_case", func(t *testing.T) {...})`
- Benchmarks: `BenchmarkOperationName`
- Examples: `ExampleFunctionName`

### Test Documentation
- Add comments explaining complex test logic
- Document test data and fixtures
- Explain performance expectations
- Update README when adding new test categories

This comprehensive test suite ensures the new architecture is robust, performant, and maintains backward compatibility while providing excellent test coverage and preventing regressions.