# Fix Containerized Linting Issue

## Problem Analysis
The containerized linting process fails with the error:
```
WARN [runner] Can't run linter goanalysis_metalinter: inspect: failed to load package : could not load export data: no export data for "github.com/containifyci/engine-ci/protos2"
ERRO Running error: can't run linter goanalysis_metalinter
```

## Root Cause Investigation
1. **Local vs Container Environment**: Local builds work because the `replace` directive in go.mod points to `./protos2`
2. **Module Resolution**: The containerized linter can't find export data for the local protos2 package
3. **Build Order**: The protos2 package may not be built before the linter runs
4. **Cache/Module Issues**: The container may not have proper module cache setup

## Solution Strategy

### Option 1: Build Dependencies First (Recommended)
- Ensure protos2 is built before linting runs
- Add explicit build step for local dependencies in container environment
- Modify the linting script to build local modules first

### Option 2: Module Cache Fix
- Ensure proper `go mod download` and `go mod tidy` in containers
- Fix module cache mounting and permissions
- Ensure replace directive works in container environment

### Option 3: Container Environment Enhancement
- Enhance the container build process to handle local dependencies
- Add pre-linting build steps for local modules

## Implementation Plan

### Phase 1: Analyze Current Linting Process
- [x] Examine golangci-lint configuration (.golangci.yml) - Uses v2 config
- [x] Check containerized linting implementation (alpine/golang.go) - Uses v2.1.2 image
- [x] Understand how local dependencies are handled - Replace directive points to ./protos2
- [x] Identify where the linting fails - In goanalysis_metalinter during export data loading

### Phase 2: Fix Module Resolution
- [x] Modify linting script to build local dependencies first
- [x] Ensure proper module resolution in container environment
- [x] Add build steps for protos2 before linting
- [x] Updated tests to match new script format

### Phase 3: Test and Validate
- [ ] Test locally with containerized linting
- [ ] Verify CI/CD pipeline works
- [ ] Ensure all linting passes without export data errors

## Technical Details

### Current Linting Flow
1. Container pulls `golangci/golangci-lint:v2.1.2` image
2. Mounts source code to `/src`
3. Runs linting script with build tags `containers_image_openpgp`
4. **FAILS**: Can't load export data for local protos2 package

### Proposed Fix
1. Before running linter, build local dependencies:
   ```bash
   cd protos2 && go build ./...
   cd .. && go mod download
   ```
2. Ensure module cache is properly populated
3. Run linter with proper export data available

### Files to Modify
- `pkg/golang/alpine/golangcilint.go` - Add pre-build steps
- Possibly `pkg/golang/alpine/golang.go` - Modify Lint() method

## Implementation Summary

### Changes Made
1. **Modified `pkg/golang/alpine/golangcilint.go`**:
   - Enhanced `LintScript()` method to build local dependencies before linting
   - Added `go mod download` and `go mod tidy` steps
   - Added explicit build step for `protos2` module 
   - Added build step for all modules to generate export data
   - Added proper logging and error handling

2. **Updated Tests**:
   - Modified `pkg/golang/alpine/golangcilint_test.go` to match new script format
   - Updated `TestCopyLintScript` and `TestCopyLintScriptGCL` expectations
   - All tests pass successfully

### Technical Solution
The fix addresses the root cause by ensuring that:
1. Local dependencies (especially `protos2`) are built before linting
2. Module cache is properly populated with `go mod download` and `go mod tidy`
3. Export data is generated for all local packages before the linter runs
4. The linter has access to all necessary package information

### Script Changes
The new linting script now includes these pre-linting steps:
```bash
# Build local dependencies first to generate export data
echo "Building local dependencies..."
go mod download
go mod tidy

# Build protos2 module to ensure export data is available
if [ -d "protos2" ]; then
    echo "Building protos2 module..."
    cd protos2
    go build ./...
    cd ..
fi

# Ensure all local modules are built
echo "Building all modules to generate export data..."
go build ./...

# Now run the linter with properly built dependencies
echo "Running linter..."
golangci-lint -v run --build-tags containers_image_openpgp --timeout=5m
```

### Validation
- [x] All unit tests pass
- [x] Engine builds successfully with changes
- [x] Binary runs correctly
- [ ] Containerized linting to be tested in CI/CD

## Next Steps
1. Commit changes and create PR
2. Test in CI/CD environment 
3. Monitor containerized builds for resolution of export data errors
4. Validate that linting completes successfully in containers