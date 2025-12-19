package doctor

import "context"

// CheckCategory groups related checks
type CheckCategory string

const (
	CategoryRuntime      CheckCategory = "Container Runtime"
	CategoryConnectivity CheckCategory = "Runtime Connectivity"
	CategoryBuildTools   CheckCategory = "Build Tools"
	CategoryNetwork      CheckCategory = "Network Access"
	CategorySystem       CheckCategory = "System Resources"
	CategoryPermissions  CheckCategory = "User Permissions"
	CategoryGitHub       CheckCategory = "GitHub Actions"
)

// CheckSeverity indicates check importance
type CheckSeverity string

const (
	SeverityCritical CheckSeverity = "critical" // Must pass for engine-ci to work
	SeverityWarning  CheckSeverity = "warning"  // Should pass but engine-ci might work
	SeverityInfo     CheckSeverity = "info"     // Informational only
)

// CheckStatus represents check result
type CheckStatus string

const (
	StatusPass    CheckStatus = "pass"
	StatusFail    CheckStatus = "fail"
	StatusWarning CheckStatus = "warning"
	StatusSkipped CheckStatus = "skipped"
)

// Check defines base metadata for a diagnostic check
type Check struct {
	Name      string
	Category  CheckCategory
	Severity  CheckSeverity
	ShouldRun bool
}

// CheckRunner interface defines the contract for running checks
type CheckRunner interface {
	Run(ctx context.Context) CheckResult
	GetCheck() *Check
}

// GetCheck returns the check metadata (satisfies CheckRunner interface)
func (c *Check) GetCheck() *Check {
	return c
}

// NewCheckResult creates a base CheckResult for a check
func (c *Check) NewCheckResult() CheckResult {
	return CheckResult{
		CheckName: c.Name,
		Category:  c.Category,
		Severity:  c.Severity,
		Metadata:  make(map[string]interface{}),
	}
}

// CheckResult contains check execution results
type CheckResult struct {
	Error       error
	Metadata    map[string]interface{}
	CheckName   string
	Category    CheckCategory
	Severity    CheckSeverity
	Status      CheckStatus
	Message     string
	Details     []string
	Suggestions []string
}
