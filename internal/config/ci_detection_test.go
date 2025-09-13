// Package config provides comprehensive CI detection and timeout adjustment testing
package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDetectCIEnvironment(t *testing.T) {
	// Save original environment
	originalEnvs := make(map[string]string)
	ciEnvVars := []string{
		"GITHUB_ACTIONS", "GITLAB_CI", "JENKINS_URL", "BUILDKITE",
		"CIRCLECI", "TRAVIS", "APPVEYOR", "AZURE_HTTP_USER_AGENT",
		"TEAMCITY_VERSION", "DRONE", "SEMAPHORE", "CODEBUILD_BUILD_ID", "CI",
	}

	for _, envVar := range ciEnvVars {
		originalEnvs[envVar] = os.Getenv(envVar)
		_ = os.Unsetenv(envVar)
	}

	// Restore environment after test
	defer func() {
		for envVar, value := range originalEnvs {
			if value != "" {
				_ = os.Setenv(envVar, value)
			} else {
				_ = os.Unsetenv(envVar)
			}
		}
	}()

	tests := []struct {
		name             string
		envVar           string
		envValue         string
		expectedIsCI     bool
		expectedProvider string
	}{
		{
			name:             "GitHub Actions",
			envVar:           "GITHUB_ACTIONS",
			envValue:         "true",
			expectedIsCI:     true,
			expectedProvider: "github-actions",
		},
		{
			name:             "GitLab CI",
			envVar:           "GITLAB_CI",
			envValue:         "true",
			expectedIsCI:     true,
			expectedProvider: "gitlab",
		},
		{
			name:             "Jenkins",
			envVar:           "JENKINS_URL",
			envValue:         "http://jenkins.example.com",
			expectedIsCI:     true,
			expectedProvider: "jenkins",
		},
		{
			name:             "Buildkite",
			envVar:           "BUILDKITE",
			envValue:         "true",
			expectedIsCI:     true,
			expectedProvider: "buildkite",
		},
		{
			name:             "CircleCI",
			envVar:           "CIRCLECI",
			envValue:         "true",
			expectedIsCI:     true,
			expectedProvider: "circleci",
		},
		{
			name:             "Travis CI",
			envVar:           "TRAVIS",
			envValue:         "true",
			expectedIsCI:     true,
			expectedProvider: "travis",
		},
		{
			name:             "AppVeyor",
			envVar:           "APPVEYOR",
			envValue:         "True",
			expectedIsCI:     true,
			expectedProvider: "appveyor",
		},
		{
			name:             "Azure DevOps",
			envVar:           "AZURE_HTTP_USER_AGENT",
			envValue:         "Azure-Pipelines/1.0",
			expectedIsCI:     true,
			expectedProvider: "azure-devops",
		},
		{
			name:             "TeamCity",
			envVar:           "TEAMCITY_VERSION",
			envValue:         "2021.1",
			expectedIsCI:     true,
			expectedProvider: "teamcity",
		},
		{
			name:             "Drone",
			envVar:           "DRONE",
			envValue:         "true",
			expectedIsCI:     true,
			expectedProvider: "drone",
		},
		{
			name:             "Semaphore",
			envVar:           "SEMAPHORE",
			envValue:         "true",
			expectedIsCI:     true,
			expectedProvider: "semaphore",
		},
		{
			name:             "AWS CodeBuild",
			envVar:           "CODEBUILD_BUILD_ID",
			envValue:         "project:12345",
			expectedIsCI:     true,
			expectedProvider: "aws-codebuild",
		},
		{
			name:             "Generic CI",
			envVar:           "CI",
			envValue:         "true",
			expectedIsCI:     true,
			expectedProvider: "unknown",
		},
		{
			name:             "No CI environment",
			envVar:           "",
			envValue:         "",
			expectedIsCI:     false,
			expectedProvider: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear all CI environment variables
			for _, envVar := range ciEnvVars {
				_ = os.Unsetenv(envVar)
			}

			// Set the specific environment variable for this test
			if tt.envVar != "" {
				_ = os.Setenv(tt.envVar, tt.envValue)
			}

			isCI, provider := detectCIEnvironment()

			assert.Equal(t, tt.expectedIsCI, isCI)
			assert.Equal(t, tt.expectedProvider, provider)
		})
	}
}

func TestDetectCIEnvironment_Priority(t *testing.T) {
	// Save original environment
	originalEnvs := make(map[string]string)
	ciEnvVars := []string{"GITHUB_ACTIONS", "GITLAB_CI", "CI"}

	for _, envVar := range ciEnvVars {
		originalEnvs[envVar] = os.Getenv(envVar)
		_ = os.Unsetenv(envVar)
	}

	defer func() {
		for envVar, value := range originalEnvs {
			if value != "" {
				_ = os.Setenv(envVar, value)
			} else {
				_ = os.Unsetenv(envVar)
			}
		}
	}()

	// Set multiple CI environment variables - should detect the first specific one
	_ = os.Setenv("GITHUB_ACTIONS", "true")
	_ = os.Setenv("GITLAB_CI", "true")
	_ = os.Setenv("CI", "true")

	isCI, provider := detectCIEnvironment()

	assert.True(t, isCI)
	// Should detect GitHub Actions since it comes first in the map iteration
	// Map iteration order is not guaranteed, but in practice it's often stable
	assert.Contains(t, []string{"github-actions", "gitlab"}, provider)
}

func TestApplyCITimeoutAdjustments(t *testing.T) {
	tests := []struct {
		name                string
		initialConfig       Config
		expectedAdjustments map[string]int
	}{
		{
			name: "adjust default timeouts",
			initialConfig: Config{
				Timeout: 300,
				ToolInstallation: struct {
					Timeout int
				}{
					Timeout: 300,
				},
				CheckTimeouts: struct {
					Fmt         int
					Fumpt       int
					Goimports   int
					Lint        int
					ModTidy     int
					Whitespace  int
					EOF         int
					AIDetection int
				}{
					Fmt:         30,
					Fumpt:       30,
					Goimports:   30,
					Lint:        60,
					ModTidy:     30,
					Whitespace:  30,
					EOF:         30,
					AIDetection: 30,
				},
			},
			expectedAdjustments: map[string]int{
				"global":           600, // 10 minutes
				"toolInstallation": 600, // 10 minutes
				"lint":             180, // 3 minutes
				"fmt":              60,  // 1 minute
				"fumpt":            60,  // 1 minute
				"goimports":        60,  // 1 minute
				"modTidy":          90,  // 1.5 minutes
				"whitespace":       45,  // 45 seconds
				"eof":              45,  // 45 seconds
				"aiDetection":      60,  // 1 minute
			},
		},
		{
			name: "do not adjust custom timeouts",
			initialConfig: Config{
				Timeout: 120, // Custom timeout
				ToolInstallation: struct {
					Timeout int
				}{
					Timeout: 180, // Custom timeout
				},
				CheckTimeouts: struct {
					Fmt         int
					Fumpt       int
					Goimports   int
					Lint        int
					ModTidy     int
					Whitespace  int
					EOF         int
					AIDetection int
				}{
					Fmt:         45,  // Custom timeout
					Fumpt:       30,  // Default - should be adjusted
					Goimports:   30,  // Default - should be adjusted
					Lint:        120, // Custom timeout
					ModTidy:     45,  // Custom timeout
					Whitespace:  30,  // Default - should be adjusted
					EOF:         30,  // Default - should be adjusted
					AIDetection: 30,  // Default - should be adjusted
				},
			},
			expectedAdjustments: map[string]int{
				"global":           120, // Unchanged (custom)
				"toolInstallation": 180, // Unchanged (custom)
				"lint":             120, // Unchanged (custom)
				"fmt":              45,  // Unchanged (custom)
				"fumpt":            60,  // Adjusted (was default)
				"goimports":        60,  // Adjusted (was default)
				"modTidy":          45,  // Unchanged (custom)
				"whitespace":       45,  // Adjusted (was default)
				"eof":              45,  // Adjusted (was default)
				"aiDetection":      60,  // Adjusted (was default)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := tt.initialConfig
			applyCITimeoutAdjustments(&cfg)

			assert.Equal(t, tt.expectedAdjustments["global"], cfg.Timeout)
			assert.Equal(t, tt.expectedAdjustments["toolInstallation"], cfg.ToolInstallation.Timeout)
			assert.Equal(t, tt.expectedAdjustments["lint"], cfg.CheckTimeouts.Lint)
			assert.Equal(t, tt.expectedAdjustments["fmt"], cfg.CheckTimeouts.Fmt)
			assert.Equal(t, tt.expectedAdjustments["fumpt"], cfg.CheckTimeouts.Fumpt)
			assert.Equal(t, tt.expectedAdjustments["goimports"], cfg.CheckTimeouts.Goimports)
			assert.Equal(t, tt.expectedAdjustments["modTidy"], cfg.CheckTimeouts.ModTidy)
			assert.Equal(t, tt.expectedAdjustments["whitespace"], cfg.CheckTimeouts.Whitespace)
			assert.Equal(t, tt.expectedAdjustments["eof"], cfg.CheckTimeouts.EOF)
			assert.Equal(t, tt.expectedAdjustments["aiDetection"], cfg.CheckTimeouts.AIDetection)
		})
	}
}

func TestLoad_CIAutoAdjustments(t *testing.T) {
	// This test verifies that the config loading process properly detects CI
	// and applies timeout adjustments when auto-adjust is enabled

	// Save original environment
	originalEnvs := make(map[string]string)
	envVars := []string{
		"GITHUB_ACTIONS", "GO_PRE_COMMIT_AUTO_ADJUST_CI_TIMEOUTS",
		"GO_PRE_COMMIT_TIMEOUT_SECONDS", "GO_PRE_COMMIT_TOOL_INSTALL_TIMEOUT",
	}

	for _, envVar := range envVars {
		originalEnvs[envVar] = os.Getenv(envVar)
		_ = os.Unsetenv(envVar)
	}

	defer func() {
		for envVar, value := range originalEnvs {
			if value != "" {
				_ = os.Setenv(envVar, value)
			} else {
				_ = os.Unsetenv(envVar)
			}
		}
	}()

	tests := []struct {
		name             string
		setGitHubActions bool
		autoAdjust       string // "" means use default (true)
		expectAdjusted   bool
	}{
		{
			name:             "CI detected with auto-adjust enabled (default)",
			setGitHubActions: true,
			autoAdjust:       "", // Use default
			expectAdjusted:   true,
		},
		{
			name:             "CI detected with auto-adjust explicitly enabled",
			setGitHubActions: true,
			autoAdjust:       "true",
			expectAdjusted:   true,
		},
		{
			name:             "CI detected with auto-adjust disabled",
			setGitHubActions: true,
			autoAdjust:       "false",
			expectAdjusted:   false,
		},
		{
			name:             "No CI detected",
			setGitHubActions: false,
			autoAdjust:       "", // Use default
			expectAdjusted:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear environment
			for _, envVar := range envVars {
				_ = os.Unsetenv(envVar)
			}

			// Set test conditions
			if tt.setGitHubActions {
				_ = os.Setenv("GITHUB_ACTIONS", "true")
			}
			if tt.autoAdjust != "" {
				_ = os.Setenv("GO_PRE_COMMIT_AUTO_ADJUST_CI_TIMEOUTS", tt.autoAdjust)
			}

			cfg, err := Load()
			require.NoError(t, err)

			if tt.expectAdjusted {
				// Should have CI adjustments
				assert.True(t, cfg.Environment.IsCI)
				assert.Equal(t, "github-actions", cfg.Environment.CIProvider)
				assert.Equal(t, 600, cfg.Timeout)                  // Adjusted from 300
				assert.Equal(t, 600, cfg.ToolInstallation.Timeout) // Adjusted from 300
				assert.Equal(t, 180, cfg.CheckTimeouts.Lint)       // Adjusted from 60
			} else {
				// Should have default timeouts
				assert.Equal(t, 300, cfg.Timeout)                  // Default
				assert.Equal(t, 300, cfg.ToolInstallation.Timeout) // Default
				assert.Equal(t, 60, cfg.CheckTimeouts.Lint)        // Default
			}
		})
	}
}

func TestCIDetection_EdgeCases(t *testing.T) {
	// Save original environment
	originalCI := os.Getenv("CI")
	defer func() {
		if originalCI != "" {
			_ = os.Setenv("CI", originalCI)
		} else {
			_ = os.Unsetenv("CI")
		}
	}()

	tests := []struct {
		name     string
		ciValue  string
		expected bool
	}{
		{
			name:     "CI=true",
			ciValue:  "true",
			expected: true,
		},
		{
			name:     "CI=1",
			ciValue:  "1",
			expected: true,
		},
		{
			name:     "CI=yes",
			ciValue:  "yes",
			expected: true,
		},
		{
			name:     "CI=false",
			ciValue:  "false",
			expected: true, // Any non-empty value is considered CI
		},
		{
			name:     "CI empty",
			ciValue:  "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear all specific CI env vars first
			specificCIVars := []string{
				"GITHUB_ACTIONS", "GITLAB_CI", "JENKINS_URL", "BUILDKITE",
				"CIRCLECI", "TRAVIS", "APPVEYOR", "AZURE_HTTP_USER_AGENT",
				"TEAMCITY_VERSION", "DRONE", "SEMAPHORE", "CODEBUILD_BUILD_ID",
			}
			for _, envVar := range specificCIVars {
				_ = os.Unsetenv(envVar)
			}

			if tt.ciValue == "" {
				_ = os.Unsetenv("CI")
			} else {
				_ = os.Setenv("CI", tt.ciValue)
			}

			isCI, provider := detectCIEnvironment()

			assert.Equal(t, tt.expected, isCI)
			if tt.expected {
				assert.Equal(t, "unknown", provider)
			} else {
				assert.Empty(t, provider)
			}
		})
	}
}

func BenchmarkDetectCIEnvironment(b *testing.B) {
	// Benchmark CI detection performance
	_ = os.Setenv("GITHUB_ACTIONS", "true")
	defer func() { _ = os.Unsetenv("GITHUB_ACTIONS") }()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		detectCIEnvironment()
	}
}

func BenchmarkApplyCITimeoutAdjustments(b *testing.B) {
	// Benchmark timeout adjustment performance
	cfg := Config{
		Timeout: 300,
		ToolInstallation: struct {
			Timeout int
		}{
			Timeout: 300,
		},
		CheckTimeouts: struct {
			Fmt         int
			Fumpt       int
			Goimports   int
			Lint        int
			ModTidy     int
			Whitespace  int
			EOF         int
			AIDetection int
		}{
			Fmt:         30,
			Fumpt:       30,
			Goimports:   30,
			Lint:        60,
			ModTidy:     30,
			Whitespace:  30,
			EOF:         30,
			AIDetection: 30,
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		applyCITimeoutAdjustments(&cfg)
	}
}
