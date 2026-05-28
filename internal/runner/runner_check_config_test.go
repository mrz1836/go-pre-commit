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
	cfg.CheckTimeouts.Fumpt = 45
	cfg.CheckTimeouts.Gitleaks = 60
	cfg.CheckTimeouts.Lint = 90
	cfg.CheckTimeouts.ModTidy = 50
	cfg.CheckTimeouts.Whitespace = 20
	cfg.CheckTimeouts.EOF = 15

	runner := New(cfg, "/tmp")

	tests := []struct {
		name         string
		checkName    string
		expectedTime time.Duration
		description  string
	}{
		{
			name:         "Fumpt timeout",
			checkName:    checkNameFumpt,
			expectedTime: 45 * time.Second,
			description:  "Should return configured fumpt timeout",
		},
		{
			name:         "Gitleaks timeout",
			checkName:    checkNameGitleaks,
			expectedTime: 60 * time.Second,
			description:  "Should return configured gitleaks timeout",
		},
		{
			name:         "Lint timeout",
			checkName:    checkNameLint,
			expectedTime: 90 * time.Second,
			description:  "Should return configured lint timeout",
		},
		{
			name:         "ModTidy timeout",
			checkName:    checkNameModTidy,
			expectedTime: 50 * time.Second,
			description:  "Should return configured mod-tidy timeout",
		},
		{
			name:         "Whitespace timeout",
			checkName:    checkNameWhitespace,
			expectedTime: 20 * time.Second,
			description:  "Should return configured whitespace timeout",
		},
		{
			name:         "EOF timeout",
			checkName:    checkNameEOF,
			expectedTime: 15 * time.Second,
			description:  "Should return configured eof timeout",
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
	cfg.CheckTimeouts.ModTidy = 0 // Zero timeout
	cfg.CheckTimeouts.Fumpt = 0
	cfg.CheckTimeouts.Lint = 0

	runner := New(cfg, "/tmp")

	tests := []struct {
		checkName    string
		expectedTime time.Duration
	}{
		{checkNameModTidy, 0 * time.Second},
		{checkNameFumpt, 0 * time.Second},
		{checkNameLint, 0 * time.Second},
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
	cfg.Checks.EOF = true
	cfg.Checks.Fumpt = true
	cfg.Checks.Gitleaks = false
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
			name:        "EOF enabled",
			checkName:   checkNameEOF,
			expected:    true,
			description: "Should return true when eof is enabled",
		},
		{
			name:        "Fumpt enabled",
			checkName:   checkNameFumpt,
			expected:    true,
			description: "Should return true when fumpt is enabled",
		},
		{
			name:        "Gitleaks disabled",
			checkName:   checkNameGitleaks,
			expected:    false,
			description: "Should return false when gitleaks is disabled",
		},
		{
			name:        "Lint enabled",
			checkName:   checkNameLint,
			expected:    true,
			description: "Should return true when lint is enabled",
		},
		{
			name:        "ModTidy disabled",
			checkName:   checkNameModTidy,
			expected:    false,
			description: "Should return false when mod-tidy is disabled",
		},
		{
			name:        "Whitespace enabled",
			checkName:   checkNameWhitespace,
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
	cfg.Checks.EOF = true
	cfg.Checks.Fumpt = true
	cfg.Checks.Gitleaks = true
	cfg.Checks.Lint = true
	cfg.Checks.ModTidy = true
	cfg.Checks.Whitespace = true

	runner := New(cfg, "/tmp")

	allChecks := []string{
		checkNameEOF,
		checkNameFumpt,
		checkNameGitleaks,
		checkNameLint,
		checkNameModTidy,
		checkNameWhitespace,
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
	cfg.Checks.EOF = false
	cfg.Checks.Fumpt = false
	cfg.Checks.Gitleaks = false
	cfg.Checks.Lint = false
	cfg.Checks.ModTidy = false
	cfg.Checks.Whitespace = false

	runner := New(cfg, "/tmp")

	allChecks := []string{
		checkNameEOF,
		checkNameFumpt,
		checkNameGitleaks,
		checkNameLint,
		checkNameModTidy,
		checkNameWhitespace,
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
	cfg.CheckTimeouts.ModTidy = 30
	cfg.CheckTimeouts.Fumpt = 45
	cfg.CheckTimeouts.Lint = 90

	runner := New(cfg, "/tmp")

	// Call multiple times and verify consistency
	for i := 0; i < 10; i++ {
		assert.Equal(t, 30*time.Second, runner.getCheckTimeout(checkNameModTidy))
		assert.Equal(t, 45*time.Second, runner.getCheckTimeout(checkNameFumpt))
		assert.Equal(t, 90*time.Second, runner.getCheckTimeout(checkNameLint))
	}
}

// TestIsCheckEnabledConsistency tests that multiple calls return consistent results
func TestIsCheckEnabledConsistency(t *testing.T) {
	cfg := &config.Config{
		Enabled: true,
	}
	cfg.Checks.ModTidy = true
	cfg.Checks.Fumpt = false
	cfg.Checks.Lint = true

	runner := New(cfg, "/tmp")

	// Call multiple times and verify consistency
	for i := 0; i < 10; i++ {
		assert.True(t, runner.isCheckEnabled(checkNameModTidy))
		assert.False(t, runner.isCheckEnabled(checkNameFumpt))
		assert.True(t, runner.isCheckEnabled(checkNameLint))
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
	assert.Equal(t, 0*time.Second, runner.getCheckTimeout(checkNameModTidy))
	assert.Equal(t, 0*time.Second, runner.getCheckTimeout("unknown"))
}

// TestCheckNameCaseSensitivity tests that check names are case-sensitive
func TestCheckNameCaseSensitivity(t *testing.T) {
	cfg := &config.Config{
		Enabled: true,
		Timeout: 120,
	}
	cfg.Checks.ModTidy = true
	cfg.Checks.Fumpt = true
	cfg.Checks.Lint = true
	cfg.CheckTimeouts.ModTidy = 30
	cfg.CheckTimeouts.Fumpt = 45
	cfg.CheckTimeouts.Lint = 60

	runner := New(cfg, "/tmp")

	// Test that check names are case-sensitive
	tests := []struct {
		name      string
		checkName string
	}{
		{"Uppercase MOD-TIDY", "MOD-TIDY"},
		{"Uppercase FUMPT", "FUMPT"},
		{"Mixed case Mod-Tidy", "Mod-Tidy"},
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
	assert.True(t, runner.isCheckEnabled(checkNameModTidy))
	assert.Equal(t, 50*time.Second, runner.getCheckTimeout(checkNameModTidy))

	// Test variations that should not match
	assert.False(t, runner.isCheckEnabled("mod_tidy"))
	assert.False(t, runner.isCheckEnabled("modtidy"))
	assert.False(t, runner.isCheckEnabled("mod tidy"))
}
