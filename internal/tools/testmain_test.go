package tools

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"testing"

	"github.com/mrz1836/go-pre-commit/internal/testutil"
)

// errOffline is returned for every release download a test makes without
// serving it from its own httptest server.
var errOffline = errors.New("release downloads are offline in tests")

// offlineTransport fails every request, so a test that reaches a release
// download without a fake release server fails fast instead of reaching the
// network.
type offlineTransport struct{}

func (offlineTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, errOffline
}

// TestMain clears go-pre-commit configuration inherited from the environment.
// Developer shells and task runners such as magex export the repository's
// .github/env files (for example GO_PRE_COMMIT_FUMPT_VERSION_LATEST), which
// would otherwise change the versions LoadVersionsFromEnv selects. Tests that
// need a value set it with t.Setenv, which restores this cleared state.
//
// It also takes release downloads offline: tests that install from a release
// serve it with newReleaseServer, and any other download fails.
func TestMain(m *testing.M) {
	if err := testutil.ClearConfigEnv(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to clear inherited configuration: %v\n", err)
		os.Exit(1)
	}
	releaseTransport = offlineTransport{}
	os.Exit(m.Run())
}

// resetToolState gives the test a fresh default tool registry and an empty
// installation cache, restoring the previous ones on cleanup. Each test gets
// its own Tool values and cache, so versions set by LoadVersionsFromEnv, tools
// added by the test, or tools cached as installed (including real binaries
// found on PATH) cannot leak into other tests, shuffled orders, or later
// -count iterations.
func resetToolState(tb testing.TB) {
	tb.Helper()

	toolsMu.Lock()
	previousTools := tools
	tools = newToolRegistry()
	toolsMu.Unlock()

	installMu.Lock()
	previousInstalled := installedTools
	installedTools = make(map[string]bool)
	installMu.Unlock()

	tb.Cleanup(func() {
		toolsMu.Lock()
		tools = previousTools
		toolsMu.Unlock()

		installMu.Lock()
		installedTools = previousInstalled
		installMu.Unlock()
	})
}
