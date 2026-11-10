package tools

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// errSimulatedNetwork is a network-looking error, so the installer retries it
var errSimulatedNetwork = errors.New("dial tcp: connection refused")

// captureInstallOutput sets the installer's message output to a buffer for
// the rest of the test, runs fn with os.Stdout replaced by a pipe, and returns
// what reached the buffer and what reached stdout
func captureInstallOutput(t *testing.T, fn func()) (messages, stdout string) {
	t.Helper()
	var buf bytes.Buffer
	previous := SetOutput(&buf)
	t.Cleanup(func() { SetOutput(previous) })

	r, w, err := os.Pipe()
	require.NoError(t, err)
	done := make(chan string)
	go func() {
		var out bytes.Buffer
		_, _ = io.Copy(&out, r)
		done <- out.String()
	}()

	func() {
		oldStdout := os.Stdout
		os.Stdout = w
		defer func() { os.Stdout = oldStdout }()
		fn()
	}()

	_ = w.Close()
	return buf.String(), <-done
}

// TestSetOutputRoutesInstallMessages tests that the installer's messages go to
// the writer set with SetOutput instead of stdout
func TestSetOutputRoutesInstallMessages(t *testing.T) {
	resetToolState(t)
	setupInstallDirs(t)
	t.Setenv("HOME", t.TempDir())
	fakeInstall(t, func(context.Context, []string, string, ...string) ([]byte, error) {
		return nil, errSimulatedNetwork
	})

	origAttempts, origDelay := GetRetryConfig()
	t.Cleanup(func() { SetRetryConfig(origAttempts, origDelay) })
	SetRetryConfig(2, time.Millisecond)

	var installErr error
	messages, stdout := captureInstallOutput(t, func() {
		installErr = InstallTool(t.Context(), &Tool{
			Name:       "test-tool-output",
			ImportPath: "example.com/test-tool-output",
			Version:    "v1.0.0",
			Binary:     "test-tool-output",
		})
	})

	require.Error(t, installErr)
	assert.Empty(t, stdout, "nothing may reach stdout once an output is set")
	assert.Contains(t, messages, "Installing test-tool-output@v1.0.0")
	assert.Contains(t, messages, "Retrying installing test-tool-output (attempt 2/2)")
}

// TestSetOutputRoutesGolangciLintFallbackMessages tests that when the
// golangci-lint release download fails and the go install fallback runs, both
// fallback messages go to the writer set with SetOutput and never to stdout
func TestSetOutputRoutesGolangciLintFallbackMessages(t *testing.T) {
	resetToolState(t)
	setupInstallDirs(t)
	t.Setenv("HOME", t.TempDir())

	origAttempts, origDelay := GetRetryConfig()
	t.Cleanup(func() { SetRetryConfig(origAttempts, origDelay) })
	SetRetryConfig(1, time.Millisecond)

	// The fake release server publishes nothing, so the download fails
	a := golangciLintAsset("v2.15.0", runtime.GOOS, runtime.GOARCH)
	rs := newReleaseServer(t)

	var commands []string
	fakeInstall(t, func(_ context.Context, _ []string, name string, args ...string) ([]byte, error) {
		commands = append(commands, name+" "+strings.Join(args, " "))
		return []byte("go install output"), nil
	})

	var installErr error
	messages, stdout := captureInstallOutput(t, func() {
		installErr = installGolangciLint(t.Context(), "v2.15.0")
	})

	require.NoError(t, installErr)
	assert.Positive(t, rs.requested(assetPath(a, a.checksums)), "the release download was tried first")
	const target = "github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.15.0"
	assert.Equal(t, []string{"go install " + target}, commands)

	assert.Empty(t, stdout, "nothing may reach stdout once an output is set")
	assert.Contains(t, messages, "Trying go install "+target+" instead...")
	assert.Contains(t, messages, "Installed golangci-lint with go install (took ")
}

// TestOutputDefaultsToStdout tests that a nil output means the current stdout,
// and that SetOutput returns the setting it replaced
func TestOutputDefaultsToStdout(t *testing.T) {
	var buf bytes.Buffer
	previous := SetOutput(nil)
	t.Cleanup(func() { SetOutput(previous) })
	assert.Equal(t, os.Stdout, Output())

	assert.Nil(t, SetOutput(&buf))
	assert.Equal(t, &buf, SetOutput(nil))
}
