package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	prerrors "github.com/mrz1836/go-pre-commit/internal/errors"
	"github.com/mrz1836/go-pre-commit/internal/tools"
)

// `status --json` and `run --json` write exactly one JSON document to stdout.
// Both documents follow the same conventions: snake_case field names, a
// "version" field holding the go-pre-commit version, lists that are arrays
// (empty, never null, unless a field's documentation says otherwise), and
// durations as whole milliseconds in "duration_ms" fields. When a command
// cannot produce its report, it writes an error document instead,
// {"version": ..., "error": {"code": ..., "message": ...}}, and exits non-zero.
// Fields may be added in later releases; existing ones keep their meaning.

// jsonFlag is the flag that selects JSON output
const jsonFlag = "json"

// Codes of the JSON error document. A code names the cause; the message is
// for people and may change.
const (
	errCodeConfigNotFound = "config_not_found" // no .github/env/*.env and no .github/.env.base
	errCodeConfigInvalid  = "config_invalid"   // the configuration could not be read or failed validation
	errCodeNotARepository = "not_a_repository" // not inside a git repository
	errCodeGitFailed      = "git_failed"       // git could not list the files to check
	errCodeNoChecksToRun  = "no_checks_to_run" // the selected checks match no enabled check
	errCodeRunFailed      = "run_failed"       // any other failure before the checks could report
)

// errorReport is the document written when a command fails before it can report
type errorReport struct {
	Version string      `json:"version"`
	Error   errorObject `json:"error"`
}

// errorObject is the "error" object of an error document
type errorObject struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// codedError attaches the JSON error code to an error. Its text is the
// wrapped error's, so the text output and exit status are unchanged.
type codedError struct {
	code string
	err  error
}

// Error returns the wrapped error's text
func (e *codedError) Error() string {
	return e.err.Error()
}

// Unwrap returns the wrapped error
func (e *codedError) Unwrap() error {
	return e.err
}

// withCode attaches a JSON error code to err
func withCode(code string, err error) error {
	return &codedError{code: code, err: err}
}

// errorCode returns the JSON error code attached to err, or errCodeRunFailed
func errorCode(err error) string {
	var coded *codedError
	if errors.As(err, &coded) {
		return coded.code
	}
	return errCodeRunFailed
}

// configErrorCode returns the JSON error code for a config.Load error
func configErrorCode(err error) string {
	if errors.Is(err, prerrors.ErrEnvFileNotFound) {
		return errCodeConfigNotFound
	}
	return errCodeConfigInvalid
}

// jsonOutputRequested reports whether cmd was given --json. A nil command (as
// in tests that call a command's function directly) means text output.
func jsonOutputRequested(cmd *cobra.Command) (bool, error) {
	if cmd == nil || cmd.Flags().Lookup(jsonFlag) == nil {
		return false, nil
	}
	return cmd.Flags().GetBool(jsonFlag)
}

// stdoutOf returns where cmd writes its standard output
func stdoutOf(cmd *cobra.Command) io.Writer {
	if cmd == nil {
		return os.Stdout
	}
	return cmd.OutOrStdout()
}

// writeJSON writes v to w as one indented JSON document
func writeJSON(w io.Writer, v any) error {
	encoder := json.NewEncoder(w)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(v); err != nil {
		return fmt.Errorf("failed to write JSON output: %w", err)
	}
	return nil
}

// writeErrorReport writes the error document for cause to w and returns cause,
// so the command still exits non-zero
func writeErrorReport(w io.Writer, version string, cause error) error {
	report := errorReport{
		Version: version,
		Error:   errorObject{Code: errorCode(cause), Message: cause.Error()},
	}
	if err := writeJSON(w, report); err != nil {
		return errors.Join(cause, err)
	}
	return cause
}

// routeHumanOutputToStderr sends the messages that other packages print to
// stdout (tool installation progress) to stderr, so stdout carries only the
// JSON document. It returns a function that restores the previous setting.
func routeHumanOutputToStderr() (restore func()) {
	previous := tools.SetOutput(os.Stderr)
	return func() { tools.SetOutput(previous) }
}
