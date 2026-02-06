package errors_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	pkgerrors "github.com/mrz1836/go-pre-commit/internal/errors"
)

// Test error variables to satisfy err113 linter
var (
	errTestBase     = errors.New("base error")
	errTestOriginal = errors.New("original error")
)

type ErrorTestSuite struct {
	suite.Suite
}

func TestErrorSuite(t *testing.T) {
	suite.Run(t, new(ErrorTestSuite))
}

// TestCommonErrors tests that all common errors are properly defined
func (s *ErrorTestSuite) TestCommonErrors() {
	tests := []struct {
		name     string
		err      error
		expected string
	}{
		{"ErrChecksFailed", pkgerrors.ErrChecksFailed, "checks failed"},
		{"ErrNoChecksToRun", pkgerrors.ErrNoChecksToRun, "no checks to run"},
		{"ErrEnvFileNotFound", pkgerrors.ErrEnvFileNotFound, "failed to find environment configuration (.github/env/ directory or .github/.env.base)"},
		{"ErrRepositoryRootNotFound", pkgerrors.ErrRepositoryRootNotFound, "unable to determine repository root"},
		{"ErrToolNotFound", pkgerrors.ErrToolNotFound, "required tool not found"},
		{"ErrLintingIssues", pkgerrors.ErrLintingIssues, "linting issues found"},
		{"ErrNotTidy", pkgerrors.ErrNotTidy, "go.mod or go.sum are not tidy"},
		{"ErrWhitespaceIssues", pkgerrors.ErrWhitespaceIssues, "whitespace issues found"},
		{"ErrEOFIssues", pkgerrors.ErrEOFIssues, "EOF issues found"},
		{"ErrToolExecutionFailed", pkgerrors.ErrToolExecutionFailed, "tool execution failed"},
		{"ErrGracefulSkip", pkgerrors.ErrGracefulSkip, "check gracefully skipped"},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.Require().Error(tt.err)
			s.Equal(tt.expected, tt.err.Error())
		})
	}
}

// TestCheckErrorConstructor tests the CheckError constructor
func (s *ErrorTestSuite) TestCheckErrorConstructor() {
	baseErr := errTestBase
	message := "something went wrong"
	suggestion := "try this fix"

	checkErr := pkgerrors.NewCheckError(baseErr, message, suggestion)

	s.NotNil(checkErr)
	s.Equal(baseErr, checkErr.Err)
	s.Equal(message, checkErr.Message)
	s.Equal(suggestion, checkErr.Suggestion)
	s.False(checkErr.CanSkip)
}

// TestCheckErrorError tests the Error method
func (s *ErrorTestSuite) TestCheckErrorError() {
	tests := []struct {
		name     string
		checkErr *pkgerrors.CheckError
		expected string
	}{
		{
			name: "message takes precedence",
			checkErr: &pkgerrors.CheckError{
				Err:     errTestBase,
				Message: "custom message",
			},
			expected: "custom message",
		},
		{
			name: "falls back to base error",
			checkErr: &pkgerrors.CheckError{
				Err: errTestBase,
			},
			expected: "base error",
		},
		{
			name:     "unknown error when both are empty",
			checkErr: &pkgerrors.CheckError{},
			expected: "unknown error",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.Equal(tt.expected, tt.checkErr.Error())
		})
	}
}

// TestCheckErrorUnwrap tests the Unwrap method
func (s *ErrorTestSuite) TestCheckErrorUnwrap() {
	baseErr := errTestBase
	checkErr := &pkgerrors.CheckError{Err: baseErr}

	unwrapped := checkErr.Unwrap()
	s.Equal(baseErr, unwrapped)
}

// TestCheckErrorIs tests the Is method
func (s *ErrorTestSuite) TestCheckErrorIs() {
	baseErr := pkgerrors.ErrToolNotFound
	checkErr := &pkgerrors.CheckError{Err: baseErr}

	s.True(checkErr.Is(pkgerrors.ErrToolNotFound))
	s.False(checkErr.Is(pkgerrors.ErrLintingIssues))
}

// TestNewToolNotFoundError tests the tool not found error constructor
func (s *ErrorTestSuite) TestNewToolNotFoundError() {
	tool := "golangci-lint"
	alternative := "try installing with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"

	err := pkgerrors.NewToolNotFoundError(tool, alternative)

	s.NotNil(err)
	s.True(err.Is(pkgerrors.ErrToolNotFound))
	s.Equal("golangci-lint not found", err.Message)
	s.Equal(alternative, err.Suggestion)
	s.True(err.CanSkip)
}

// TestNewToolExecutionError tests the tool execution error constructor
func (s *ErrorTestSuite) TestNewToolExecutionError() {
	command := "golangci-lint run"
	output := "some error output"
	suggestion := "fix the issues found"

	err := pkgerrors.NewToolExecutionError(command, output, suggestion)

	s.NotNil(err)
	s.True(err.Is(pkgerrors.ErrToolExecutionFailed))
	s.Equal("command 'golangci-lint run' failed", err.Message)
	s.Equal(command, err.Command)
	s.Equal(output, err.Output)
	s.Equal(suggestion, err.Suggestion)
	s.False(err.CanSkip)
}

// TestNewGracefulSkipError tests the graceful skip error constructor
func (s *ErrorTestSuite) TestNewGracefulSkipError() {
	reason := "check requirements not met"

	err := pkgerrors.NewGracefulSkipError(reason)

	s.NotNil(err)
	s.True(err.Is(pkgerrors.ErrGracefulSkip))
	s.Equal(reason, err.Message)
	s.Equal("This check was skipped to allow other checks to continue", err.Suggestion)
	s.True(err.CanSkip)
}

// TestCheckErrorChaining tests error chaining and wrapping
func (s *ErrorTestSuite) TestCheckErrorChaining() {
	originalErr := errTestOriginal
	wrappedErr := pkgerrors.NewCheckError(originalErr, "wrapped message", "fix suggestion")

	// Test that we can unwrap to the original error
	s.Require().ErrorIs(wrappedErr, originalErr)
	s.Equal(originalErr, errors.Unwrap(wrappedErr))

	// Test error chaining with standard library
	var checkErr *pkgerrors.CheckError
	s.Require().ErrorAs(wrappedErr, &checkErr)
	s.Equal(wrappedErr, checkErr)
}

// TestCheckErrorFields tests all fields of CheckError
func (s *ErrorTestSuite) TestCheckErrorFields() {
	err := &pkgerrors.CheckError{
		Err:        pkgerrors.ErrLintingIssues,
		Message:    "custom message",
		Suggestion: "fix the linting issues",
		Command:    "golangci-lint run",
		Output:     "error output",
		Files:      []string{"main.go", "test.go"},
		CanSkip:    true,
	}

	s.Equal(pkgerrors.ErrLintingIssues, err.Err)
	s.Equal("custom message", err.Message)
	s.Equal("fix the linting issues", err.Suggestion)
	s.Equal("golangci-lint run", err.Command)
	s.Equal("error output", err.Output)
	s.Equal([]string{"main.go", "test.go"}, err.Files)
	s.True(err.CanSkip)
}

// Unit tests for edge cases
func TestCheckErrorNilWrapping(t *testing.T) {
	checkErr := &pkgerrors.CheckError{Err: nil}
	require.NoError(t, checkErr.Unwrap())
	assert.False(t, checkErr.Is(pkgerrors.ErrToolNotFound))
}

func TestCheckErrorEmptyMessage(t *testing.T) {
	checkErr := &pkgerrors.CheckError{
		Err:     pkgerrors.ErrToolNotFound,
		Message: "",
	}
	assert.Equal(t, "required tool not found", checkErr.Error())
}

func TestErrorComparisons(t *testing.T) {
	// Test that our predefined errors are distinct
	assert.NotEqual(t, pkgerrors.ErrChecksFailed, pkgerrors.ErrNoChecksToRun)
	assert.NotEqual(t, pkgerrors.ErrToolNotFound, pkgerrors.ErrLintingIssues)
}

func TestErrorWrappingWithStandardLibrary(t *testing.T) {
	originalErr := pkgerrors.ErrToolNotFound
	wrappedErr := pkgerrors.NewCheckError(originalErr, "custom message", "fix it")

	// Test with errors.Is
	require.ErrorIs(t, wrappedErr, pkgerrors.ErrToolNotFound)
	require.NotErrorIs(t, wrappedErr, pkgerrors.ErrLintingIssues)

	// Test with errors.As
	var checkErr *pkgerrors.CheckError
	require.ErrorAs(t, wrappedErr, &checkErr)
	assert.Equal(t, "custom message", checkErr.Message)
}

func TestAllConstructors(t *testing.T) {
	tests := []struct {
		name        string
		constructor func() *pkgerrors.CheckError
		expectedErr error
		canSkip     bool
	}{
		{
			name: "NewCheckError",
			constructor: func() *pkgerrors.CheckError {
				return pkgerrors.NewCheckError(pkgerrors.ErrLintingIssues, "msg", "suggestion")
			},
			expectedErr: pkgerrors.ErrLintingIssues,
			canSkip:     false,
		},
		{
			name:        "NewToolNotFoundError",
			constructor: func() *pkgerrors.CheckError { return pkgerrors.NewToolNotFoundError("tool", "alt") },
			expectedErr: pkgerrors.ErrToolNotFound,
			canSkip:     true,
		},
		{
			name:        "NewToolExecutionError",
			constructor: func() *pkgerrors.CheckError { return pkgerrors.NewToolExecutionError("cmd", "output", "suggestion") },
			expectedErr: pkgerrors.ErrToolExecutionFailed,
			canSkip:     false,
		},
		{
			name:        "NewGracefulSkipError",
			constructor: func() *pkgerrors.CheckError { return pkgerrors.NewGracefulSkipError("reason") },
			expectedErr: pkgerrors.ErrGracefulSkip,
			canSkip:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.constructor()
			assert.NotNil(t, err)
			assert.True(t, err.Is(tt.expectedErr))
			assert.Equal(t, tt.canSkip, err.CanSkip)
			assert.NotEmpty(t, err.Error())
		})
	}
}
