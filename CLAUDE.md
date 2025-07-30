# General Layer

# Prompt Registry - Claude Instructions

## Plan & Review

### Before starting work
- Write a plan to .claude/tasks/TASK_NAME.md.
- The plan should be a detailed implementation plan and the reasoning behind them, as well as tasks broken down.
- Don't over plan it, always think MVP.
- Once you write the plan, firstly ask me to review it. Do not continue until I approve the plan.

### While implementing
- You should update the plan as you work.
- After you complete tasks in the plan, you should update and append detailed descriptions of the changes you made, so following tasks can be easily hand over to other engineers.

## Project Context
Go CLI tool for managing, versioning, and fetching LLM instruction prompts with centralized storage, semantic versioning, and validation capabilities.

**Tech Stack**: Go 1.24, Cobra CLI, testify testing, filesystem storage, clean architecture

## Git & PR Workflow
```bash
# Branch naming: <username>_<feature_description> (underscores)
git checkout -b fr12k_new_feature
git commit -m "feat(scope): description" # Conventional commits
make fmt lint test # Always before committing
git push -u origin fr12k_new_feature

# PR creation (use temp files due to quoting issues)
echo "PR title" > /tmp/pr_title.txt
cat > /tmp/pr_body.md << EOF
PR description content
EOF
gh pr create --fill-first

# Copilot review
gh copilot-review <PR_URL>
gh pr view <number> --comments # Check reviews
gh pr comment <number> --body-file /tmp/response.txt # Respond via temp file
```

## IMPORTANT INSTRUCTIONS - DO NOT DELETE THIS SECTION

### GitHub CLI Limitations
- **CRITICAL**: Always use temp files for `gh pr create` and `gh pr comment` commands due to shell quoting issues with long strings
- **Never** provide PR titles or bodies directly as command arguments
- **Always** use `--body-file` and `--title "$(cat /tmp/file.txt)"` patterns

### Required Workflow Pattern
```bash
# CORRECT - Use temp files
echo "PR title here" > /tmp/pr_title.txt
cat > /tmp/pr_body.md << 'EOF'
PR description content here
EOF
gh pr create --title "$(cat /tmp/pr_title.txt)" --body-file /tmp/pr_body.md
gh pr comment <number> --body-file /tmp/response.txt

# WRONG - Direct arguments (will fail with quoting errors)
gh pr create --title "long title with spaces" --body "long body text"
```

### Error Handling Requirements  
- **Always** check return values from `os.Chdir()` in deferred functions to satisfy linter
- **Always** use package constants instead of hardcoded strings for maintainability
- **Always** run `make fmt lint test` before committing

### Architecture Documentation
- **Update architecture.d2**: When making significant structural changes, update the D2Lang architecture file
- **Regenerate diagram**: Run `d2 architecture.d2 architecture.png` after architecture updates
- **Keep docs in sync**: Ensure README.md reflects current architecture and features

### Testing Practices
- Use temp directories with proper cleanup for file system tests
- Avoid complex mocking for interactive CLI features - integration tests are sufficient
- Focus unit tests on business logic, use integration tests for user workflows

### CRITICAL: Container-Based Build System Lessons Learned

#### **Integration Testing Protocol**
- **MANDATORY**: Always run `go run --tags containers_image_openpgp main.go run -t all` before committing any changes
- The build system runs inside multiple Docker containers - changes that work locally may break in container environment
- Docker-in-Docker limitations cause test failures that aren't apparent in local development

#### **Test Environment Constraints**
```bash
# REQUIRED: Full integration test before any commit
go run --tags containers_image_openpgp main.go run -t all

# This catches issues that local testing misses:
# - Container volume mounting problems
# - Build script generation failures  
# - Docker execution context issues
# - Interface compatibility problems
```

#### **Known Test Limitations**
- **Unit tests that create Docker containers WILL FAIL** in CI environment
- Tests calling `NewGoBuilder()` or similar container-creating functions cause panics
- Performance tests requiring container runtime access must be removed
- Integration tests that create actual containers are incompatible with container-based CI

#### **Test File Patterns to Avoid**
```go
// ❌ These patterns cause panics in container CI:
func TestNewGoBuilder(t *testing.T) {
    builder, err := NewGoBuilder(build, VariantAlpine) // Creates container - FAILS
}

func TestContainerOperations(t *testing.T) {
    container := NewContainer(...) // Creates container - FAILS  
}

// ✅ Safe alternatives:
func TestBuilderLogic(t *testing.T) {
    // Test pure logic without container creation
    result := CalculateBuildScript(config)
    assert.NotEmpty(t, result)
}
```

#### **Regression Prevention**
1. **Interface Changes**: Major refactoring can break Build interface compatibility
2. **Volume Mounting**: Changes to volume mounting logic affect container execution 
3. **Build Script Generation**: Script generation failures appear as "No build script defined"
4. **Test Removals**: Removing tests may break compilation if helper functions are also removed

#### **Emergency Fixes Applied**
- Removed all tests creating Docker containers (`pkg/golang/builder_test.go`)
- Added `createTestBuild()` helper back to `integration_test.go` for compilation
- Fixed Build interface compatibility by adding missing `Run()` and `IsAsync()` methods
- Optimized struct field alignment for memory efficiency (golangci-lint fieldalignment)

#### **Architecture Safety**
- **Build Interface**: All builders must implement `Run()`, `IsAsync()`, and `Build()` methods
- **Volume Strategy**: Always mount project root, not subfolders (containers cd as needed)
- **Factory Pattern**: Factories must initialize all required maps before use
- **Container Cleanup**: Proper container lifecycle management is critical

---

# General + Claude Layer

# Prompt Registry - Claude Instructions

## Plan & Review

### Before starting work
- Write a plan to .claude/tasks/TASK_NAME.md.
- The plan should be a detailed implementation plan and the reasoning behind them, as well as tasks broken down.
- Don't over plan it, always think MVP.
- Once you write the plan, firstly ask me to review it. Do not continue until I approve the plan.

### While implementing
- You should update the plan as you work.
- After you complete tasks in the plan, you should update and append detailed descriptions of the changes you made, so following tasks can be easily hand over to other engineers.

## Project Context
Go CLI tool for managing, versioning, and fetching LLM instruction prompts with centralized storage, semantic versioning, and validation capabilities.

**Tech Stack**: Go 1.24, Cobra CLI, testify testing, filesystem storage, clean architecture

## Git & PR Workflow
```bash
# Branch naming: <username>_<feature_description> (underscores)
git checkout -b fr12k_new_feature
git commit -m "feat(scope): description" # Conventional commits
make fmt lint test # Always before committing
git push -u origin fr12k_new_feature

# PR creation (use temp files due to quoting issues)
echo "PR title" > /tmp/pr_title.txt
cat > /tmp/pr_body.md << EOF
PR description content
EOF
gh pr create --fill-first

# Copilot review
gh copilot-review <PR_URL>
sleep 60 # Wait for review completion
gh pr view <number> --comments # Check review summary
```

## IMPORTANT INSTRUCTIONS - DO NOT DELETE THIS SECTION

### GitHub CLI Limitations
- **CRITICAL**: Always use temp files for `gh pr create` and `gh pr comment` commands due to shell quoting issues with long strings
- **Never** provide PR titles or bodies directly as command arguments
- **Always** use `--body-file` and `--title "$(cat /tmp/file.txt)"` patterns

### Required Workflow Pattern
```bash
# CORRECT - Use temp files
echo "PR title here" > /tmp/pr_title.txt
cat > /tmp/pr_body.md << 'EOF'
PR description content here
EOF
gh pr create --title "$(cat /tmp/pr_title.txt)" --body-file /tmp/pr_body.md
gh pr comment <number> --body-file /tmp/response.txt

# WRONG - Direct arguments (will fail with quoting errors)
gh pr create --title "long title with spaces" --body "long body text"
```

### Error Handling Requirements  
- **Always** check return values from `os.Chdir()` in deferred functions to satisfy linter
- **Always** use package constants instead of hardcoded strings for maintainability
- **Always** run `make fmt lint test` before committing

### Architecture Documentation
- **Update architecture.d2**: When making significant structural changes, update the D2Lang architecture file
- **Regenerate diagram**: Run `d2 architecture.d2 architecture.png` after architecture updates
- **Keep docs in sync**: Ensure README.md reflects current architecture and features

### Copilot Review Methodology
Copilot reviews provide progressive, multi-layered feedback that evolves as code quality improves.

#### **Complete Review Retrieval**
```bash
# Get comprehensive review data
gh pr view <number> --json comments,reviews

# Get ALL detailed findings (line-specific comments)
gh api repos/goflink/prompt-registry/pulls/<number>/comments
```

#### **Review Types & Behavior**
- **Progressive Feedback**: Copilot generates fresh reviews for new commits, building on previous fixes
- **Suppressed vs Visible**: Low-confidence findings are suppressed in summary but accessible via API
- **Multi-Review Evolution**: Each commit may trigger new review with different focus areas
- **Line-Specific Comments**: Actual findings are in individual line comments, not just the summary

#### **Response Workflow**
```bash
# Respond to review (always use temp files)
cat > /tmp/copilot_response.txt << 'EOF'
Thanks for the review! I've addressed the findings:
1. Issue description - Fix implemented
2. Issue description - Fix implemented
EOF
gh pr comment <number> --body-file /tmp/copilot_response.txt
```

#### **Review Analysis Pattern**
1. **Initial Reviews**: Focus on basic code quality (naming, constants, error handling)
2. **Follow-up Reviews**: Address advanced optimizations (performance, architecture)
3. **Confidence Levels**: High-confidence issues appear immediately, low-confidence suppressed
4. **Evolutionary Feedback**: Copilot provides increasingly sophisticated suggestions as code improves

### Testing Practices
- Use temp directories with proper cleanup for file system tests
- Avoid complex mocking for interactive CLI features - integration tests are sufficient
- Focus unit tests on business logic, use integration tests for user workflows

## Sub-Agent Usage Guidelines

The project has specialized sub-agents in `.claude/agents/` for parallel development. Use these agents when their expertise matches the task requirements:

### When to Use Sub-Agents

#### **go-developer** - Use for:
- Implementing new CLI commands in `cmd/` package
- Writing business logic in `internal/` packages  
- Refactoring Go code and improving architecture
- Adding new features to registry, runner, or cache layers
- Optimizing performance and memory usage

#### **test-engineer** - Use for:
- Writing unit tests for new functionality
- Creating integration tests for CLI workflows
- Improving test coverage and quality
- Debugging test failures and flaky tests
- Setting up test fixtures and utilities

#### **storage-architect** - Use for:
- Implementing new storage backends (S3, HTTP, etc.)
- Modifying storage interfaces and contracts
- Optimizing storage performance and caching
- Handling data migration and versioning
- Debugging storage-related issues

#### **github-integrator** - Use for:
- Creating and managing pull requests
- Setting up GitHub Actions workflows
- Managing Copilot reviews and responses
- Repository configuration and automation
- Release management and tagging

#### **documentation-maintainer** - Use for:
- Updating README.md and user documentation
- Maintaining architecture diagrams (architecture.d2)
- Creating usage examples and integration guides
- Writing API documentation and CLI references
- Updating project knowledge base

### Parallel Development Patterns

#### **Feature Development** (Use multiple agents)
```
1. go-developer: Implement core functionality
2. test-engineer: Write comprehensive tests (parallel)
3. documentation-maintainer: Update docs and examples (parallel)
4. github-integrator: Create PR when ready
```

#### **Storage Backend Addition** (Specialized focus)
```
1. storage-architect: Design and implement new backend
2. test-engineer: Create storage-specific tests
3. documentation-maintainer: Update storage documentation
4. go-developer: Integrate with CLI commands
```

#### **Bug Fixes** (Targeted approach)
```
1. Identify domain: Use appropriate specialist agent
2. test-engineer: Add regression tests first
3. Domain expert: Implement fix
4. github-integrator: Handle PR workflow
```

### Agent Coordination

- **Main conversation**: Coordinate between agents and handle high-level planning
- **Single responsibility**: Each agent focuses on their domain expertise
- **Cross-agent collaboration**: Agents reference each other's work when needed
- **Quality gates**: All agents must ensure `make fmt lint test` passes

### Task Assignment Examples

```bash
# Complex feature requiring multiple domains
"Add S3 storage backend with CLI integration"
→ storage-architect: Interface design and implementation
→ go-developer: CLI command integration  
→ test-engineer: Comprehensive testing
→ documentation-maintainer: User guides and examples

# Testing and quality focus
"Improve test coverage for registry package"
→ test-engineer: Primary responsibility
→ go-developer: Support with code understanding

# Documentation and architecture updates
"Update architecture diagram and README for new features"
→ documentation-maintainer: Primary responsibility
→ Coordinate with relevant domain expert for technical accuracy
```

### Best Practices
- Use sub-agents for their specialized domains, not general tasks
- Coordinate between agents when tasks span multiple domains
- Maintain consistent code quality across all agent contributions
- Ensure all agents follow the same Git workflow and testing requirements

---

# General + Golang Layer

# Golang Development Guidelines

Comprehensive guidance for Go development, best practices, and integration with the prompt-registry project.

## Core Go Philosophy

### Simplicity and Clarity
- **"Less is more"**: Prefer simple solutions over complex ones
- **Explicit over implicit**: Make dependencies and behavior clear
- **Readability counts**: Code is read more often than written
- **Composition over inheritance**: Use interfaces and embedding

### Go Way of Thinking
- **Do one thing well**: Functions and packages should have focused responsibilities  
- **Handle errors explicitly**: Don't ignore errors, handle them appropriately
- **Concurrency with communication**: Use goroutines and channels effectively
- **Start simple, add complexity when needed**: Begin with the simplest solution

## Language Best Practices

### Naming Conventions
```go
// ✅ Good naming
type UserService struct {
    db Database
    logger Logger
}

func (s *UserService) CreateUser(ctx context.Context, user User) error {
    // Implementation
}

// ❌ Avoid abbreviations and unclear names
type UsrSvc struct {
    d DB
    l Log  
}

func (s *UsrSvc) CrtUsr(c context.Context, u User) error {
    // Implementation
}
```

**Rules:**
- **Exported identifiers**: Use PascalCase (`UserService`, `CreateUser`)
- **Unexported identifiers**: Use camelCase (`userService`, `createUser`)
- **Package names**: Short, lowercase, no underscores (`http`, `json`, `user`)
- **Interface names**: Often end with `-er` (`Reader`, `Writer`, `Stringer`)
- **Be descriptive**: `userCount` not `uc`, `httpClient` not `hc`

### Error Handling Patterns

#### Standard Error Handling
```go
// ✅ Explicit error handling
func processUser(id string) (*User, error) {
    user, err := fetchUser(id)
    if err != nil {
        return nil, fmt.Errorf("failed to fetch user %s: %w", id, err)
    }
    
    if err := validateUser(user); err != nil {
        return nil, fmt.Errorf("user validation failed: %w", err)
    }
    
    return user, nil
}

// ✅ Early return pattern
func validateRequest(req *Request) error {
    if req == nil {
        return errors.New("request cannot be nil")
    }
    
    if req.ID == "" {
        return errors.New("request ID is required")
    }
    
    if req.Timestamp.IsZero() {
        return errors.New("request timestamp is required")
    }
    
    return nil
}
```

#### Custom Error Types
```go
// ✅ Custom errors for better error handling
type ValidationError struct {
    Field   string
    Message string
}

func (e ValidationError) Error() string {
    return fmt.Sprintf("validation error in field %s: %s", e.Field, e.Message)
}

// Usage
func validateAge(age int) error {
    if age < 0 {
        return ValidationError{
            Field:   "age",
            Message: "must be non-negative",
        }
    }
    return nil
}
```

### Interface Design

#### Small, Focused Interfaces
```go
// ✅ Small, focused interfaces
type Reader interface {
    Read([]byte) (int, error)
}

type Writer interface {
    Write([]byte) (int, error)
}

type ReadWriter interface {
    Reader
    Writer
}

// ✅ Domain-specific interfaces
type UserRepository interface {
    Create(ctx context.Context, user User) error
    GetByID(ctx context.Context, id string) (*User, error)
    Update(ctx context.Context, user User) error
    Delete(ctx context.Context, id string) error
}

// ❌ Avoid large, monolithic interfaces
type MegaInterface interface {
    DoEverything()
    HandleAllCases()
    ProcessAllData()
    ManageAllStates()
}
```

#### Accept Interfaces, Return Structs
```go
// ✅ Accept interfaces for flexibility
func ProcessData(r io.Reader, w io.Writer) error {
    data, err := io.ReadAll(r)
    if err != nil {
        return err
    }
    
    processed := transform(data)
    _, err = w.Write(processed)
    return err
}

// ✅ Return concrete structs for clarity
func NewUserService(db Database) *UserService {
    return &UserService{
        db: db,
    }
}
```

## Testing Strategies

### Table-Driven Tests
```go
func TestValidateEmail(t *testing.T) {
    tests := []struct {
        name    string
        email   string
        want    bool
        wantErr bool
    }{
        {
            name:  "valid email",
            email: "user@example.com",
            want:  true,
        },
        {
            name:    "empty email",
            email:   "",
            want:    false,
            wantErr: true,
        },
        {
            name:    "invalid format",
            email:   "invalid-email",
            want:    false,
            wantErr: true,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := ValidateEmail(tt.email)
            if (err != nil) != tt.wantErr {
                t.Errorf("ValidateEmail() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            if got != tt.want {
                t.Errorf("ValidateEmail() = %v, want %v", got, tt.want)
            }
        })
    }
}
```

### Testify Integration
```go
import (
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
    "github.com/stretchr/testify/mock"
)

// ✅ Using testify for cleaner assertions
func TestUserService_CreateUser(t *testing.T) {
    // Setup
    mockDB := &MockDatabase{}
    service := NewUserService(mockDB)
    user := User{ID: "123", Name: "John"}
    
    // Mock expectations
    mockDB.On("Create", mock.Anything, user).Return(nil)
    
    // Execute
    err := service.CreateUser(context.Background(), user)
    
    // Assert
    require.NoError(t, err)
    mockDB.AssertExpectations(t)
}

// ✅ Test helper functions
func setupTestDB(t *testing.T) *sql.DB {
    db, err := sql.Open("sqlite3", ":memory:")
    require.NoError(t, err)
    
    t.Cleanup(func() {
        db.Close()
    })
    
    return db
}
```

### Integration Testing
```go
func TestPromptRegistry_Integration(t *testing.T) {
    // Setup temporary directory
    tempDir := t.TempDir()
    
    // Initialize registry with test configuration
    config := &Config{
        StoragePath: tempDir,
        Storage:     "filesystem",
    }
    
    registry, err := NewRegistry(config)
    require.NoError(t, err)
    
    // Test the full workflow
    prompt := &Prompt{
        Name:    "test-prompt",
        Version: "1.0.0",
        Content: "Test content",
    }
    
    // Add prompt
    err = registry.AddPrompt(prompt)
    assert.NoError(t, err)
    
    // Fetch prompt
    fetched, err := registry.FetchPrompt("test-prompt", "latest")
    assert.NoError(t, err)
    assert.Equal(t, prompt.Content, fetched.Content)
    
    // List prompts
    prompts, err := registry.ListPrompts()
    assert.NoError(t, err)
    assert.Contains(t, prompts, "test-prompt")
}
```

## Performance Best Practices

### Memory Management
```go
// ✅ Efficient slice operations
func processLargeSlice(items []Item) []ProcessedItem {
    // Pre-allocate with known capacity
    result := make([]ProcessedItem, 0, len(items))
    
    for _, item := range items {
        if shouldProcess(item) {
            processed := processItem(item)
            result = append(result, processed)
        }
    }
    
    return result
}

// ✅ String building for multiple concatenations
func buildLargeString(parts []string) string {
    var builder strings.Builder
    
    // Pre-allocate capacity if known
    totalLen := 0
    for _, part := range parts {
        totalLen += len(part)
    }
    builder.Grow(totalLen)
    
    for _, part := range parts {
        builder.WriteString(part)
    }
    
    return builder.String()
}

// ✅ Pooling for frequently allocated objects
var bufferPool = sync.Pool{
    New: func() interface{} {
        return make([]byte, 1024)
    },
}

func processData(data []byte) ([]byte, error) {
    buf := bufferPool.Get().([]byte)
    defer bufferPool.Put(buf)
    
    // Use buf for processing
    return processWithBuffer(data, buf)
}
```

### Benchmarking
```go
func BenchmarkStringConcatenation(b *testing.B) {
    parts := []string{"hello", " ", "world", " ", "from", " ", "go"}
    
    b.Run("string concatenation", func(b *testing.B) {
        for i := 0; i < b.N; i++ {
            var result string
            for _, part := range parts {
                result += part
            }
            _ = result
        }
    })
    
    b.Run("strings.Builder", func(b *testing.B) {
        for i := 0; i < b.N; i++ {
            var builder strings.Builder
            for _, part := range parts {
                builder.WriteString(part)
            }
            _ = builder.String()
        }
    })
}
```

## Concurrency Patterns

### Goroutines and Channels
```go
// ✅ Worker pool pattern
func processItems(items []Item, workers int) []Result {
    jobs := make(chan Item, len(items))
    results := make(chan Result, len(items))
    
    // Start workers
    for w := 0; w < workers; w++ {
        go worker(jobs, results)
    }
    
    // Send jobs
    for _, item := range items {
        jobs <- item
    }
    close(jobs)
    
    // Collect results
    var allResults []Result
    for r := 0; r < len(items); r++ {
        allResults = append(allResults, <-results)
    }
    
    return allResults
}

func worker(jobs <-chan Item, results chan<- Result) {
    for item := range jobs {
        result := processItem(item)
        results <- result
    }
}
```

### Context Usage
```go
// ✅ Proper context usage
func fetchUserWithTimeout(ctx context.Context, userID string) (*User, error) {
    // Create context with timeout
    ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
    defer cancel()
    
    // Channel for result
    type result struct {
        user *User
        err  error
    }
    
    resultChan := make(chan result, 1)
    
    // Start operation in goroutine
    go func() {
        user, err := fetchUserFromDB(userID)
        resultChan <- result{user: user, err: err}
    }()
    
    // Wait for result or timeout
    select {
    case res := <-resultChan:
        return res.user, res.err
    case <-ctx.Done():
        return nil, ctx.Err()
    }
}

// ✅ Context cancellation propagation
func processChain(ctx context.Context, data Data) error {
    // Pass context through the chain
    processed, err := step1(ctx, data)
    if err != nil {
        return err
    }
    
    validated, err := step2(ctx, processed)
    if err != nil {
        return err
    }
    
    return step3(ctx, validated)
}
```

### Common Concurrency Pitfalls
```go
// ❌ Race condition - accessing shared state without synchronization
var counter int
func incrementCounter() {
    counter++ // Race condition!
}

// ✅ Synchronized access
var (
    counter int
    mu      sync.Mutex
)

func incrementCounter() {
    mu.Lock()
    defer mu.Unlock()
    counter++
}

// ✅ Or use atomic operations for simple cases
var counter int64

func incrementCounter() {
    atomic.AddInt64(&counter, 1)
}

// ❌ Goroutine leak - not waiting for goroutines to finish
func badConcurrency() {
    for i := 0; i < 10; i++ {
        go func(i int) {
            // Long running operation
            time.Sleep(time.Hour)
        }(i)
    }
    // Function returns, but goroutines keep running!
}

// ✅ Proper goroutine management
func goodConcurrency() {
    var wg sync.WaitGroup
    
    for i := 0; i < 10; i++ {
        wg.Add(1)
        go func(i int) {
            defer wg.Done()
            // Operation with context for cancellation
            processItem(i)
        }(i)
    }
    
    wg.Wait() // Wait for all goroutines to complete
}
```

## Project Structure and Organization

### Standard Project Layout
```
prompt-registry/
├── main.go                          # Entry point
├── cmd/                            # CLI commands (Cobra)
│   ├── root.go
│   ├── add.go
│   ├── fetch.go
│   └── list.go
├── internal/                       # Private application code
│   ├── models/                     # Domain models
│   │   ├── prompt.go
│   │   └── validation.go
│   ├── storage/                    # Storage interface & implementations
│   │   ├── interface.go
│   │   ├── filesystem.go
│   │   └── github.go
│   ├── registry/                   # Business logic
│   │   └── registry.go
│   └── runner/                     # Tool execution
│       └── launcher.go
├── pkg/                           # Public library code (if any)
├── test/                          # Integration tests
├── examples/                      # Example prompts
└── README.md
```

### Package Organization
```go
// ✅ Good package structure - focused responsibility
package storage

type Interface interface {
    Store(prompt Prompt) error
    Fetch(name, version string) (*Prompt, error)
    List() ([]string, error)
}

type FilesystemStorage struct {
    basePath string
}

func NewFilesystemStorage(path string) *FilesystemStorage {
    return &FilesystemStorage{basePath: path}
}

// ✅ Clear separation of concerns
package registry

type Registry struct {
    storage storage.Interface
    cache   cache.Interface
    logger  Logger
}

func (r *Registry) AddPrompt(prompt *models.Prompt) error {
    // Business logic for adding prompts
}
```

## Prompt Registry Specific Guidelines

### CLI Command Structure
```go
// ✅ Clean command structure using Cobra
var addCmd = &cobra.Command{
    Use:   "add [prompt-name]",
    Short: "Add a new prompt to the registry",
    Args:  cobra.ExactArgs(1),
    RunE: func(cmd *cobra.Command, args []string) error {
        return runAdd(cmd, args)
    },
}

func runAdd(cmd *cobra.Command, args []string) error {
    // Extract flags
    version, _ := cmd.Flags().GetString("version")
    file, _ := cmd.Flags().GetString("file")
    
    // Validate input
    if version == "" {
        return errors.New("version is required")
    }
    
    // Execute business logic
    registry, err := createRegistry()
    if err != nil {
        return fmt.Errorf("failed to create registry: %w", err)
    }
    
    return registry.AddPrompt(args[0], version, file)
}
```

### Configuration Management
```go
// ✅ Configuration with validation
type Config struct {
    Storage     string `yaml:"storage" validate:"required,oneof=filesystem github"`
    StoragePath string `yaml:"storage_path" validate:"required"`
    CacheSize   int    `yaml:"cache_size" validate:"min=0,max=1000"`
    LogLevel    string `yaml:"log_level" validate:"oneof=debug info warn error"`
}

func LoadConfig(path string) (*Config, error) {
    data, err := os.ReadFile(path)
    if err != nil {
        return nil, fmt.Errorf("failed to read config file: %w", err)
    }
    
    var config Config
    if err := yaml.Unmarshal(data, &config); err != nil {
        return nil, fmt.Errorf("failed to parse config: %w", err)
    }
    
    if err := validateConfig(&config); err != nil {
        return nil, fmt.Errorf("invalid configuration: %w", err)
    }
    
    return &config, nil
}
```

### Error Handling in CLI Context
```go
// ✅ User-friendly error messages in CLI
func (r *Registry) FetchPrompt(name, version string) (*models.Prompt, error) {
    if name == "" {
        return nil, fmt.Errorf("prompt name cannot be empty")
    }
    
    prompt, err := r.storage.Fetch(name, version)
    if err != nil {
        if os.IsNotExist(err) {
            return nil, fmt.Errorf("prompt '%s' version '%s' not found", name, version)
        }
        return nil, fmt.Errorf("failed to fetch prompt '%s': %w", name, err)
    }
    
    return prompt, nil
}

// ✅ CLI error handling with exit codes
func Execute() {
    if err := rootCmd.Execute(); err != nil {
        // Log the error
        log.Error("Command failed", "error", err)
        
        // Exit with appropriate code
        if exitErr, ok := err.(*ExitError); ok {
            os.Exit(exitErr.Code)
        }
        os.Exit(1)
    }
}
```

## Quality Assurance

### Code Quality Tools
```bash
# Format code
go fmt ./...
gofmt -s -w .

# Vet for common issues
go vet ./...

# Lint with golangci-lint (recommended)
golangci-lint run

# Run tests with coverage
go test -v -race -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Makefile Integration
```makefile
.PHONY: fmt lint test build

# Format code
fmt:
	gofmt -s -w .
	goimports -w .

# Lint code
lint:
	golangci-lint run

# Run tests
test:
	go test -v -race -coverprofile=coverage.out ./...

# Build binary
build:
	go build -o build/prompt-registry ./main.go

# Run all quality checks
quality: fmt lint test

# Install dependencies
deps:
	go mod download
	go mod tidy
```

### Documentation Standards
```go
// ✅ Package documentation
// Package registry provides centralized management of LLM instruction prompts
// with versioning, validation, and multiple storage backend support.
//
// The registry supports both filesystem and GitHub-based storage, allowing
// teams to manage prompts either locally or in a shared repository.
//
// Example usage:
//
//	config := &Config{Storage: "filesystem", StoragePath: "./prompts"}
//	registry, err := NewRegistry(config)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	prompt, err := registry.FetchPrompt("claude", "latest")
package registry

// ✅ Function documentation with examples
// AddPrompt stores a new prompt in the registry with the specified version.
// If the prompt already exists with the same version, it returns an error.
//
// The prompt content is validated according to the registry's validation rules
// before storage. Version must follow semantic versioning format.
//
// Example:
//
//	err := registry.AddPrompt("claude", "1.2.0", "./CLAUDE.md")
//	if err != nil {
//	    return fmt.Errorf("failed to add prompt: %w", err)
//	}
func (r *Registry) AddPrompt(name, version, filepath string) error {
    // Implementation
}
```

## Development Workflow

### Git and Version Control
```bash
# Branch naming: <username>_<feature_description>
git checkout -b fr12k_golang_guidelines

# Commit messages: conventional commits
git commit -m "feat(docs): add comprehensive golang development guidelines"
git commit -m "fix(storage): handle file not found errors properly"
git commit -m "refactor(registry): simplify prompt validation logic"

# Before committing
make fmt lint test

# Push and create PR
git push -u origin fr12k_golang_guidelines
gh pr create --fill
```

### Development Best Practices
1. **Write tests first**: TDD approach for better design
2. **Small commits**: Each commit should have a single responsibility
3. **Code review**: Always have code reviewed before merging
4. **Documentation**: Update docs alongside code changes
5. **Performance**: Profile critical paths and benchmark improvements

### Debugging and Troubleshooting
```go
// ✅ Structured logging
import "log/slog"

func (r *Registry) AddPrompt(name, version string) error {
    logger := slog.With("prompt", name, "version", version)
    logger.Info("Adding prompt to registry")
    
    if err := r.validatePrompt(name, version); err != nil {
        logger.Error("Prompt validation failed", "error", err)
        return fmt.Errorf("validation failed: %w", err)
    }
    
    logger.Info("Prompt added successfully")
    return nil
}

// ✅ Debug build tags
//go:build debug

package main

import "log/slog"

func init() {
    // Enable debug logging in debug builds
    slog.SetLogLoggerLevel(slog.LevelDebug)
}
```

## Integration with Prompt Registry

When working on the prompt-registry project specifically:

### Adding New Commands
1. Create command file in `cmd/` directory
2. Implement using Cobra patterns established in existing commands
3. Add comprehensive tests including table-driven tests
4. Update help text and documentation
5. Ensure error messages are user-friendly

### Storage Backend Development
1. Implement the `storage.Interface`
2. Add configuration options to `Config` struct
3. Include connection testing and validation
4. Add comprehensive error handling
5. Write integration tests with real storage

### Model Changes
1. Update struct definitions in `internal/models/`
2. Add validation rules using struct tags
3. Update serialization/deserialization logic
4. Ensure backward compatibility for existing prompts
5. Add migration logic if needed

This comprehensive guide ensures consistent, high-quality Go development practices while maintaining the specific patterns and conventions established in the prompt-registry project.

---

🤖 Generated with Multi-Dimensional Prompt System

**Layer Composition:**
- general@latest (v1.0.1) (https://github.com/goflink/ai/blob/fr12k_prompt/prompts/general/v1.0.1.md)
- general-claude@latest (v1.0.0) (https://github.com/goflink/ai/blob/fr12k_prompt/prompts/general-claude/v1.0.0.md)
- general-golang@latest (v1.0.0) (https://github.com/goflink/ai/blob/fr12k_prompt/prompts/general-golang/v1.0.0.md)

