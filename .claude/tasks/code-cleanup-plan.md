# Code Cleanup Plan - Unused & Redundant Code Removal

## Overview
This plan outlines a systematic approach to identify and remove unused code, redundant implementations, and overly complex tests that don't provide meaningful coverage. We'll work package by package to maintain a manageable context.

## Methodology

### 1. Detection Strategy
- **Unused Code**: Use `staticcheck` and manual analysis to find:
  - Unexported functions/types with no internal references
  - Dead code paths
  - Unused struct fields
  - Unreachable code
  
- **Redundant Code**: Look for:
  - Duplicate implementations
  - Similar functions that could be consolidated
  - Wrapper functions that add no value
  
- **Test Complexity**: Identify tests that:
  - Have high cyclomatic complexity but test trivial cases
  - Use excessive mocking without testing real behavior
  - Are flaky or environment-dependent
  - Test implementation details rather than behavior

### 2. Package Analysis Order

Working from least dependent to most dependent packages:

#### Phase 1: Utility Packages (Low Dependencies)
1. **pkg/memory** - New package, likely minimal unused code
2. **pkg/logger** - Logging utilities
3. **pkg/utils** - General utilities
4. **pkg/filesystem** - File operations

#### Phase 2: Core Infrastructure (Medium Dependencies)
5. **pkg/kv** - Key-value store
6. **pkg/network** - Network utilities
7. **pkg/svc** - Service utilities
8. **pkg/build** - Build configurations

#### Phase 3: Container Runtime (High Dependencies)
9. **pkg/cri** - Container runtime interface
10. **pkg/container** - Core container management

#### Phase 4: Language-Specific (Independent)
11. **pkg/golang** - Go-specific builds
12. **pkg/python** - Python builds
13. **pkg/maven** - Java/Maven builds
14. **pkg/protobuf** - Protocol buffer support

#### Phase 5: Integration & Tools
15. **pkg/github** - GitHub integration
16. **pkg/gcloud** - Google Cloud integration
17. **pkg/trivy** - Security scanning
18. **cmd** - CLI commands

## Package-Specific Analysis Plans

### pkg/memory (Phase 1)
**Focus Areas**:
- Check if all pool types are actually used
- Verify metrics collection is utilized
- Look for redundant buffer size categories

**Expected Findings**:
- Possibly unused metric fields
- Redundant pool size configurations

### pkg/logger (Phase 1)
**Focus Areas**:
- Custom slog handler implementation necessity
- Terminal logger complexity vs actual usage
- Benchmark tests value

**Expected Findings**:
- Overly complex terminal formatting
- Unused log levels or fields
- Redundant benchmark scenarios

### pkg/container (Phase 3)
**Focus Areas**:
- Unused container options
- Redundant error types
- Complex wait/retry logic
- Overly detailed benchmark tests

**Expected Findings**:
- Legacy container configurations
- Duplicate error handling paths
- Unused PushOption fields
- Complex tests for simple scenarios

### pkg/cri (Phase 3)
**Focus Areas**:
- Interface methods actually used
- Platform-specific code usage
- Mock implementations necessity

**Expected Findings**:
- Unused CRI interface methods
- Platform code for unsupported systems
- Over-mocked test scenarios

## Test Cleanup Criteria

### Tests to Remove/Simplify:
1. **Trivial Getters/Setters**: Tests that only verify simple property access
2. **Over-Mocked Units**: Tests with more mock setup than actual testing
3. **Implementation Tests**: Tests tied to internal implementation details
4. **Redundant Scenarios**: Multiple tests covering the same code path
5. **Complex Table Tests**: Table-driven tests with only 1-2 cases

### Tests to Keep:
1. **Integration Tests**: Real component interaction tests
2. **Edge Cases**: Boundary condition and error handling
3. **Regression Tests**: Tests for previously found bugs
4. **Performance Benchmarks**: Actually used for optimization decisions

## Implementation Strategy

### For Each Package:

1. **Static Analysis**
   ```bash
   staticcheck ./pkg/[package]/...
   go vet ./pkg/[package]/...
   ```

2. **Usage Analysis**
   - Search for function/type references across codebase
   - Check if exported functions are used externally
   - Verify interface implementations are complete

3. **Test Review**
   - Calculate actual code coverage vs complexity
   - Identify tests with high setup/assertion ratio
   - Find tests that break with minor refactoring

4. **Documentation**
   - Document why code is being removed
   - Note any potential breaking changes
   - Update package documentation

## Success Metrics

- **Code Reduction**: Target 10-20% reduction in LOC
- **Test Efficiency**: Increase test value/complexity ratio
- **Build Time**: Reduce test execution time by 15-30%
- **Maintainability**: Simpler codebase with clearer purpose

## Risk Mitigation

1. **Breaking Changes**: 
   - Check all internal usage before removing exported functions
   - Verify no external projects depend on removed code

2. **Test Coverage**:
   - Ensure meaningful coverage remains after cleanup
   - Don't remove tests just for coverage percentage

3. **Performance**:
   - Keep performance-critical optimizations
   - Maintain benchmarks that guide optimization

## Next Steps

1. Start with Phase 1 packages (memory, logger)
2. Create per-package analysis reports
3. Submit incremental PRs for review
4. Track metrics before/after cleanup
5. Document lessons learned for future maintenance

---

This plan provides a structured approach to code cleanup while minimizing risk and maintaining code quality. Each package will be analyzed independently with specific findings documented before any changes are made.