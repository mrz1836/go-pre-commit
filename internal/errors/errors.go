// Package errors defines common errors for the pre-commit system
package errors

import (
	"errors"
	"fmt"
	"time"
)

// Common errors
var (
	// ErrChecksFailed is returned when one or more checks fail
	ErrChecksFailed = errors.New("checks failed")

	// ErrNoChecksToRun is returned when no checks are configured to run
	ErrNoChecksToRun = errors.New("no checks to run")

	// ErrEnvFileNotFound is returned when .env.base cannot be found
	ErrEnvFileNotFound = errors.New("failed to find .env.base")

	// ErrRepositoryRootNotFound is returned when git repository root cannot be determined
	ErrRepositoryRootNotFound = errors.New("unable to determine repository root")

	// ErrToolNotFound is returned when a required tool is not available
	ErrToolNotFound = errors.New("required tool not found")

	// ErrFileNotFound is returned when a file is expected but not found
	ErrFileNotFound = errors.New("file not found")

	// ErrFileStillExists is returned when a file is expected to be deleted but still exists
	ErrFileStillExists = errors.New("file expected to be deleted but still exists")

	// ErrFmtIssues is returned when go fmt finds formatting issues
	ErrFmtIssues = errors.New("formatting issues found")

	// ErrLintingIssues is returned when linting finds issues
	ErrLintingIssues = errors.New("linting issues found")

	// ErrNotTidy is returned when go.mod/go.sum are not tidy
	ErrNotTidy = errors.New("go.mod or go.sum are not tidy")

	// ErrWhitespaceIssues is returned when whitespace issues are found
	ErrWhitespaceIssues = errors.New("whitespace issues found")

	// ErrEOFIssues is returned when EOF issues are found
	ErrEOFIssues = errors.New("EOF issues found")

	// ErrAIAttributionFound is returned when AI attribution is detected
	ErrAIAttributionFound = errors.New("AI attribution detected")

	// ErrSecretsFound is returned when gitleaks finds secrets
	ErrSecretsFound = errors.New("secrets found")

	// ErrToolExecutionFailed is returned when a tool execution fails
	ErrToolExecutionFailed = errors.New("tool execution failed")

	// ErrGracefulSkip is returned when a check is gracefully skipped
	ErrGracefulSkip = errors.New("check gracefully skipped")

	// ErrNilContext is returned when a nil context is provided
	ErrNilContext = errors.New("context cannot be nil")

	// ErrTimeout is returned when an operation times out
	ErrTimeout = errors.New("operation timed out")

	// Git-related errors
	ErrNotGitRepository      = errors.New("not a git repository")
	ErrGitBaseCommitNotFound = errors.New("could not determine git base commit")
	ErrUnsupportedHookType   = errors.New("unsupported hook type")
	ErrPreCommitDirNotExist  = errors.New("pre-commit directory does not exist")
	ErrHookNotExecutable     = errors.New("hook file is not executable")
	ErrHookMarkerMissing     = errors.New("installed hook does not contain expected marker")
)

// CheckError represents an enhanced error with context and suggestions
type CheckError struct {
	// Base error
	Err error

	// Human-readable message explaining what went wrong
	Message string

	// Actionable suggestion for how to fix the issue
	Suggestion string

	// Command that failed (if applicable)
	Command string

	// Raw output from the failed command
	Output string

	// Files that were being processed
	Files []string

	// Whether this error allows graceful degradation
	CanSkip bool
}

// TimeoutError represents a timeout error with detailed context
type TimeoutError struct {
	// Base error
	Err error

	// Operation that timed out (e.g., "tool installation", "check execution")
	Operation string

	// Specific context (e.g., tool name, check name)
	Context string

	// Duration of the timeout that was applied
	Timeout time.Duration

	// Duration that elapsed before timeout
	Elapsed time.Duration

	// Configuration variable that can be used to adjust the timeout
	ConfigVar string

	// Suggested new timeout value
	SuggestedTimeout time.Duration
}

// Error implements the error interface for TimeoutError
func (e *TimeoutError) Error() string {
	baseMsg := fmt.Sprintf("%s timed out after %v", e.Operation, e.Timeout)
	if e.Context != "" {
		baseMsg = fmt.Sprintf("%s (%s) timed out after %v", e.Operation, e.Context, e.Timeout)
	}

	if e.ConfigVar != "" {
		baseMsg += fmt.Sprintf(". Consider increasing %s", e.ConfigVar)
		if e.SuggestedTimeout > 0 {
			baseMsg += fmt.Sprintf(" (suggested: %v)", e.SuggestedTimeout)
		}
	}

	return baseMsg
}

// Unwrap implements the error unwrapping interface for TimeoutError
func (e *TimeoutError) Unwrap() error {
	return e.Err
}

// Is implements the error checking interface for TimeoutError
func (e *TimeoutError) Is(target error) bool {
	return errors.Is(e.Err, target)
}

// Error implements the error interface
func (e *CheckError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	if e.Err != nil {
		return e.Err.Error()
	}
	return "unknown error"
}

// Unwrap implements the error unwrapping interface
func (e *CheckError) Unwrap() error {
	return e.Err
}

// Is implements the error checking interface
func (e *CheckError) Is(target error) bool {
	return errors.Is(e.Err, target)
}

// NewCheckError creates a new CheckError
func NewCheckError(err error, message, suggestion string) *CheckError {
	return &CheckError{
		Err:        err,
		Message:    message,
		Suggestion: suggestion,
	}
}

// NewToolNotFoundError creates an error for missing tools with graceful degradation
func NewToolNotFoundError(tool, alternative string) *CheckError {
	return &CheckError{
		Err:        ErrToolNotFound,
		Message:    fmt.Sprintf("%s not found", tool),
		Suggestion: alternative,
		CanSkip:    true,
	}
}

// NewToolExecutionError creates an error for tool execution failures
func NewToolExecutionError(command, output, suggestion string) *CheckError {
	return &CheckError{
		Err:        ErrToolExecutionFailed,
		Command:    command,
		Output:     output,
		Message:    fmt.Sprintf("command '%s' failed", command),
		Suggestion: suggestion,
	}
}

// NewGracefulSkipError creates an error for gracefully skipped checks
func NewGracefulSkipError(reason string) *CheckError {
	return &CheckError{
		Err:        ErrGracefulSkip,
		Message:    reason,
		Suggestion: "This check was skipped to allow other checks to continue",
		CanSkip:    true,
	}
}

// NewTimeoutError creates a new TimeoutError with context
func NewTimeoutError(operation, context string, timeout, elapsed time.Duration, configVar string) *TimeoutError {
	// Suggest a timeout that's 1.5x the current timeout or 2x the elapsed time, whichever is larger
	suggestedTimeout := timeout * 3 / 2
	if elapsed > 0 && elapsed*2 > suggestedTimeout {
		suggestedTimeout = elapsed * 2
	}

	return &TimeoutError{
		Err:              ErrTimeout,
		Operation:        operation,
		Context:          context,
		Timeout:          timeout,
		Elapsed:          elapsed,
		ConfigVar:        configVar,
		SuggestedTimeout: suggestedTimeout,
	}
}

// NewToolInstallTimeoutError creates a timeout error specifically for tool installation
func NewToolInstallTimeoutError(toolName string, timeout, elapsed time.Duration) *TimeoutError {
	return NewTimeoutError("Tool installation", toolName, timeout, elapsed, "GO_PRE_COMMIT_TOOL_INSTALL_TIMEOUT")
}

// NewCheckTimeoutError creates a timeout error specifically for check execution
func NewCheckTimeoutError(checkName string, timeout, elapsed time.Duration) *TimeoutError {
	configVar := "GO_PRE_COMMIT_TIMEOUT_SECONDS"
	switch checkName {
	case "fmt":
		configVar = "GO_PRE_COMMIT_FMT_TIMEOUT"
	case "fumpt":
		configVar = "GO_PRE_COMMIT_FUMPT_TIMEOUT"
	case "goimports":
		configVar = "GO_PRE_COMMIT_GOIMPORTS_TIMEOUT"
	case "lint":
		configVar = "GO_PRE_COMMIT_LINT_TIMEOUT"
	case "mod-tidy":
		configVar = "GO_PRE_COMMIT_MOD_TIDY_TIMEOUT"
	case "whitespace":
		configVar = "GO_PRE_COMMIT_WHITESPACE_TIMEOUT"
	case "eof":
		configVar = "GO_PRE_COMMIT_EOF_TIMEOUT"
	case "ai_detection":
		configVar = "GO_PRE_COMMIT_AI_DETECTION_TIMEOUT"
	case "gitleaks":
		configVar = "GO_PRE_COMMIT_GITLEAKS_TIMEOUT"
	}

	return NewTimeoutError("Check execution", checkName, timeout, elapsed, configVar)
}
