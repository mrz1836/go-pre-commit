package cmd

import (
	"cmp"
	"fmt"
	"slices"

	"github.com/mrz1836/go-pre-commit/internal/runner"
)

// disabledWarning is printed when ENABLE_GO_PRE_COMMIT is false
const disabledWarning = "Pre-commit system is disabled in configuration (ENABLE_GO_PRE_COMMIT=false)"

// Results reported by `run --json`, for the run and for each check
const (
	runResultPassed  = "passed"  // every check that ran passed (or was skipped)
	runResultFailed  = "failed"  // at least one check failed
	runResultSkipped = "skipped" // no check ran: the system is disabled or there were no files

	checkResultPassed  = "passed"
	checkResultFailed  = "failed"
	checkResultSkipped = "skipped" // the check could not run and --graceful skipped it
)

// runRecord is what a run did, for the JSON report
type runRecord struct {
	enabled bool
	files   []string
	results *runner.Results // nil when no check ran
}

// runReport is the document `run --json` prints
type runReport struct {
	Version  string     `json:"version"`
	Enabled  bool       `json:"enabled"`
	Result   string     `json:"result"`
	Files    []string   `json:"files"`
	Checks   []runCheck `json:"checks"`
	Totals   runTotals  `json:"totals"`
	Warnings []string   `json:"warnings"`
}

// runCheck is one check in the run report
type runCheck struct {
	Name       string   `json:"name"`
	Result     string   `json:"result"`
	DurationMs int64    `json:"duration_ms"`
	Files      []string `json:"files"`
	// ChangedFiles is null for a check that does not report what it changes
	ChangedFiles []string `json:"changed_files"`
	Error        string   `json:"error,omitempty"`
	Suggestion   string   `json:"suggestion,omitempty"`
	Output       string   `json:"output,omitempty"`
}

// runTotals sums up the run report
type runTotals struct {
	Checks     int   `json:"checks"`
	Passed     int   `json:"passed"`
	Failed     int   `json:"failed"`
	Skipped    int   `json:"skipped"`
	Files      int   `json:"files"`
	DurationMs int64 `json:"duration_ms"`
}

// newRunReport builds the run report from what the run did
func newRunReport(version string, record *runRecord) runReport {
	report := runReport{
		Version:  version,
		Enabled:  record.enabled,
		Files:    nonNil(record.files),
		Checks:   []runCheck{},
		Warnings: []string{},
	}
	if !record.enabled {
		report.Warnings = append(report.Warnings, disabledWarning)
	}

	report.Totals.Files = len(report.Files)
	if record.results != nil {
		report.Totals.Files = record.results.TotalFiles
		report.Totals.DurationMs = record.results.TotalDuration.Milliseconds()
		for _, result := range record.results.CheckResults {
			check := newRunCheck(result)
			report.Checks = append(report.Checks, check)
			switch check.Result {
			case checkResultFailed:
				report.Totals.Failed++
			case checkResultSkipped:
				report.Totals.Skipped++
				report.Warnings = append(report.Warnings, fmt.Sprintf("%s skipped: %s", check.Name, check.Error))
			default:
				report.Totals.Passed++
			}
		}
	}
	// Parallel checks finish in any order; report them in a stable one
	slices.SortFunc(report.Checks, func(a, b runCheck) int { return cmp.Compare(a.Name, b.Name) })
	slices.Sort(report.Warnings)
	report.Totals.Checks = len(report.Checks)

	switch {
	case report.Totals.Failed > 0:
		report.Result = runResultFailed
	case report.Totals.Checks == 0:
		report.Result = runResultSkipped
	default:
		report.Result = runResultPassed
	}
	return report
}

// newRunCheck reports one check result. The runner records a check skipped
// by --graceful as a success that carries the reason it could not run.
func newRunCheck(result runner.CheckResult) runCheck {
	check := runCheck{
		Name:         result.Name,
		DurationMs:   result.Duration.Milliseconds(),
		Files:        nonNil(result.Files),
		ChangedFiles: result.ChangedFiles,
	}
	switch {
	case result.Success && result.CanSkip:
		check.Result = checkResultSkipped
	case result.Success:
		check.Result = checkResultPassed
		return check
	default:
		check.Result = checkResultFailed
		check.Output = stripANSI(result.Output)
	}
	check.Error = result.Error
	check.Suggestion = result.Suggestion
	return check
}

// nonNil returns files, or an empty list when it is nil, so JSON shows []
func nonNil(files []string) []string {
	if files == nil {
		return []string{}
	}
	return files
}
