package doctor

import (
	"context"
	"sort"
	"sync"
)

// Doctor coordinates diagnostic checks
type Doctor struct {
	checks []CheckRunner
	opts   DoctorOptions
}

// DoctorOptions configures doctor behavior
type DoctorOptions struct {
	Categories          []CheckCategory
	Verbose             bool
	JSONOutput          bool
	IncludeWarnings     bool
	Parallel            bool
	KeepTestContainers  bool // Don't cleanup test containers (for debugging)
}

// NewDoctor creates a new doctor instance
func NewDoctor(opts DoctorOptions) *Doctor {
	d := &Doctor{
		checks: make([]CheckRunner, 0),
		opts:   opts,
	}

	// Register all checks
	d.registerChecks()

	return d
}

// registerChecks adds all available checks
func (d *Doctor) registerChecks() {
	// Runtime checks
	d.checks = append(d.checks, NewRuntimeDetectionCheck())
	d.checks = append(d.checks, NewRuntimeConnectivityCheck())
	d.checks = append(d.checks, NewRuntimeVersionCheck())

	// Volume permission checks
	d.checks = append(d.checks, NewVolumeConfigCheck())
	d.checks = append(d.checks, NewVolumeWriteTestCheck(d.opts.KeepTestContainers))
}

// RegisterCheck adds a check to the doctor
func (d *Doctor) RegisterCheck(check CheckRunner) {
	d.checks = append(d.checks, check)
}

// RunChecks executes all applicable checks
func (d *Doctor) RunChecks(ctx context.Context) []CheckResult {
	// Filter checks based on options
	applicableChecks := d.filterChecks(ctx)

	if d.opts.Parallel {
		return d.runParallel(ctx, applicableChecks)
	}

	return d.runSequential(ctx, applicableChecks)
}

// filterChecks returns checks that should run
func (d *Doctor) filterChecks(_ context.Context) []CheckRunner {
	filtered := make([]CheckRunner, 0)

	for _, runner := range d.checks {
		check := runner.GetCheck()

		// Filter by category if specified
		if len(d.opts.Categories) > 0 {
			found := false
			for _, cat := range d.opts.Categories {
				if check.Category == cat {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		// Check if applicable
		if !check.ShouldRun {
			continue
		}

		filtered = append(filtered, runner)
	}

	return filtered
}

// runSequential runs checks one by one
func (d *Doctor) runSequential(ctx context.Context, checks []CheckRunner) []CheckResult {
	results := make([]CheckResult, 0, len(checks))

	for _, runner := range checks {
		result := runner.Run(ctx)
		results = append(results, result)
	}

	return results
}

// runParallel runs checks concurrently
func (d *Doctor) runParallel(ctx context.Context, checks []CheckRunner) []CheckResult {
	results := make([]CheckResult, 0, len(checks))
	resultsMu := sync.Mutex{}

	var wg sync.WaitGroup
	for _, runner := range checks {
		wg.Add(1)
		go func(r CheckRunner) {
			defer wg.Done()
			result := r.Run(ctx)

			resultsMu.Lock()
			results = append(results, result)
			resultsMu.Unlock()
		}(runner)
	}

	wg.Wait()

	// Sort by category and name for consistent output
	sort.Slice(results, func(i, j int) bool {
		if results[i].Category != results[j].Category {
			return results[i].Category < results[j].Category
		}
		return results[i].CheckName < results[j].CheckName
	})

	return results
}
