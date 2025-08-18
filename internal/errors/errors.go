// Package errors defines common errors for the pre-commit system
package errors

import (
	"errors"
	"fmt"
)

// Common errors
var (
	// ErrChecksFailed is returned when one or more checks fail
	ErrChecksFailed = errors.New("checks failed")

	// ErrNoChecksToRun is returned when no checks are configured to run
	ErrNoChecksToRun = errors.New("no checks to run")

	// ErrEnvFileNotFound is returned when .env.base cannot be found
	ErrEnvFileNotFound = errors.New(".github/.env.base not found in any parent directory")

	// ErrRepositoryRootNotFound is returned when git repository root cannot be determined
	ErrRepositoryRootNotFound = errors.New("unable to determine repository root")

	// ErrToolNotFound is returned when a required tool is not available
	ErrToolNotFound = errors.New("required tool not found")

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

	// ErrMagexTargetNotFound is returned when a magex target is not available
	ErrMagexTargetNotFound = errors.New("magex target not found")

	// ErrToolExecutionFailed is returned when a tool execution fails
	ErrToolExecutionFailed = errors.New("tool execution failed")

	// ErrGracefulSkip is returned when a check is gracefully skipped
	ErrGracefulSkip = errors.New("check gracefully skipped")

	// Git-related errors
	ErrNotGitRepository     = errors.New("not a git repository")
	ErrUnsupportedHookType  = errors.New("unsupported hook type")
	ErrPreCommitDirNotExist = errors.New("pre-commit directory does not exist")
	ErrHookNotExecutable    = errors.New("hook file is not executable")
	ErrHookMarkerMissing    = errors.New("installed hook does not contain expected marker")

	// Magex-related errors
	ErrMagexTargetTimeout = errors.New("timeout checking magex target")
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

// NewMagexTargetNotFoundError creates an error for missing magex targets with graceful degradation
func NewMagexTargetNotFoundError(target, alternative string) *CheckError {
	return &CheckError{
		Err:        ErrMagexTargetNotFound,
		Message:    fmt.Sprintf("magex target '%s' not found", target),
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
