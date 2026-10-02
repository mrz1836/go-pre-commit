// Package testutil provides helpers shared by the module's tests.
package testutil

import (
	"os"
	"strings"
	"testing"
)

// IsolateEnv snapshots the entire process environment and restores it exactly
// when tb finishes: variables added during the test are unset, and changed or
// removed ones are put back. Use it in tests that call config.Load or the
// envfile loaders, which write every configuration value they read into the
// process environment (and, in legacy mode, never override values already
// set), so one test's configuration would otherwise leak into the next test
// and into later -count iterations.
func IsolateEnv(tb testing.TB) {
	tb.Helper()
	snapshot := environMap(os.Environ())
	tb.Cleanup(func() {
		if err := restoreEnv(snapshot); err != nil {
			tb.Errorf("failed to restore environment: %v", err)
		}
	})
}

// ClearConfigEnv unsets every go-pre-commit configuration variable inherited
// by the test process (ENABLE_GO_PRE_COMMIT and GO_PRE_COMMIT_*). Task runners
// such as magex export the repository's .github/env files into the process,
// and config.Load does not override variables that are already set, so those
// values would otherwise replace the settings in each test's fixture
// configuration. Call it from TestMain before m.Run; tests that need a value
// set it themselves.
func ClearConfigEnv() error {
	for key := range environMap(os.Environ()) {
		if IsConfigEnvVar(key) {
			if err := os.Unsetenv(key); err != nil {
				return err
			}
		}
	}
	return nil
}

// IsConfigEnvVar reports whether key is a go-pre-commit configuration variable
func IsConfigEnvVar(key string) bool {
	return key == "ENABLE_GO_PRE_COMMIT" || strings.HasPrefix(key, "GO_PRE_COMMIT_")
}

// restoreEnv makes the process environment match snapshot exactly
func restoreEnv(snapshot map[string]string) error {
	for key := range environMap(os.Environ()) {
		if _, ok := snapshot[key]; !ok {
			if err := os.Unsetenv(key); err != nil {
				return err
			}
		}
	}
	for key, value := range snapshot {
		if current, ok := os.LookupEnv(key); !ok || current != value {
			if err := os.Setenv(key, value); err != nil {
				return err
			}
		}
	}
	return nil
}

// environMap converts os.Environ output to a map. Entries without a name, such
// as the per-drive "=C:" variables on Windows, are skipped.
func environMap(environ []string) map[string]string {
	env := make(map[string]string, len(environ))
	for _, kv := range environ {
		key, value, _ := strings.Cut(kv, "=")
		if key != "" {
			env[key] = value
		}
	}
	return env
}
