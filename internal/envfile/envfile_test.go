package envfile

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"
)

// EnvFileTestSuite is a test suite for the envfile package
type EnvFileTestSuite struct {
	suite.Suite

	tempDir string
}

// SetupTest creates a temporary directory for each test
func (s *EnvFileTestSuite) SetupTest() {
	tempDir, err := os.MkdirTemp("", "envfile-test-*")
	s.Require().NoError(err)
	s.tempDir = tempDir

	// Clear relevant environment variables before each test
	s.clearEnv()
}

// TearDownTest cleans up the temporary directory
func (s *EnvFileTestSuite) TearDownTest() {
	if s.tempDir != "" {
		_ = os.RemoveAll(s.tempDir)
	}
	s.clearEnv()
}

// clearEnv clears test environment variables
func (s *EnvFileTestSuite) clearEnv() {
	testEnvVars := []string{
		"TEST_KEY", "TEST_VALUE", "SIMPLE_KEY", "KEY_WITH_SPACES",
		"KEY_WITH_COMMENT", "QUOTED_VALUE", "SINGLE_QUOTED",
		"ENABLE_GO_PRE_COMMIT", "GO_PRE_COMMIT_LOG_LEVEL",
		"EXISTING_VAR", "NEW_VAR", "OVERRIDE_ME",
		"CORE_VAR", "TOOLS_VAR", "PROJECT_VAR", "SHARED_VAR", "LOCAL_VAR", "ORDER_VAR",
	}
	for _, key := range testEnvVars {
		_ = os.Unsetenv(key)
	}
}

// TestLoad tests the Load function
func (s *EnvFileTestSuite) TestLoad() {
	// Create a test .env file
	envFile := filepath.Join(s.tempDir, ".env")
	content := `# Comment line
SIMPLE_KEY=simple_value
KEY_WITH_SPACES=value with spaces
KEY_WITH_COMMENT=value # inline comment

# Another comment
QUOTED_VALUE="quoted value with spaces"
SINGLE_QUOTED='single quoted value'
`
	err := os.WriteFile(envFile, []byte(content), 0o600)
	s.Require().NoError(err)

	// Load the file
	err = Load(envFile)
	s.Require().NoError(err)

	// Verify environment variables were set
	s.Equal("simple_value", os.Getenv("SIMPLE_KEY"))
	s.Equal("value with spaces", os.Getenv("KEY_WITH_SPACES"))
	s.Equal("value", os.Getenv("KEY_WITH_COMMENT"))
	s.Equal("quoted value with spaces", os.Getenv("QUOTED_VALUE"))
	s.Equal("single quoted value", os.Getenv("SINGLE_QUOTED"))
}

// TestLoad_DoesNotOverrideExisting tests that Load does not override existing env vars
func (s *EnvFileTestSuite) TestLoad_DoesNotOverrideExisting() {
	// Set an existing environment variable
	err := os.Setenv("EXISTING_VAR", "original_value")
	s.Require().NoError(err)

	// Create a test .env file that tries to override it
	envFile := filepath.Join(s.tempDir, ".env")
	content := `EXISTING_VAR=new_value
NEW_VAR=new_value
`
	err = os.WriteFile(envFile, []byte(content), 0o600)
	s.Require().NoError(err)

	// Load the file
	err = Load(envFile)
	s.Require().NoError(err)

	// Verify existing var was NOT overridden
	s.Equal("original_value", os.Getenv("EXISTING_VAR"))
	// But new var was set
	s.Equal("new_value", os.Getenv("NEW_VAR"))
}

// TestOverload tests the Overload function
func (s *EnvFileTestSuite) TestOverload() {
	// Set an existing environment variable
	err := os.Setenv("OVERRIDE_ME", "original_value")
	s.Require().NoError(err)

	// Create a test .env file
	envFile := filepath.Join(s.tempDir, ".env")
	content := `OVERRIDE_ME=overridden_value
NEW_VAR=new_value
`
	err = os.WriteFile(envFile, []byte(content), 0o600)
	s.Require().NoError(err)

	// Overload the file
	err = Overload(envFile)
	s.Require().NoError(err)

	// Verify existing var WAS overridden
	s.Equal("overridden_value", os.Getenv("OVERRIDE_ME"))
	s.Equal("new_value", os.Getenv("NEW_VAR"))
}

// TestLoad_FileNotFound tests error handling for missing files
func (s *EnvFileTestSuite) TestLoad_FileNotFound() {
	err := Load(filepath.Join(s.tempDir, "nonexistent.env"))
	s.Require().Error(err)
	s.Contains(err.Error(), "failed to read file")
}

// TestParse_EmptyLines tests parsing with empty lines
func (s *EnvFileTestSuite) TestParse_EmptyLines() {
	content := `
KEY1=value1

KEY2=value2


KEY3=value3
`
	result := parse(content)
	s.Len(result, 3)
	s.Equal("value1", result["KEY1"])
	s.Equal("value2", result["KEY2"])
	s.Equal("value3", result["KEY3"])
}

// TestParse_Comments tests parsing with comments
func (s *EnvFileTestSuite) TestParse_Comments() {
	content := `# Full line comment
KEY1=value1
# Another comment
KEY2=value2 # Inline comment
KEY3=value3#no space before comment
`
	result := parse(content)
	s.Len(result, 3)
	s.Equal("value1", result["KEY1"])
	s.Equal("value2", result["KEY2"])
	s.Equal("value3#no space before comment", result["KEY3"]) // # without space is kept
}

// TestParse_QuotedValues tests parsing with quoted values
func (s *EnvFileTestSuite) TestParse_QuotedValues() {
	content := `DOUBLE_QUOTED="value with spaces"
SINGLE_QUOTED='value with spaces'
EMPTY_DOUBLE=""
EMPTY_SINGLE=''
WITH_HASH_IN_QUOTES="value#with#hashes"
`
	result := parse(content)
	s.Len(result, 5)
	s.Equal("value with spaces", result["DOUBLE_QUOTED"])
	s.Equal("value with spaces", result["SINGLE_QUOTED"])
	s.Empty(result["EMPTY_DOUBLE"])
	s.Empty(result["EMPTY_SINGLE"])
	s.Equal("value#with#hashes", result["WITH_HASH_IN_QUOTES"])
}

// TestParse_MalformedLines tests tolerant parsing of malformed lines
func (s *EnvFileTestSuite) TestParse_MalformedLines() {
	content := `VALID_KEY=valid_value
=no_key_only_value
KEY_WITHOUT_VALUE

ANOTHER_VALID=another_value
INVALID_LINE_WITHOUT_EQUALS
=
KEY_WITH_WHITESPACE  =  value with surrounding spaces
`
	result := parse(content)
	// Should only parse the valid lines
	s.Len(result, 3)
	s.Equal("valid_value", result["VALID_KEY"])
	s.Equal("another_value", result["ANOTHER_VALID"])
	s.Equal("value with surrounding spaces", result["KEY_WITH_WHITESPACE"])
}

// TestParse_InlineComments tests inline comment stripping
func (s *EnvFileTestSuite) TestParse_InlineComments() {
	content := `KEY1=value # this is a comment
KEY2=value# no space
KEY3=value#with#multiple#hashes
KEY4=url_with_fragment#anchor
KEY5=  value with leading spaces  # comment
`
	result := parse(content)
	s.Len(result, 5)
	s.Equal("value", result["KEY1"])
	s.Equal("value# no space", result["KEY2"])            // # without space is kept
	s.Equal("value#with#multiple#hashes", result["KEY3"]) // # without space is kept
	s.Equal("url_with_fragment#anchor", result["KEY4"])   // # without space is kept (URL fragments, etc.)
	s.Equal("value with leading spaces", result["KEY5"])
}

// TestParse_SpecialCharacters tests parsing with special characters
func (s *EnvFileTestSuite) TestParse_SpecialCharacters() {
	content := `KEY_WITH_EQUALS=value=with=equals
KEY_WITH_COLON=value:with:colons
KEY_WITH_SLASH=value/with/slashes
KEY_WITH_BACKSLASH=value\with\backslashes
EMPTY_VALUE=
`
	result := parse(content)
	s.Len(result, 5)
	s.Equal("value=with=equals", result["KEY_WITH_EQUALS"])
	s.Equal("value:with:colons", result["KEY_WITH_COLON"])
	s.Equal("value/with/slashes", result["KEY_WITH_SLASH"])
	s.Equal("value\\with\\backslashes", result["KEY_WITH_BACKSLASH"])
	s.Empty(result["EMPTY_VALUE"])
}

// TestParse_RealWorldExample tests parsing a real-world .env file
func (s *EnvFileTestSuite) TestParse_RealWorldExample() {
	content := `# GoFortress Configuration
ENABLE_GO_PRE_COMMIT=true

# Core settings
GO_PRE_COMMIT_LOG_LEVEL=info  # Log level: debug, info, warn, error
GO_PRE_COMMIT_TIMEOUT_SECONDS=300

# Tool versions
GO_PRE_COMMIT_FUMPT_VERSION=v0.9.1     # https://github.com/mvdan/gofumpt
GO_PRE_COMMIT_GOLANGCI_LINT_VERSION=v2.5.0

# Empty line and comment

GO_PRE_COMMIT_COLOR_OUTPUT=false
`
	result := parse(content)
	s.Len(result, 6) // 6 environment variables in total
	s.Equal("true", result["ENABLE_GO_PRE_COMMIT"])
	s.Equal("info", result["GO_PRE_COMMIT_LOG_LEVEL"])
	s.Equal("300", result["GO_PRE_COMMIT_TIMEOUT_SECONDS"])
	s.Equal("v0.9.1", result["GO_PRE_COMMIT_FUMPT_VERSION"])
	s.Equal("v2.5.0", result["GO_PRE_COMMIT_GOLANGCI_LINT_VERSION"])
	s.Equal("false", result["GO_PRE_COMMIT_COLOR_OUTPUT"])
}

// TestProcessValue tests the processValue function
func (s *EnvFileTestSuite) TestProcessValue() {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{"simple value", "simple", "simple"},
		{"value with spaces", "  value  ", "value"},
		{"value with inline comment", "value # comment", "value"},
		{"double quoted", "\"quoted value\"", "quoted value"},
		{"single quoted", "'quoted value'", "quoted value"},
		{"empty quotes", "\"\"", ""},
		{"value with # in middle", "value#anchor", "value#anchor"}, // No space before # means it's kept
		{"quoted with # inside", "\"value#with#hash\"", "value#with#hash"},
		{"leading spaces", "  value", "value"},
		{"trailing spaces", "value  ", "value"},
		{"only whitespace before comment", "value  # comment", "value"},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			result := processValue(tc.input)
			s.Equal(tc.expected, result, "Test case: %s", tc.name)
		})
	}
}

// TestIntegration_LoadAndOverload tests the full workflow
func (s *EnvFileTestSuite) TestIntegration_LoadAndOverload() {
	// Create base .env file
	baseFile := filepath.Join(s.tempDir, ".env.base")
	baseContent := `# Base configuration
BASE_VAR=base_value
SHARED_VAR=base_shared
`
	err := os.WriteFile(baseFile, []byte(baseContent), 0o600)
	s.Require().NoError(err)

	// Create custom .env file
	customFile := filepath.Join(s.tempDir, ".env.custom")
	customContent := `# Custom configuration
CUSTOM_VAR=custom_value
SHARED_VAR=custom_shared
`
	err = os.WriteFile(customFile, []byte(customContent), 0o600)
	s.Require().NoError(err)

	// Load base file
	err = Load(baseFile)
	s.Require().NoError(err)

	// Verify base vars are set
	s.Equal("base_value", os.Getenv("BASE_VAR"))
	s.Equal("base_shared", os.Getenv("SHARED_VAR"))

	// Overload with custom file
	err = Overload(customFile)
	s.Require().NoError(err)

	// Verify custom vars are set and shared var is overridden
	s.Equal("base_value", os.Getenv("BASE_VAR"))      // Unchanged
	s.Equal("custom_shared", os.Getenv("SHARED_VAR")) // Overridden
	s.Equal("custom_value", os.Getenv("CUSTOM_VAR"))  // New
}

// TestLoadDir tests that LoadDir loads multiple env files in order with last-wins semantics
func (s *EnvFileTestSuite) TestLoadDir() {
	envDir := filepath.Join(s.tempDir, "env")
	s.Require().NoError(os.MkdirAll(envDir, 0o750))

	s.Require().NoError(os.WriteFile(filepath.Join(envDir, "00-core.env"), []byte("CORE_VAR=core\nSHARED_VAR=core\n"), 0o600))
	s.Require().NoError(os.WriteFile(filepath.Join(envDir, "10-tools.env"), []byte("TOOLS_VAR=tools\nSHARED_VAR=tools\n"), 0o600))
	s.Require().NoError(os.WriteFile(filepath.Join(envDir, "90-project.env"), []byte("PROJECT_VAR=project\nSHARED_VAR=project\n"), 0o600))

	err := LoadDir(envDir, false)
	s.Require().NoError(err)

	s.Equal("core", os.Getenv("CORE_VAR"))
	s.Equal("tools", os.Getenv("TOOLS_VAR"))
	s.Equal("project", os.Getenv("PROJECT_VAR"))
	s.Equal("project", os.Getenv("SHARED_VAR")) // last-wins
}

// TestLoadDirSkipsLocalInCI tests that 99-local.env is skipped when skipLocal=true
func (s *EnvFileTestSuite) TestLoadDirSkipsLocalInCI() {
	envDir := filepath.Join(s.tempDir, "env")
	s.Require().NoError(os.MkdirAll(envDir, 0o750))

	s.Require().NoError(os.WriteFile(filepath.Join(envDir, "00-core.env"), []byte("CORE_VAR=core\n"), 0o600))
	s.Require().NoError(os.WriteFile(filepath.Join(envDir, "99-local.env"), []byte("LOCAL_VAR=local\n"), 0o600))

	err := LoadDir(envDir, true)
	s.Require().NoError(err)

	s.Equal("core", os.Getenv("CORE_VAR"))
	s.Empty(os.Getenv("LOCAL_VAR")) // skipped
}

// TestLoadDirIncludesLocalWhenNotCI tests that 99-local.env is loaded when skipLocal=false
func (s *EnvFileTestSuite) TestLoadDirIncludesLocalWhenNotCI() {
	envDir := filepath.Join(s.tempDir, "env")
	s.Require().NoError(os.MkdirAll(envDir, 0o750))

	s.Require().NoError(os.WriteFile(filepath.Join(envDir, "00-core.env"), []byte("CORE_VAR=core\n"), 0o600))
	s.Require().NoError(os.WriteFile(filepath.Join(envDir, "99-local.env"), []byte("LOCAL_VAR=local\n"), 0o600))

	err := LoadDir(envDir, false)
	s.Require().NoError(err)

	s.Equal("core", os.Getenv("CORE_VAR"))
	s.Equal("local", os.Getenv("LOCAL_VAR")) // loaded
}

// TestLoadDirEmptyDirectory tests error when directory has no .env files
func (s *EnvFileTestSuite) TestLoadDirEmptyDirectory() {
	envDir := filepath.Join(s.tempDir, "empty-env")
	s.Require().NoError(os.MkdirAll(envDir, 0o750))

	err := LoadDir(envDir, false)
	s.Require().Error(err)
	s.Contains(err.Error(), "no .env files found")
}

// TestLoadDirNonexistentDirectory tests error when directory doesn't exist
func (s *EnvFileTestSuite) TestLoadDirNonexistentDirectory() {
	err := LoadDir(filepath.Join(s.tempDir, "nonexistent"), false)
	s.Require().Error(err)
	s.Contains(err.Error(), "env directory not found")
}

// TestLoadDirSortOrder tests that files load in correct lexicographic order regardless of creation order
func (s *EnvFileTestSuite) TestLoadDirSortOrder() {
	envDir := filepath.Join(s.tempDir, "env")
	s.Require().NoError(os.MkdirAll(envDir, 0o750))

	// Create files in reverse order
	s.Require().NoError(os.WriteFile(filepath.Join(envDir, "90-project.env"), []byte("ORDER_VAR=90\n"), 0o600))
	s.Require().NoError(os.WriteFile(filepath.Join(envDir, "10-tools.env"), []byte("ORDER_VAR=10\n"), 0o600))
	s.Require().NoError(os.WriteFile(filepath.Join(envDir, "00-core.env"), []byte("ORDER_VAR=00\n"), 0o600))

	err := LoadDir(envDir, false)
	s.Require().NoError(err)

	// 90-project.env is last in sort order, so it wins
	s.Equal("90", os.Getenv("ORDER_VAR"))
}

// TestLoadDirOnlyEnvFiles tests that non-.env files are ignored
func (s *EnvFileTestSuite) TestLoadDirOnlyEnvFiles() {
	envDir := filepath.Join(s.tempDir, "env")
	s.Require().NoError(os.MkdirAll(envDir, 0o750))

	s.Require().NoError(os.WriteFile(filepath.Join(envDir, "00-core.env"), []byte("CORE_VAR=core\n"), 0o600))
	s.Require().NoError(os.WriteFile(filepath.Join(envDir, "README.md"), []byte("# Env files\n"), 0o600))
	s.Require().NoError(os.WriteFile(filepath.Join(envDir, "load-env.sh"), []byte("#!/bin/bash\n"), 0o600))

	err := LoadDir(envDir, false)
	s.Require().NoError(err)

	s.Equal("core", os.Getenv("CORE_VAR"))
}

// TestEnvFileTestSuite runs the test suite
func TestEnvFileTestSuite(t *testing.T) {
	suite.Run(t, new(EnvFileTestSuite))
}
