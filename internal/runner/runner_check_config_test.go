package runner

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mrz1836/go-pre-commit/internal/config"
)

// TestGetCheckTimeout tests the getCheckTimeout method for all check types
func TestGetCheckTimeout(t *testing.T) {
	// Create a config with specific timeout values for each check
	cfg := &config.Config{
		Enabled: true,
		Timeout: 120, // Default timeout
	}
	cfg.CheckTimeouts.Fmt = 30
	cfg.CheckTimeouts.Fumpt = 45
	cfg.CheckTimeouts.Gitleaks = 60
	cfg.CheckTimeouts.Goimports = 35
	cfg.CheckTimeouts.Lint = 90
	cfg.CheckTimeouts.ModTidy = 50
	cfg.CheckTimeouts.Whitespace = 20
	cfg.CheckTimeouts.EOF = 15
	cfg.CheckTimeouts.AIDetection = 100

	runner := New(cfg, "/tmp")

	tests := []struct {
		name         string
		checkName    string
		expectedTime time.Duration
		description  string
	}{
		{
			name:         "Fmt timeout",
			checkName:    "fmt",
			expectedTime: 30 * time.Second,
			description:  "Should return configured fmt timeout",
		},
		{
			name:         "Fumpt timeout",
			checkName:    "fumpt",
			expectedTime: 45 * time.Second,
			description:  "Should return configured fumpt timeout",
		},
		{
			name:         "Gitleaks timeout",
			checkName:    "gitleaks",
			expectedTime: 60 * time.Second,
			description:  "Should return configured gitleaks timeout",
		},
		{
			name:         "Goimports timeout",
			checkName:    "goimports",
			expectedTime: 35 * time.Second,
			description:  "Should return configured goimports timeout",
		},
		{
			name:         "Lint timeout",
			checkName:    "lint",
			expectedTime: 90 * time.Second,
			description:  "Should return configured lint timeout",
		},
		{
			name:         "ModTidy timeout",
			checkName:    "mod-tidy",
			expectedTime: 50 * time.Second,
			description:  "Should return configured mod-tidy timeout",
		},
		{
			name:         "Whitespace timeout",
			checkName:    "whitespace",
			expectedTime: 20 * time.Second,
			description:  "Should return configured whitespace timeout",
		},
		{
			name:         "EOF timeout",
			checkName:    "eof",
			expectedTime: 15 * time.Second,
			description:  "Should return configured eof timeout",
		},
		{
			name:         "AI Detection timeout",
			checkName:    "ai_detection",
			expectedTime: 100 * time.Second,
			description:  "Should return configured ai_detection timeout",
		},
		{
			name:         "Unknown check defaults to global timeout",
			checkName:    "unknown-check",
			expectedTime: 120 * time.Second,
			description:  "Should return global timeout for unknown check",
		},
		{
			name:         "Empty check name defaults to global timeout",
			checkName:    "",
			expectedTime: 120 * time.Second,
			description:  "Should return global timeout for empty check name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := runner.getCheckTimeout(tt.checkName)
			assert.Equal(t, tt.expectedTime, result, tt.description)
		})
	}
}

// TestGetCheckTimeoutWithZeroValues tests behavior when timeouts are set to zero
func TestGetCheckTimeoutWithZeroValues(t *testing.T) {
	cfg := &config.Config{
		Enabled: true,
		Timeout: 60,
	}
	cfg.CheckTimeouts.Fmt = 0 // Zero timeout
	cfg.CheckTimeouts.Fumpt = 0
	cfg.CheckTimeouts.Lint = 0

	runner := New(cfg, "/tmp")

	tests := []struct {
		checkName    string
		expectedTime time.Duration
	}{
		{"fmt", 0 * time.Second},
		{"fumpt", 0 * time.Second},
		{"lint", 0 * time.Second},
		{"unknown", 60 * time.Second}, // Should still use global timeout
	}

	for _, tt := range tests {
		t.Run(tt.checkName, func(t *testing.T) {
			result := runner.getCheckTimeout(tt.checkName)
			assert.Equal(t, tt.expectedTime, result)
		})
	}
}

// TestIsCheckEnabled tests the isCheckEnabled method for all check types
func TestIsCheckEnabled(t *testing.T) {
	// Create a config with specific checks enabled/disabled
	cfg := &config.Config{
		Enabled: true,
	}
	cfg.Checks.AIDetection = true
	cfg.Checks.EOF = true
	cfg.Checks.Fmt = false
	cfg.Checks.Fumpt = true
	cfg.Checks.Gitleaks = false
	cfg.Checks.Goimports = true
	cfg.Checks.Lint = true
	cfg.Checks.ModTidy = false
	cfg.Checks.Whitespace = true

	runner := New(cfg, "/tmp")

	tests := []struct {
		name        string
		checkName   string
		expected    bool
		description string
	}{
		{
			name:        "AI Detection enabled",
			checkName:   "ai_detection",
			expected:    true,
			description: "Should return true when ai_detection is enabled",
		},
		{
			name:        "EOF enabled",
			checkName:   "eof",
			expected:    true,
			description: "Should return true when eof is enabled",
		},
		{
			name:        "Fmt disabled",
			checkName:   "fmt",
			expected:    false,
			description: "Should return false when fmt is disabled",
		},
		{
			name:        "Fumpt enabled",
			checkName:   "fumpt",
			expected:    true,
			description: "Should return true when fumpt is enabled",
		},
		{
			name:        "Gitleaks disabled",
			checkName:   "gitleaks",
			expected:    false,
			description: "Should return false when gitleaks is disabled",
		},
		{
			name:        "Goimports enabled",
			checkName:   "goimports",
			expected:    true,
			description: "Should return true when goimports is enabled",
		},
		{
			name:        "Lint enabled",
			checkName:   "lint",
			expected:    true,
			description: "Should return true when lint is enabled",
		},
		{
			name:        "ModTidy disabled",
			checkName:   "mod-tidy",
			expected:    false,
			description: "Should return false when mod-tidy is disabled",
		},
		{
			name:        "Whitespace enabled",
			checkName:   "whitespace",
			expected:    true,
			description: "Should return true when whitespace is enabled",
		},
		{
			name:        "Unknown check returns false",
			checkName:   "unknown-check",
			expected:    false,
			description: "Should return false for unknown check",
		},
		{
			name:        "Empty check name returns false",
			checkName:   "",
			expected:    false,
			description: "Should return false for empty check name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := runner.isCheckEnabled(tt.checkName)
			assert.Equal(t, tt.expected, result, tt.description)
		})
	}
}

// TestIsCheckEnabledAllEnabled tests when all checks are enabled
func TestIsCheckEnabledAllEnabled(t *testing.T) {
	cfg := &config.Config{
		Enabled: true,
	}
	cfg.Checks.AIDetection = true
	cfg.Checks.EOF = true
	cfg.Checks.Fmt = true
	cfg.Checks.Fumpt = true
	cfg.Checks.Gitleaks = true
	cfg.Checks.Goimports = true
	cfg.Checks.Lint = true
	cfg.Checks.ModTidy = true
	cfg.Checks.Whitespace = true

	runner := New(cfg, "/tmp")

	allChecks := []string{
		"ai_detection",
		"eof",
		"fmt",
		"fumpt",
		"gitleaks",
		"goimports",
		"lint",
		"mod-tidy",
		"whitespace",
	}

	for _, checkName := range allChecks {
		t.Run(checkName, func(t *testing.T) {
			assert.True(t, runner.isCheckEnabled(checkName), "Check %s should be enabled", checkName)
		})
	}
}

// TestIsCheckEnabledAllDisabled tests when all checks are disabled
func TestIsCheckEnabledAllDisabled(t *testing.T) {
	cfg := &config.Config{
		Enabled: true,
	}
	cfg.Checks.AIDetection = false
	cfg.Checks.EOF = false
	cfg.Checks.Fmt = false
	cfg.Checks.Fumpt = false
	cfg.Checks.Gitleaks = false
	cfg.Checks.Goimports = false
	cfg.Checks.Lint = false
	cfg.Checks.ModTidy = false
	cfg.Checks.Whitespace = false

	runner := New(cfg, "/tmp")

	allChecks := []string{
		"ai_detection",
		"eof",
		"fmt",
		"fumpt",
		"gitleaks",
		"goimports",
		"lint",
		"mod-tidy",
		"whitespace",
	}

	for _, checkName := range allChecks {
		t.Run(checkName, func(t *testing.T) {
			assert.False(t, runner.isCheckEnabled(checkName), "Check %s should be disabled", checkName)
		})
	}
}

// TestGetCheckTimeoutConsistency tests that multiple calls return consistent results
func TestGetCheckTimeoutConsistency(t *testing.T) {
	cfg := &config.Config{
		Enabled: true,
		Timeout: 120,
	}
	cfg.CheckTimeouts.Fmt = 30
	cfg.CheckTimeouts.Fumpt = 45
	cfg.CheckTimeouts.Lint = 90

	runner := New(cfg, "/tmp")

	// Call multiple times and verify consistency
	for i := 0; i < 10; i++ {
		assert.Equal(t, 30*time.Second, runner.getCheckTimeout("fmt"))
		assert.Equal(t, 45*time.Second, runner.getCheckTimeout("fumpt"))
		assert.Equal(t, 90*time.Second, runner.getCheckTimeout("lint"))
	}
}

// TestIsCheckEnabledConsistency tests that multiple calls return consistent results
func TestIsCheckEnabledConsistency(t *testing.T) {
	cfg := &config.Config{
		Enabled: true,
	}
	cfg.Checks.Fmt = true
	cfg.Checks.Fumpt = false
	cfg.Checks.Lint = true

	runner := New(cfg, "/tmp")

	// Call multiple times and verify consistency
	for i := 0; i < 10; i++ {
		assert.True(t, runner.isCheckEnabled("fmt"))
		assert.False(t, runner.isCheckEnabled("fumpt"))
		assert.True(t, runner.isCheckEnabled("lint"))
	}
}

// TestRunnerConfigNil tests that runner handles nil config gracefully
func TestRunnerWithNilConfig(t *testing.T) {
	// With nil config, New returns nil
	runner := New(nil, "/tmp")

	// The current implementation returns nil for nil config
	assert.Nil(t, runner)
}

// TestRunnerConfigDefaults tests default values in config
func TestRunnerConfigDefaults(t *testing.T) {
	cfg := &config.Config{
		Enabled: true,
		// Use default values for timeouts
		Timeout: 0,
	}

	runner := New(cfg, "/tmp")
	require.NotNil(t, runner)

	// When timeout is 0, getCheckTimeout should return 0
	assert.Equal(t, 0*time.Second, runner.getCheckTimeout("fmt"))
	assert.Equal(t, 0*time.Second, runner.getCheckTimeout("unknown"))
}

// TestCheckNameCaseSensitivity tests that check names are case-sensitive
func TestCheckNameCaseSensitivity(t *testing.T) {
	cfg := &config.Config{
		Enabled: true,
		Timeout: 120,
	}
	cfg.Checks.Fmt = true
	cfg.Checks.Fumpt = true
	cfg.Checks.Lint = true
	cfg.CheckTimeouts.Fmt = 30
	cfg.CheckTimeouts.Fumpt = 45
	cfg.CheckTimeouts.Lint = 60

	runner := New(cfg, "/tmp")

	// Test that check names are case-sensitive
	tests := []struct {
		name      string
		checkName string
	}{
		{"Uppercase FMT", "FMT"},
		{"Uppercase FUMPT", "FUMPT"},
		{"Mixed case Fmt", "Fmt"},
		{"Mixed case Fumpt", "Fumpt"},
		{"Uppercase LINT", "LINT"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Case-sensitive names should not match
			assert.False(t, runner.isCheckEnabled(tt.checkName),
				"Check name %s should not match (case-sensitive)", tt.checkName)

			// Should return default timeout for non-matching names
			assert.Equal(t, 120*time.Second, runner.getCheckTimeout(tt.checkName),
				"Unknown check %s should return default timeout", tt.checkName)
		})
	}
}

// TestCheckNameWithSpecialCharacters tests handling of special characters
func TestCheckNameWithSpecialCharacters(t *testing.T) {
	cfg := &config.Config{
		Enabled: true,
		Timeout: 60,
	}
	cfg.Checks.ModTidy = true
	cfg.CheckTimeouts.ModTidy = 50

	runner := New(cfg, "/tmp")

	// mod-tidy has a hyphen, which is valid
	assert.True(t, runner.isCheckEnabled("mod-tidy"))
	assert.Equal(t, 50*time.Second, runner.getCheckTimeout("mod-tidy"))

	// Test variations that should not match
	assert.False(t, runner.isCheckEnabled("mod_tidy"))
	assert.False(t, runner.isCheckEnabled("modtidy"))
	assert.False(t, runner.isCheckEnabled("mod tidy"))
}
