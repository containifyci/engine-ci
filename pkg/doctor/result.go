package doctor

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

const (
	checkmark = "✓"
	cross     = "✗"
	warning   = "⚠"
	info      = "ℹ"

	colorReset  = "\033[0m"
	colorGreen  = "\033[32m"
	colorRed    = "\033[31m"
	colorYellow = "\033[33m"
	colorGray   = "\033[90m"
	colorBold   = "\033[1m"
)

// ResultFormatter handles output formatting
type ResultFormatter struct {
	writer     io.Writer
	verbose    bool
	jsonOutput bool
	useColor   bool
}

// NewResultFormatter creates a formatter
func NewResultFormatter(w io.Writer, verbose, jsonOutput, useColor bool) *ResultFormatter {
	return &ResultFormatter{
		writer:     w,
		verbose:    verbose,
		jsonOutput: jsonOutput,
		useColor:   useColor,
	}
}

// FormatResults outputs all results
func (f *ResultFormatter) FormatResults(results []CheckResult) error {
	if f.jsonOutput {
		return f.formatJSON(results)
	}

	return f.formatHuman(results)
}

// formatJSON outputs results as JSON
func (f *ResultFormatter) formatJSON(results []CheckResult) error {
	output := struct {
		Results []CheckResult `json:"results"`
		Summary Summary       `json:"summary"`
	}{
		Results: results,
		Summary: f.calculateSummary(results),
	}

	encoder := json.NewEncoder(f.writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(output)
}

// formatHuman outputs human-readable format
func (f *ResultFormatter) formatHuman(results []CheckResult) error {
	// Group by category
	categoryGroups := make(map[CheckCategory][]CheckResult)
	for _, result := range results {
		categoryGroups[result.Category] = append(
			categoryGroups[result.Category],
			result,
		)
	}

	// Print header
	f.printHeader()

	// Print each category
	categories := []CheckCategory{
		CategoryRuntime,
		CategoryConnectivity,
		CategoryBuildTools,
		CategoryNetwork,
		CategorySystem,
		CategoryPermissions,
		CategoryGitHub,
	}

	for _, cat := range categories {
		if catResults, ok := categoryGroups[cat]; ok && len(catResults) > 0 {
			f.printCategory(cat, catResults)
		}
	}

	// Print summary
	f.printSummary(results)

	return nil
}

// printCategory prints a category section
func (f *ResultFormatter) printCategory(cat CheckCategory, results []CheckResult) {
	fmt.Fprintf(f.writer, "\n%s%s%s\n", f.color(colorBold), cat, f.color(colorReset))
	fmt.Fprintf(f.writer, "%s\n", strings.Repeat("─", len(cat)))

	for _, result := range results {
		f.printResult(result)
	}
}

// printResult prints a single result
func (f *ResultFormatter) printResult(result CheckResult) {
	// Status symbol and color
	var symbol, color string
	switch result.Status {
	case StatusPass:
		symbol = checkmark
		color = colorGreen
	case StatusFail:
		symbol = cross
		color = colorRed
	case StatusWarning:
		symbol = warning
		color = colorYellow
	case StatusSkipped:
		symbol = info
		color = colorGray
	}

	// Print check name and status
	fmt.Fprintf(f.writer, "  %s%s%s %s\n",
		f.color(color), symbol, f.color(colorReset), result.CheckName)

	// Print message
	if result.Message != "" {
		fmt.Fprintf(f.writer, "    %s%s%s\n",
			f.color(colorGray), result.Message, f.color(colorReset))
	}

	// Print details if verbose
	if f.verbose && len(result.Details) > 0 {
		for _, detail := range result.Details {
			fmt.Fprintf(f.writer, "      • %s\n", detail)
		}
	}

	// Print suggestions for failed/warning checks
	if (result.Status == StatusFail || result.Status == StatusWarning) &&
		len(result.Suggestions) > 0 {
		fmt.Fprintf(f.writer, "    %sSuggestions:%s\n",
			f.color(colorBold), f.color(colorReset))
		for _, suggestion := range result.Suggestions {
			fmt.Fprintf(f.writer, "      → %s\n", suggestion)
		}
	}

	fmt.Fprintln(f.writer)
}

// Summary contains check result summary
type Summary struct {
	Total    int `json:"total"`
	Passed   int `json:"passed"`
	Failed   int `json:"failed"`
	Warnings int `json:"warnings"`
	Skipped  int `json:"skipped"`
	Critical int `json:"critical_failures"`
}

// calculateSummary computes summary statistics
func (f *ResultFormatter) calculateSummary(results []CheckResult) Summary {
	summary := Summary{Total: len(results)}

	for _, result := range results {
		switch result.Status {
		case StatusPass:
			summary.Passed++
		case StatusFail:
			summary.Failed++
			if result.Severity == SeverityCritical {
				summary.Critical++
			}
		case StatusWarning:
			summary.Warnings++
		case StatusSkipped:
			summary.Skipped++
		}
	}

	return summary
}

// printSummary prints summary statistics
func (f *ResultFormatter) printSummary(results []CheckResult) {
	summary := f.calculateSummary(results)

	fmt.Fprintf(f.writer, "\n%s%s%s\n",
		f.color(colorBold), "Summary", f.color(colorReset))
	fmt.Fprintf(f.writer, "%s\n", strings.Repeat("─", 7))

	fmt.Fprintf(f.writer, "Total checks:      %d\n", summary.Total)
	fmt.Fprintf(f.writer, "%sPassed:%s           %d\n",
		f.color(colorGreen), f.color(colorReset), summary.Passed)

	if summary.Failed > 0 {
		fmt.Fprintf(f.writer, "%sFailed:%s           %d",
			f.color(colorRed), f.color(colorReset), summary.Failed)
		if summary.Critical > 0 {
			fmt.Fprintf(f.writer, " (%d critical)", summary.Critical)
		}
		fmt.Fprintln(f.writer)
	}

	if summary.Warnings > 0 {
		fmt.Fprintf(f.writer, "%sWarnings:%s         %d\n",
			f.color(colorYellow), f.color(colorReset), summary.Warnings)
	}

	if summary.Skipped > 0 {
		fmt.Fprintf(f.writer, "%sSkipped:%s          %d\n",
			f.color(colorGray), f.color(colorReset), summary.Skipped)
	}

	// Final verdict
	fmt.Fprintln(f.writer)
	if summary.Critical > 0 {
		fmt.Fprintf(f.writer, "%s%s Critical issues detected. Engine-CI may not function properly.%s\n",
			f.color(colorRed), cross, f.color(colorReset))
	} else if summary.Failed > 0 {
		fmt.Fprintf(f.writer, "%s%s Some checks failed. Review suggestions above.%s\n",
			f.color(colorYellow), warning, f.color(colorReset))
	} else {
		fmt.Fprintf(f.writer, "%s%s All checks passed! Environment is ready.%s\n",
			f.color(colorGreen), checkmark, f.color(colorReset))
	}
}

// printHeader prints output header
func (f *ResultFormatter) printHeader() {
	fmt.Fprintf(f.writer, "%sEngine-CI Environment Diagnostics%s\n",
		f.color(colorBold), f.color(colorReset))
	fmt.Fprintf(f.writer, "%s\n\n", strings.Repeat("=", 35))
}

// color returns color code if color is enabled, empty string otherwise
func (f *ResultFormatter) color(code string) string {
	if f.useColor {
		return code
	}
	return ""
}
