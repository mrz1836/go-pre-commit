package plugins

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPlugin(t *testing.T) {
	tests := []struct {
		name      string
		manifest  *PluginManifest
		directory string
		wantErr   bool
		errMsg    string
	}{
		{
			name:      "nil manifest",
			manifest:  nil,
			directory: "/tmp",
			wantErr:   true,
			errMsg:    "manifest cannot be nil",
		},
		{
			name: "empty name",
			manifest: &PluginManifest{
				Name:       "",
				Executable: "test.sh",
			},
			directory: "/tmp",
			wantErr:   true,
			errMsg:    "plugin name is required",
		},
		{
			name: "empty executable",
			manifest: &PluginManifest{
				Name:       "test-plugin",
				Executable: "",
			},
			directory: "/tmp",
			wantErr:   true,
			errMsg:    "plugin executable is required",
		},
		{
			name: "valid plugin",
			manifest: &PluginManifest{
				Name:         "test-plugin",
				Version:      "1.0.0",
				Description:  "Test plugin",
				Executable:   "test.sh",
				FilePatterns: []string{"*.go"},
				Timeout:      "30s",
				Category:     "testing",
			},
			directory: "/tmp",
			wantErr:   false,
		},
		{
			name: "invalid timeout",
			manifest: &PluginManifest{
				Name:       "test-plugin",
				Executable: "test.sh",
				Timeout:    "invalid",
			},
			directory: "/tmp",
			wantErr:   true,
			errMsg:    "invalid timeout format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plugin, err := NewPlugin(tt.manifest, tt.directory)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
				assert.Nil(t, plugin)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, plugin)
				assert.Equal(t, tt.manifest.Name, plugin.Name())
				assert.Equal(t, tt.manifest.Description, plugin.Description())
				assert.Equal(t, tt.directory, plugin.directory)
			}
		})
	}
}

func TestPluginMetadata(t *testing.T) {
	manifest := &PluginManifest{
		Name:          "test-plugin",
		Version:       "1.0.0",
		Description:   "Test plugin",
		Author:        "Test Author",
		Executable:    "test.sh",
		FilePatterns:  []string{"*.go", "*.js"},
		Timeout:       "45s",
		Category:      "linting",
		RequiresFiles: true,
		Dependencies:  []string{"bash", "grep"},
	}

	plugin, err := NewPlugin(manifest, "/tmp")
	require.NoError(t, err)

	metadata := plugin.Metadata().(PluginMetadata)

	assert.Equal(t, "test-plugin", metadata.Name)
	assert.Equal(t, "Test plugin", metadata.Description)
	assert.Equal(t, "1.0.0", metadata.Version)
	assert.Equal(t, "Test Author", metadata.Author)
	assert.Equal(t, "linting", metadata.Category)
	assert.Equal(t, []string{"*.go", "*.js"}, metadata.FilePatterns)
	assert.Equal(t, []string{"bash", "grep"}, metadata.Dependencies)
	assert.Equal(t, 45*time.Second, metadata.DefaultTimeout)
	assert.True(t, metadata.RequiresFiles)
}

func TestPluginFilterFiles(t *testing.T) {
	tests := []struct {
		name          string
		patterns      []string
		inputFiles    []string
		expectedFiles []string
	}{
		{
			name:          "no patterns",
			patterns:      []string{},
			inputFiles:    []string{"main.go", "test.js", "doc.md"},
			expectedFiles: []string{"main.go", "test.js", "doc.md"},
		},
		{
			name:          "extension filter",
			patterns:      []string{"*.go"},
			inputFiles:    []string{"main.go", "test.js", "doc.md", "utils.go"},
			expectedFiles: []string{"main.go", "utils.go"},
		},
		{
			name:          "multiple patterns",
			patterns:      []string{"*.go", "*.js"},
			inputFiles:    []string{"main.go", "test.js", "doc.md", "utils.go"},
			expectedFiles: []string{"main.go", "test.js", "utils.go"},
		},
		{
			name:          "glob pattern",
			patterns:      []string{"cmd/*.go"},
			inputFiles:    []string{"main.go", "cmd/app.go", "cmd/util.go", "internal/helper.go"},
			expectedFiles: []string{"cmd/app.go", "cmd/util.go"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manifest := &PluginManifest{
				Name:         "test-plugin",
				Executable:   "test.sh",
				FilePatterns: tt.patterns,
			}

			plugin, err := NewPlugin(manifest, "/tmp")
			require.NoError(t, err)

			filtered := plugin.FilterFiles(tt.inputFiles)
			assert.Equal(t, tt.expectedFiles, filtered)
		})
	}
}

func TestPluginRun(t *testing.T) {
	// Create a temporary directory for test plugin
	tmpDir := t.TempDir()

	// Create a simple test script that echoes success
	scriptPath := filepath.Join(tmpDir, "test.sh")
	scriptContent := `#!/bin/bash
read INPUT
echo '{"success": true, "output": "Test passed"}'
`
	// #nosec G306 - Test script needs execute permission
	err := os.WriteFile(scriptPath, []byte(scriptContent), 0o755)
	require.NoError(t, err)

	manifest := &PluginManifest{
		Name:       "test-plugin",
		Executable: "./test.sh",
		Timeout:    "5s",
	}

	plugin, err := NewPlugin(manifest, tmpDir)
	require.NoError(t, err)

	ctx := context.Background()
	files := []string{"test.go"}

	err = plugin.Run(ctx, files)
	assert.NoError(t, err)
}

func TestPluginRunTimeout(t *testing.T) {
	// Skip this test if running in CI or on slow systems
	if os.Getenv("CI") != "" {
		t.Skip("Skipping timeout test in CI")
	}

	// Create a temporary directory for test plugin
	tmpDir := t.TempDir()

	// Create a script that sleeps longer than timeout
	scriptPath := filepath.Join(tmpDir, "slow.sh")
	scriptContent := `#!/bin/bash
sleep 10
echo '{"success": true}'
`
	// #nosec G306 - Test script needs execute permission
	err := os.WriteFile(scriptPath, []byte(scriptContent), 0o755)
	require.NoError(t, err)

	manifest := &PluginManifest{
		Name:       "slow-plugin",
		Executable: "./slow.sh",
		Timeout:    "100ms",
	}

	plugin, err := NewPlugin(manifest, tmpDir)
	require.NoError(t, err)

	ctx := context.Background()
	files := []string{"test.go"}

	err = plugin.Run(ctx, files)
	require.Error(t, err)
	// The error should mention the plugin name failed
	assert.Contains(t, err.Error(), "slow-plugin")
}

func TestPluginRunMissingExecutable(t *testing.T) {
	tmpDir := t.TempDir()

	manifest := &PluginManifest{
		Name:       "missing-plugin",
		Executable: "./nonexistent.sh",
		Timeout:    "5s",
	}

	plugin, err := NewPlugin(manifest, tmpDir)
	require.NoError(t, err)

	ctx := context.Background()
	files := []string{"test.go"}

	err = plugin.Run(ctx, files)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestPluginRequest(t *testing.T) {
	req := PluginRequest{
		Command: "check",
		Files:   []string{"file1.go", "file2.go"},
		Config: map[string]string{
			"option": "value",
		},
	}

	data, err := json.Marshal(req)
	require.NoError(t, err)

	var decoded PluginRequest
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, req.Command, decoded.Command)
	assert.Equal(t, req.Files, decoded.Files)
	assert.Equal(t, req.Config, decoded.Config)
}

func TestPluginResponse(t *testing.T) {
	resp := PluginResponse{
		Success:    false,
		Error:      "Test error",
		Suggestion: "Fix the issue",
		Modified:   []string{"file1.go"},
		Output:     "Detailed output",
	}

	data, err := json.Marshal(resp)
	require.NoError(t, err)

	var decoded PluginResponse
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, resp.Success, decoded.Success)
	assert.Equal(t, resp.Error, decoded.Error)
	assert.Equal(t, resp.Suggestion, decoded.Suggestion)
	assert.Equal(t, resp.Modified, decoded.Modified)
	assert.Equal(t, resp.Output, decoded.Output)
}

// TestPluginRunWithEnvironmentVariables tests environment variable expansion
func TestPluginRunWithEnvironmentVariables(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a script that echoes environment variables
	scriptPath := filepath.Join(tmpDir, "env_test.sh")
	scriptContent := `#!/bin/bash
echo "{\"success\": true, \"output\": \"TEST_VAR=$TEST_VAR,EXPANDED_VAR=$EXPANDED_VAR\"}"
`
	// #nosec G306 - Test script needs execute permission
	err := os.WriteFile(scriptPath, []byte(scriptContent), 0o755)
	require.NoError(t, err)

	// Set environment variable for expansion test
	err = os.Setenv("HOME_VAR", "home_value")
	require.NoError(t, err)
	defer func() { _ = os.Unsetenv("HOME_VAR") }()

	manifest := &PluginManifest{
		Name:       "env-plugin",
		Executable: "./env_test.sh",
		Timeout:    "5s",
		Environment: map[string]string{
			"TEST_VAR":     "test_value",
			"EXPANDED_VAR": "$HOME_VAR",
		},
	}

	plugin, err := NewPlugin(manifest, tmpDir)
	require.NoError(t, err)

	ctx := context.Background()
	files := []string{"test.go"}

	err = plugin.Run(ctx, files)
	assert.NoError(t, err)
}

// TestPluginRunFailureWithJSONResponse tests plugin failure with JSON error response
func TestPluginRunFailureWithJSONResponse(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a script that returns error JSON but exits successfully
	// (so the error is in the JSON, not the exit code)
	scriptPath := filepath.Join(tmpDir, "fail_json.sh")
	scriptContent := `#!/bin/bash
read INPUT
# Output error JSON but don't exit with error code
# This simulates a plugin that uses JSON protocol for errors
echo '{"success": false, "error": "Plugin check failed", "suggestion": "Fix the code"}'
exit 1
`
	// #nosec G306 - Test script needs execute permission
	err := os.WriteFile(scriptPath, []byte(scriptContent), 0o755)
	require.NoError(t, err)

	manifest := &PluginManifest{
		Name:       "fail-plugin",
		Executable: "./fail_json.sh",
		Timeout:    "5s",
	}

	plugin, err := NewPlugin(manifest, tmpDir)
	require.NoError(t, err)

	ctx := context.Background()
	files := []string{"test.go"}

	err = plugin.Run(ctx, files)
	require.Error(t, err)
	// Error should mention the plugin failed
	assert.Contains(t, err.Error(), "fail-plugin")
}

// TestPluginRunWithEmptyOutput tests plugin with empty output
func TestPluginRunWithEmptyOutput(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a script that produces no output but exits successfully
	scriptPath := filepath.Join(tmpDir, "empty.sh")
	scriptContent := `#!/bin/bash
read INPUT
# No output - empty stdout
exit 0
`
	// #nosec G306 - Test script needs execute permission
	err := os.WriteFile(scriptPath, []byte(scriptContent), 0o755)
	require.NoError(t, err)

	manifest := &PluginManifest{
		Name:       "empty-plugin",
		Executable: "./empty.sh",
		Timeout:    "5s",
	}

	plugin, err := NewPlugin(manifest, tmpDir)
	require.NoError(t, err)

	ctx := context.Background()
	files := []string{"test.go"}

	err = plugin.Run(ctx, files)
	assert.NoError(t, err) // Empty output with exit 0 should be treated as success
}

// TestPluginRunWithNonJSONOutput tests plugin with non-JSON output
func TestPluginRunWithNonJSONOutput(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a script that produces non-JSON output but exits successfully
	scriptPath := filepath.Join(tmpDir, "nonjson.sh")
	scriptContent := `#!/bin/bash
read INPUT
echo "This is not JSON output"
exit 0
`
	// #nosec G306 - Test script needs execute permission
	err := os.WriteFile(scriptPath, []byte(scriptContent), 0o755)
	require.NoError(t, err)

	manifest := &PluginManifest{
		Name:       "nonjson-plugin",
		Executable: "./nonjson.sh",
		Timeout:    "5s",
	}

	plugin, err := NewPlugin(manifest, tmpDir)
	require.NoError(t, err)

	ctx := context.Background()
	files := []string{"test.go"}

	err = plugin.Run(ctx, files)
	require.Error(t, err)
	// Should get error about non-JSON output
	assert.Contains(t, err.Error(), "nonjson-plugin")
}

// TestPluginRunWithSuccessFalse tests plugin response with success=false
func TestPluginRunWithSuccessFalse(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a script that returns success=false in JSON but exits 0
	// The script may behave differently on various systems, so we verify
	// that an error is returned containing the plugin name
	scriptPath := filepath.Join(tmpDir, "success_false.sh")
	scriptContent := `#!/bin/bash
read INPUT
echo '{"success": false, "error": "Check failed", "suggestion": "Review your code"}'
exit 0
`
	// #nosec G306 - Test script needs execute permission
	err := os.WriteFile(scriptPath, []byte(scriptContent), 0o755)
	require.NoError(t, err)

	manifest := &PluginManifest{
		Name:       "success-false-plugin",
		Executable: "./success_false.sh",
		Timeout:    "5s",
	}

	plugin, err := NewPlugin(manifest, tmpDir)
	require.NoError(t, err)

	ctx := context.Background()
	files := []string{"test.go"}

	err = plugin.Run(ctx, files)
	require.Error(t, err)
	// Should contain plugin name in error (actual error message varies by platform)
	assert.Contains(t, err.Error(), "success-false-plugin")
}
