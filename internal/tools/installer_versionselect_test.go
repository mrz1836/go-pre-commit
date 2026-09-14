package tools

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// errStubGoNotFound simulates `go` being unavailable in stubbed version detection.
var errStubGoNotFound = errors.New("go not found")

// stubGoVersion replaces the Go-version detector for the duration of a test,
// restoring the real one on cleanup. Passing an empty string with wantErr=true
// simulates `go` not being available.
func stubGoVersion(t *testing.T, version string, wantErr bool) {
	t.Helper()
	orig := runGoVersionCommand
	t.Cleanup(func() { runGoVersionCommand = orig })
	runGoVersionCommand = func() (string, error) {
		if wantErr {
			return "", errStubGoNotFound
		}
		return version, nil
	}
}

func TestParseGoMajorMinor(t *testing.T) {
	tests := []struct {
		name         string
		in           string
		major, minor int
		ok           bool
	}{
		{"go version output", "go1.25.3", 1, 25, true},
		{"go version with newline", "go1.26.0\n", 1, 26, true},
		{"minor only", "1.26", 1, 26, true},
		{"dot-x suffix", "1.26.x", 1, 26, true},
		{"future major", "2.0.0", 2, 0, true},
		{"empty", "", 0, 0, false},
		{"garbage", "abc", 0, 0, false},
		{"leading junk rejected", "v1.25", 0, 0, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			major, minor, ok := parseGoMajorMinor(tc.in)
			assert.Equal(t, tc.ok, ok)
			if tc.ok {
				assert.Equal(t, tc.major, major)
				assert.Equal(t, tc.minor, minor)
			}
		})
	}
}

func TestSelectToolVersion(t *testing.T) {
	const (
		baseline = "v0.11.0"
		latest   = "v0.12.0"
	)

	t.Run("latest unset falls back to baseline (dual-pin disabled)", func(t *testing.T) {
		stubGoVersion(t, "go1.30.0", false) // even on new Go, no latest => baseline
		assert.Equal(t, baseline, selectToolVersion(baseline, "", "1.26"))
	})

	t.Run("go at min-go uses latest", func(t *testing.T) {
		stubGoVersion(t, "go1.26.0", false)
		assert.Equal(t, latest, selectToolVersion(baseline, latest, "1.26"))
	})

	t.Run("go above min-go uses latest", func(t *testing.T) {
		stubGoVersion(t, "go1.27.1", false)
		assert.Equal(t, latest, selectToolVersion(baseline, latest, "1.26"))
	})

	t.Run("go below min-go uses baseline", func(t *testing.T) {
		stubGoVersion(t, "go1.25.4", false)
		assert.Equal(t, baseline, selectToolVersion(baseline, latest, "1.26"))
	})

	t.Run("undetectable go is conservative (baseline)", func(t *testing.T) {
		stubGoVersion(t, "", true)
		assert.Equal(t, baseline, selectToolVersion(baseline, latest, "1.26"))
	})

	t.Run("unparseable min-go defaults to 1.26", func(t *testing.T) {
		stubGoVersion(t, "go1.25.0", false) // 1.25 < default 1.26 => baseline
		assert.Equal(t, baseline, selectToolVersion(baseline, latest, "not-a-version"))
	})
}

func TestNewToolchainMismatchError(t *testing.T) {
	tool := &Tool{Name: "gofumpt", ImportPath: "mvdan.cc/gofumpt", Version: "v0.12.0", Binary: "gofumpt"}

	t.Run("detects go-version requirement", func(t *testing.T) {
		stubGoVersion(t, "go1.25.1", false)
		t.Setenv("GOTOOLCHAIN", "local")

		out := []byte("go: mvdan.cc/gofumpt@v0.12.0 requires go >= 1.26.0 (running go 1.25.1; GOTOOLCHAIN=local)")
		err := newToolchainMismatchError(tool, out)

		require.Error(t, err)
		require.ErrorIs(t, err, ErrToolNeedsNewerGo)
		msg := err.Error()
		assert.Contains(t, msg, "requires Go >= 1.26.0")
		assert.Contains(t, msg, "Go 1.25")
		assert.Contains(t, msg, "GOTOOLCHAIN=local")
		assert.Contains(t, msg, "GO_PRE_COMMIT_FUMPT_VERSION")
		// The raw output is preserved so the true cause is never hidden.
		assert.Contains(t, msg, "Output:")
	})

	t.Run("nil for unrelated failures", func(t *testing.T) {
		assert.NoError(t, newToolchainMismatchError(tool, []byte("exit status 1: some other error")))
	})
}

func TestToolEnvKey(t *testing.T) {
	assert.Equal(t, "FUMPT", toolEnvKey("gofumpt"))
	assert.Equal(t, "GOLANGCI_LINT", toolEnvKey(toolGolangciLint))
	assert.Equal(t, "GOIMPORTS", toolEnvKey("goimports"))
}

func TestLoadVersionsFromEnv_DualPin(t *testing.T) {
	// Reset tool registry so this standalone test does not depend on suite setup.
	toolsMu.Lock()
	tools["gofumpt"] = &Tool{Name: "gofumpt", ImportPath: "mvdan.cc/gofumpt", Version: "", Binary: "gofumpt"}
	toolsMu.Unlock()

	t.Setenv("GO_PRE_COMMIT_FUMPT_VERSION", "v0.11.0")
	t.Setenv("GO_PRE_COMMIT_FUMPT_VERSION_LATEST", "v0.12.0")
	t.Setenv("GO_PRE_COMMIT_FUMPT_VERSION_LATEST_MIN_GO", "1.26")

	t.Run("newer Go selects latest pin", func(t *testing.T) {
		stubGoVersion(t, "go1.26.2", false)
		LoadVersionsFromEnv()
		toolsMu.RLock()
		defer toolsMu.RUnlock()
		assert.Equal(t, "v0.12.0", tools["gofumpt"].Version)
	})

	t.Run("older Go selects baseline pin", func(t *testing.T) {
		stubGoVersion(t, "go1.25.5", false)
		LoadVersionsFromEnv()
		toolsMu.RLock()
		defer toolsMu.RUnlock()
		assert.Equal(t, "v0.11.0", tools["gofumpt"].Version)
	})
}
