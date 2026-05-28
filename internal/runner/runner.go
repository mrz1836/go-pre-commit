// Package runner provides the check execution engine for the pre-commit system
package runner

import (
	"context"
	"errors"
	"fmt"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/mrz1836/go-pre-commit/internal/checks"
	"github.com/mrz1836/go-pre-commit/internal/config"
	prerrors "github.com/mrz1836/go-pre-commit/internal/errors"
	"github.com/mrz1836/go-pre-commit/internal/tools"
)

// Check name constants
const (
	checkNameFumpt      = "fumpt"
	checkNameGitleaks   = "gitleaks"
	checkNameLint       = "lint"
	checkNameModTidy    = "mod-tidy"
	checkNameEOF        = "eof"
	checkNameWhitespace = "whitespace"
	envSkip             = "SKIP"
)

// ErrCheckPanicked indicates a check's Run method panicked. The runner recovers
// from it so one faulty check or plugin degrades to a failed result instead of
// crashing the entire pre-commit run.
var ErrCheckPanicked = errors.New("check panicked")

// Runner executes pre-commit checks
type Runner struct {
	config   *config.Config
	repoRoot string
	registry *checks.Registry
}

// Options configures a check run
type Options struct {
	Files               []string
	OnlyChecks          []string
	SkipChecks          []string
	Parallel            int
	FailFast            bool
	ProgressCallback    ProgressCallback
	GracefulDegradation bool
	DebugTimeout        bool
}

// Results contains the results of a check run
type Results struct {
	CheckResults  []CheckResult
	Passed        int
	Failed        int
	Skipped       int
	TotalDuration time.Duration
	TotalFiles    int
}

// CheckResult contains the result of a single check
type CheckResult struct {
	Name       string
	Success    bool
	Error      string
	Output     string
	Duration   time.Duration
	Files      []string
	Suggestion string
	CanSkip    bool
	Command    string
}

// ProgressCallback is called during check execution for progress updates
// The duration parameter contains the check execution time (0 for "running" status)
type ProgressCallback func(checkName, status string, duration time.Duration)

// New creates a new Runner
func New(cfg *config.Config, repoRoot string) *Runner {
	if cfg == nil {
		return nil
	}

	// Configure tool installation timeout from config
	toolTimeout := time.Duration(cfg.ToolInstallation.Timeout) * time.Second
	tools.SetInstallTimeout(toolTimeout)

	return &Runner{
		config:   cfg,
		repoRoot: repoRoot,
		registry: checks.NewRegistryWithConfig(cfg),
	}
}

// Run executes checks based on the provided options
func (r *Runner) Run(ctx context.Context, opts Options) (*Results, error) {
	start := time.Now()

	// Process SKIP environment variables and combine with CLI skip options
	opts.SkipChecks = r.combineSkipSources(opts.SkipChecks)

	// Determine which checks to run
	checksToRun, err := r.determineChecks(opts)
	if err != nil {
		return nil, err
	}

	// Determine parallelism
	parallel := r.resolveParallelism(opts)

	// Create context with timeout
	globalTimeout := time.Duration(r.config.Timeout) * time.Second
	ctxWithTimeout, cancel := context.WithTimeout(ctx, globalTimeout)
	defer cancel()

	// Debug timeout information
	if opts.DebugTimeout {
		r.debugTimeoutInfo(globalTimeout)
	}

	// Run checks
	results := &Results{
		CheckResults: make([]CheckResult, 0, len(checksToRun)),
		TotalFiles:   len(opts.Files),
	}

	if opts.FailFast {
		r.runSequential(ctxWithTimeout, checksToRun, opts, results)
	} else {
		r.runParallel(ctxWithTimeout, checksToRun, parallel, opts, results)
	}

	results.TotalDuration = time.Since(start)
	return results, nil
}

// resolveParallelism determines the worker count, preferring the explicit
// option, then the configured value, then the host CPU count.
func (r *Runner) resolveParallelism(opts Options) int {
	if opts.Parallel > 0 {
		return opts.Parallel
	}
	if r.config.Performance.ParallelWorkers > 0 {
		return r.config.Performance.ParallelWorkers
	}
	return runtime.NumCPU()
}

// debugTimeoutInfo prints timeout diagnostics to stderr when --debug-timeout is set.
func (r *Runner) debugTimeoutInfo(globalTimeout time.Duration) {
	fmt.Fprintf(os.Stderr, "🐛 [DEBUG-TIMEOUT] Global timeout set to: %v\n", globalTimeout)
	fmt.Fprintf(os.Stderr, "🐛 [DEBUG-TIMEOUT] Tool installation timeout: %v\n", time.Duration(r.config.ToolInstallation.Timeout)*time.Second)
	if r.config.Environment.IsCI {
		fmt.Fprintf(os.Stderr, "🐛 [DEBUG-TIMEOUT] CI environment detected: %s (auto-adjust: %t)\n", r.config.Environment.CIProvider, r.config.Environment.AutoAdjustTimers)
	}
}

// runSequential executes checks one at a time, stopping at the first hard failure.
func (r *Runner) runSequential(ctx context.Context, checksToRun []checks.Check, opts Options, results *Results) {
	for _, check := range checksToRun {
		r.notifyProgress(opts, check.Name(), "running", 0)
		result := r.runCheck(ctx, check, opts.Files, opts.GracefulDegradation, opts.DebugTimeout)
		if r.tallyResult(result, opts, results) {
			break // Stop on first failure
		}
	}
}

// runParallel executes checks concurrently, bounded by the given worker count.
func (r *Runner) runParallel(ctx context.Context, checksToRun []checks.Check, parallel int, opts Options, results *Results) {
	resultsChan := make(chan CheckResult, len(checksToRun))
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, parallel)

	for _, check := range checksToRun {
		wg.Add(1)
		go func(c checks.Check) {
			defer wg.Done()

			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			r.notifyProgress(opts, c.Name(), "running", 0)
			resultsChan <- r.runCheck(ctx, c, opts.Files, opts.GracefulDegradation, opts.DebugTimeout)
		}(check)
	}

	wg.Wait()
	close(resultsChan)

	for result := range resultsChan {
		r.tallyResult(result, opts, results)
	}
}

// tallyResult records a finished check result into the aggregate counts and
// emits the matching progress callback. It reports whether the result counted
// as a hard failure (used to drive fail-fast termination).
func (r *Runner) tallyResult(result CheckResult, opts Options, results *Results) (failed bool) {
	results.CheckResults = append(results.CheckResults, result)
	switch {
	case result.Success:
		results.Passed++
		r.notifyProgress(opts, result.Name, "passed", result.Duration)
	case result.CanSkip && opts.GracefulDegradation:
		results.Skipped++
		r.notifyProgress(opts, result.Name, "skipped", result.Duration)
	default:
		results.Failed++
		r.notifyProgress(opts, result.Name, "failed", result.Duration)
		failed = true
	}
	return failed
}

// notifyProgress invokes the progress callback when one is configured.
func (r *Runner) notifyProgress(opts Options, name, status string, duration time.Duration) {
	if opts.ProgressCallback != nil {
		opts.ProgressCallback(name, status, duration)
	}
}

// runCheck executes a single check
func (r *Runner) runCheck(ctx context.Context, check checks.Check, files []string, gracefulDegradation, debugTimeout bool) CheckResult {
	start := time.Now()

	if debugTimeout {
		checkName := check.Name()
		timeout := r.getCheckTimeout(checkName)
		fmt.Fprintf(os.Stderr, "🐛 [DEBUG-TIMEOUT] Starting check '%s' with timeout: %v\n", checkName, timeout)
	}

	// Apply configured exclude patterns, then filter files for this check
	nonExcludedFiles := r.applyExcludePatterns(files)
	filteredFiles := check.FilterFiles(nonExcludedFiles)
	if len(filteredFiles) == 0 {
		return CheckResult{
			Name:     check.Name(),
			Success:  true,
			Duration: time.Since(start),
			Files:    filteredFiles,
		}
	}

	// Run the check, recovering from panics so a single faulty check (or plugin)
	// becomes a failed result rather than crashing the whole run.
	err := r.safeCheckRun(ctx, check, filteredFiles)

	result := CheckResult{
		Name:     check.Name(),
		Success:  err == nil,
		Duration: time.Since(start),
		Files:    filteredFiles,
	}

	if err != nil {
		result.Error = err.Error()

		// Check if this is a timeout error first
		var timeoutErr *prerrors.TimeoutError
		if errors.As(err, &timeoutErr) {
			result.Suggestion = timeoutErr.Error() // The TimeoutError already contains helpful message
			if debugTimeout {
				fmt.Fprintf(os.Stderr, "🐛 [DEBUG-TIMEOUT] Check '%s' failed with TimeoutError: %v\n", check.Name(), timeoutErr.Error())
			}
		} else if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			// General timeout - create a timeout error with context
			timeout := time.Duration(r.config.Timeout) * time.Second
			timeoutErr := prerrors.NewCheckTimeoutError(check.Name(), timeout, result.Duration)
			result.Error = timeoutErr.Error()
			result.Suggestion = timeoutErr.Error()
			if debugTimeout {
				fmt.Fprintf(os.Stderr, "🐛 [DEBUG-TIMEOUT] Check '%s' hit context deadline: elapsed=%v, timeout=%v\n", check.Name(), result.Duration, timeout)
			}
		} else {
			// Check if this is an enhanced CheckError with context
			var checkErr *prerrors.CheckError
			if errors.As(err, &checkErr) {
				result.Suggestion = checkErr.Suggestion
				result.CanSkip = checkErr.CanSkip
				result.Command = checkErr.Command
				result.Output = checkErr.Output

				// If graceful degradation is enabled and this error can be skipped
				if gracefulDegradation && checkErr.CanSkip {
					result.Success = true // Mark as success but with warning info
				}
			}
		}
	}

	return result
}

// safeCheckRun executes check.Run, converting any panic into an error so a faulty
// check or plugin degrades to a failed result instead of crashing the process.
func (r *Runner) safeCheckRun(ctx context.Context, check checks.Check, files []string) (err error) {
	defer func() {
		if rec := recover(); rec != nil {
			err = fmt.Errorf("%w in %s: %v", ErrCheckPanicked, check.Name(), rec)
		}
	}()
	return check.Run(ctx, files)
}

// determineChecks figures out which checks to run based on options and config
func (r *Runner) determineChecks(opts Options) ([]checks.Check, error) {
	// Get all available checks
	allChecks := r.registry.GetChecks()

	checksToRun := make([]checks.Check, 0, len(allChecks))

	// Filter based on options
	for _, check := range allChecks {
		name := check.Name()

		// Skip if disabled in config
		if !r.isCheckEnabled(name) {
			continue
		}

		// Handle --only flag
		if len(opts.OnlyChecks) > 0 {
			found := false
			for _, only := range opts.OnlyChecks {
				if only == name {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		// Handle --skip flag
		if len(opts.SkipChecks) > 0 {
			skip := false
			for _, skipName := range opts.SkipChecks {
				if skipName == name {
					skip = true
					break
				}
			}
			if skip {
				continue
			}
		}

		checksToRun = append(checksToRun, check)
	}

	if len(checksToRun) == 0 {
		return nil, prerrors.ErrNoChecksToRun
	}

	return checksToRun, nil
}

// getCheckTimeout returns the timeout for a specific check
func (r *Runner) getCheckTimeout(checkName string) time.Duration {
	switch checkName {
	case checkNameFumpt:
		return time.Duration(r.config.CheckTimeouts.Fumpt) * time.Second
	case checkNameGitleaks:
		return time.Duration(r.config.CheckTimeouts.Gitleaks) * time.Second
	case checkNameLint:
		return time.Duration(r.config.CheckTimeouts.Lint) * time.Second
	case checkNameModTidy:
		return time.Duration(r.config.CheckTimeouts.ModTidy) * time.Second
	case checkNameWhitespace:
		return time.Duration(r.config.CheckTimeouts.Whitespace) * time.Second
	case checkNameEOF:
		return time.Duration(r.config.CheckTimeouts.EOF) * time.Second
	default:
		return time.Duration(r.config.Timeout) * time.Second
	}
}

// isCheckEnabled checks if a check is enabled in the configuration
func (r *Runner) isCheckEnabled(name string) bool {
	switch name {
	case checkNameEOF:
		return r.config.Checks.EOF
	case checkNameFumpt:
		return r.config.Checks.Fumpt
	case checkNameGitleaks:
		return r.config.Checks.Gitleaks
	case checkNameLint:
		return r.config.Checks.Lint
	case checkNameModTidy:
		return r.config.Checks.ModTidy
	case checkNameWhitespace:
		return r.config.Checks.Whitespace
	default:
		return false
	}
}

// applyExcludePatterns filters out files matching configured exclude patterns
func (r *Runner) applyExcludePatterns(files []string) []string {
	patterns := r.config.Git.ExcludePatterns
	if len(patterns) == 0 {
		return files
	}

	filtered := make([]string, 0, len(files))
	for _, file := range files {
		excluded := false
		for _, pattern := range patterns {
			if matchesExcludePattern(file, pattern) {
				excluded = true
				break
			}
		}
		if !excluded {
			filtered = append(filtered, file)
		}
	}
	return filtered
}

// matchesExcludePattern checks if a file path matches an exclude pattern
func matchesExcludePattern(filePath, pattern string) bool {
	// Empty pattern matches nothing
	if pattern == "" {
		return false
	}
	// Directory pattern (ends with /)
	if strings.HasSuffix(pattern, "/") {
		// Match if file is in this directory or subdirectory
		return strings.Contains(filePath, pattern) || strings.HasPrefix(filePath, pattern)
	}
	// Exact or substring match
	return strings.Contains(filePath, pattern)
}

// combineSkipSources processes SKIP environment variables and combines them with CLI skip options
func (r *Runner) combineSkipSources(cliSkips []string) []string {
	// Start with CLI skip options
	allSkips := make([]string, 0)
	if len(cliSkips) > 0 {
		allSkips = append(allSkips, cliSkips...)
	}

	// Process environment variables in order of precedence
	envSkips := r.processSkipEnvironment()
	if len(envSkips) > 0 {
		allSkips = append(allSkips, envSkips...)
	}

	// Remove duplicates and validate
	return r.deduplicateAndValidateSkips(allSkips)
}

// processSkipEnvironment reads and processes SKIP-related environment variables
func (r *Runner) processSkipEnvironment() []string {
	var skips []string

	// Check multiple environment variables in order of precedence
	skipEnvVars := []string{
		envSkip,              // Standard pre-commit convention
		"GO_PRE_COMMIT_SKIP", // GoFortress-specific
	}

	for _, envVar := range skipEnvVars {
		if value := strings.TrimSpace(os.Getenv(envVar)); value != "" {
			// Parse the skip value (comma-separated list)
			parsed := r.parseSkipValue(value)
			if len(parsed) > 0 {
				skips = append(skips, parsed...)
				// Use the first non-empty environment variable found
				break
			}
		}
	}

	return skips
}

// parseSkipValue parses a SKIP environment variable value
func (r *Runner) parseSkipValue(value string) []string {
	if value == "" {
		return nil
	}

	// Handle special values
	if strings.ToLower(value) == "all" {
		return []string{checkNameFumpt, checkNameGitleaks, checkNameLint, checkNameModTidy, checkNameWhitespace, checkNameEOF}
	}

	// Split by comma and clean up
	parts := strings.Split(value, ",")
	var skips []string
	var hasContent bool // Track if we found any non-empty content
	validChecks := map[string]bool{
		checkNameFumpt:      true,
		checkNameGitleaks:   true,
		checkNameLint:       true,
		checkNameModTidy:    true,
		checkNameWhitespace: true,
		checkNameEOF:        true,
	}
	for _, part := range parts {
		if cleaned := strings.TrimSpace(part); cleaned != "" {
			hasContent = true // Found non-empty content
			// Only add valid check names
			if validChecks[cleaned] {
				skips = append(skips, cleaned)
			}
		}
	}

	// If no content was found (only commas/whitespace), return nil
	if !hasContent {
		return nil
	}

	// If content was found but nothing valid, return empty slice
	if skips == nil {
		return []string{}
	}

	return skips
}

// deduplicateAndValidateSkips removes duplicates and validates skip names
func (r *Runner) deduplicateAndValidateSkips(skips []string) []string {
	seen := make(map[string]bool)
	result := make([]string, 0, len(skips))

	validChecks := map[string]bool{
		checkNameFumpt:      true,
		checkNameGitleaks:   true,
		checkNameLint:       true,
		checkNameModTidy:    true,
		checkNameWhitespace: true,
		checkNameEOF:        true,
	}

	for _, skip := range skips {
		skip = strings.TrimSpace(skip)
		if skip == "" {
			continue
		}

		// Skip duplicates
		if seen[skip] {
			continue
		}

		// Validate check name
		if !validChecks[skip] {
			// Log warning for invalid check names but don't fail
			// This allows for future extensibility
			continue
		}

		seen[skip] = true
		result = append(result, skip)
	}

	return result
}
