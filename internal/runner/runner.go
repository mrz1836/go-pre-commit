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
	parallel := opts.Parallel
	if parallel <= 0 {
		parallel = r.config.Performance.ParallelWorkers
		if parallel <= 0 {
			parallel = runtime.NumCPU()
		}
	}

	// Create context with timeout
	globalTimeout := time.Duration(r.config.Timeout) * time.Second
	ctxWithTimeout, cancel := context.WithTimeout(ctx, globalTimeout)
	defer cancel()

	// Debug timeout information
	if opts.DebugTimeout {
		fmt.Fprintf(os.Stderr, "🐛 [DEBUG-TIMEOUT] Global timeout set to: %v\n", globalTimeout)
		fmt.Fprintf(os.Stderr, "🐛 [DEBUG-TIMEOUT] Tool installation timeout: %v\n", time.Duration(r.config.ToolInstallation.Timeout)*time.Second)
		if r.config.Environment.IsCI {
			fmt.Fprintf(os.Stderr, "🐛 [DEBUG-TIMEOUT] CI environment detected: %s (auto-adjust: %t)\n", r.config.Environment.CIProvider, r.config.Environment.AutoAdjustTimers)
		}
	}

	// Run checks
	results := &Results{
		CheckResults: make([]CheckResult, 0, len(checksToRun)),
		TotalFiles:   len(opts.Files),
	}

	if opts.FailFast {
		// Sequential execution with fail-fast
		for _, check := range checksToRun {
			if opts.ProgressCallback != nil {
				opts.ProgressCallback(check.Name(), "running", 0)
			}

			result := r.runCheck(ctxWithTimeout, check, opts.Files, opts.GracefulDegradation, opts.DebugTimeout)
			results.CheckResults = append(results.CheckResults, result)

			if result.Success {
				results.Passed++
				if opts.ProgressCallback != nil {
					opts.ProgressCallback(check.Name(), "passed", result.Duration)
				}
			} else if result.CanSkip && opts.GracefulDegradation {
				results.Skipped++
				if opts.ProgressCallback != nil {
					opts.ProgressCallback(check.Name(), "skipped", result.Duration)
				}
			} else {
				results.Failed++
				if opts.ProgressCallback != nil {
					opts.ProgressCallback(check.Name(), "failed", result.Duration)
				}
				break // Stop on first failure
			}
		}
	} else {
		// Parallel execution
		resultsChan := make(chan CheckResult, len(checksToRun))
		var wg sync.WaitGroup
		semaphore := make(chan struct{}, parallel)

		for _, check := range checksToRun {
			wg.Add(1)
			go func(c checks.Check) {
				defer wg.Done()

				semaphore <- struct{}{}
				defer func() { <-semaphore }()

				if opts.ProgressCallback != nil {
					opts.ProgressCallback(c.Name(), "running", 0)
				}

				result := r.runCheck(ctxWithTimeout, c, opts.Files, opts.GracefulDegradation, opts.DebugTimeout)
				resultsChan <- result
			}(check)
		}

		wg.Wait()
		close(resultsChan)

		// Collect results
		for result := range resultsChan {
			results.CheckResults = append(results.CheckResults, result)
			if result.Success {
				results.Passed++
				if opts.ProgressCallback != nil {
					opts.ProgressCallback(result.Name, "passed", result.Duration)
				}
			} else if result.CanSkip && opts.GracefulDegradation {
				results.Skipped++
				if opts.ProgressCallback != nil {
					opts.ProgressCallback(result.Name, "skipped", result.Duration)
				}
			} else {
				results.Failed++
				if opts.ProgressCallback != nil {
					opts.ProgressCallback(result.Name, "failed", result.Duration)
				}
			}
		}
	}

	results.TotalDuration = time.Since(start)
	return results, nil
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

	// Run the check
	err := check.Run(ctx, filteredFiles)

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
	case "fmt":
		return time.Duration(r.config.CheckTimeouts.Fmt) * time.Second
	case "fumpt":
		return time.Duration(r.config.CheckTimeouts.Fumpt) * time.Second
	case "gitleaks":
		return time.Duration(r.config.CheckTimeouts.Gitleaks) * time.Second
	case "goimports":
		return time.Duration(r.config.CheckTimeouts.Goimports) * time.Second
	case "lint":
		return time.Duration(r.config.CheckTimeouts.Lint) * time.Second
	case "mod-tidy":
		return time.Duration(r.config.CheckTimeouts.ModTidy) * time.Second
	case "whitespace":
		return time.Duration(r.config.CheckTimeouts.Whitespace) * time.Second
	case "eof":
		return time.Duration(r.config.CheckTimeouts.EOF) * time.Second
	case "ai_detection":
		return time.Duration(r.config.CheckTimeouts.AIDetection) * time.Second
	default:
		return time.Duration(r.config.Timeout) * time.Second
	}
}

// isCheckEnabled checks if a check is enabled in the configuration
func (r *Runner) isCheckEnabled(name string) bool {
	switch name {
	case "ai_detection":
		return r.config.Checks.AIDetection
	case "eof":
		return r.config.Checks.EOF
	case "fmt":
		return r.config.Checks.Fmt
	case "fumpt":
		return r.config.Checks.Fumpt
	case "gitleaks":
		return r.config.Checks.Gitleaks
	case "goimports":
		return r.config.Checks.Goimports
	case "lint":
		return r.config.Checks.Lint
	case "mod-tidy":
		return r.config.Checks.ModTidy
	case "whitespace":
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
		"SKIP",               // Standard pre-commit convention
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
		return []string{"fmt", "fumpt", "gitleaks", "goimports", "lint", "mod-tidy", "whitespace", "eof", "ai_detection"}
	}

	// Split by comma and clean up
	parts := strings.Split(value, ",")
	var skips []string
	var hasContent bool // Track if we found any non-empty content
	validChecks := map[string]bool{
		"fmt":          true,
		"fumpt":        true,
		"gitleaks":     true,
		"goimports":    true,
		"lint":         true,
		"mod-tidy":     true,
		"whitespace":   true,
		"eof":          true,
		"ai_detection": true,
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
		"fmt":          true,
		"fumpt":        true,
		"gitleaks":     true,
		"goimports":    true,
		"lint":         true,
		"mod-tidy":     true,
		"whitespace":   true,
		"eof":          true,
		"ai_detection": true,
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
