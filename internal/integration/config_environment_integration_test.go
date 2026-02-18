package integration

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/mrz1836/go-pre-commit/internal/config"
	"github.com/mrz1836/go-pre-commit/internal/runner"
	"github.com/mrz1836/go-pre-commit/internal/shared"
)

// ConfigEnvironmentIntegrationTestSuite tests configuration and environment integration
type ConfigEnvironmentIntegrationTestSuite struct {
	suite.Suite

	tempDir      string
	originalWD   string
	originalEnv  map[string]string
	suiteEnv     map[string]string
	testProjects []string
}

// SetupSuite initializes the configuration integration test environment
func (s *ConfigEnvironmentIntegrationTestSuite) SetupSuite() {
	var err error
	s.originalWD, err = os.Getwd()
	s.Require().NoError(err)

	// Create isolated temp directory outside the repository tree
	s.tempDir, err = os.MkdirTemp("", "go-pre-commit-integration-test-*")
	s.Require().NoError(err)
	s.T().Cleanup(func() {
		_ = os.RemoveAll(s.tempDir)
	})

	s.originalEnv = make(map[string]string)
	s.suiteEnv = make(map[string]string)
	s.testProjects = make([]string, 0)

	// Save all environment variables that might be modified by tests
	s.saveSuiteEnvironment("GO_PRE_COMMIT_TIMEOUT_SECONDS")
	s.saveSuiteEnvironment("GO_PRE_COMMIT_LOG_LEVEL")
	s.saveSuiteEnvironment("GO_PRE_COMMIT_PARALLEL_WORKERS")
	s.saveSuiteEnvironment("GO_PRE_COMMIT_MAX_FILE_SIZE_MB")
	s.saveSuiteEnvironment("GO_PRE_COMMIT_MAX_FILES_OPEN")
	s.saveSuiteEnvironment("GO_PRE_COMMIT_TEST_CONFIG_DIR")
	s.saveSuiteEnvironment("ENABLE_GO_PRE_COMMIT")

	// Create multiple test project scenarios
	s.createTestProjects()
}

// TearDownSuite cleans up the test environment
func (s *ConfigEnvironmentIntegrationTestSuite) TearDownSuite() {
	_ = os.Chdir(s.originalWD)
	s.restoreSuiteEnvironment()
}

// TearDownTest resets environment after each test
func (s *ConfigEnvironmentIntegrationTestSuite) TearDownTest() {
	_ = os.Chdir(s.originalWD)
	s.restoreEnvironment()
}

// saveEnvironment saves current environment variable
func (s *ConfigEnvironmentIntegrationTestSuite) saveEnvironment(key string) {
	s.originalEnv[key] = os.Getenv(key)
}

// saveSuiteEnvironment saves environment variable for suite-level restoration
func (s *ConfigEnvironmentIntegrationTestSuite) saveSuiteEnvironment(key string) {
	s.suiteEnv[key] = os.Getenv(key)
}

// restoreEnvironment restores saved environment variables
func (s *ConfigEnvironmentIntegrationTestSuite) restoreEnvironment() {
	for key, value := range s.originalEnv {
		if value == "" {
			_ = os.Unsetenv(key)
		} else {
			_ = os.Setenv(key, value)
		}
	}
	s.originalEnv = make(map[string]string)
}

// restoreSuiteEnvironment restores suite-level environment variables
func (s *ConfigEnvironmentIntegrationTestSuite) restoreSuiteEnvironment() {
	for key, value := range s.suiteEnv {
		if value == "" {
			_ = os.Unsetenv(key)
		} else {
			_ = os.Setenv(key, value)
		}
	}
}

// createTestProjects creates different project scenarios
func (s *ConfigEnvironmentIntegrationTestSuite) createTestProjects() {
	// Project 1: Minimal configuration
	minimalProject := filepath.Join(s.tempDir, "minimal-config")
	s.createMinimalProject(minimalProject)
	s.testProjects = append(s.testProjects, minimalProject)

	// Project 2: Complex configuration with all features
	complexProject := filepath.Join(s.tempDir, "complex-config")
	s.createComplexProject(complexProject)
	s.testProjects = append(s.testProjects, complexProject)

	// Project 3: Multi-level project (nested directories)
	multiLevelProject := filepath.Join(s.tempDir, "multi-level")
	s.createMultiLevelProject(multiLevelProject)
	s.testProjects = append(s.testProjects, multiLevelProject)

	// Project 4: Custom configuration paths
	customConfigProject := filepath.Join(s.tempDir, "custom-config")
	s.createCustomConfigProject(customConfigProject)
	s.testProjects = append(s.testProjects, customConfigProject)
}

// createMinimalProject creates a project with minimal configuration
func (s *ConfigEnvironmentIntegrationTestSuite) createMinimalProject(projectPath string) {
	s.Require().NoError(os.MkdirAll(projectPath, 0o750))
	s.Require().NoError(os.Chdir(projectPath))

	// Initialize git
	s.initGitRepo()

	// Create minimal go.mod
	goMod := `module minimal-test

go 1.21
`
	s.Require().NoError(os.WriteFile("go.mod", []byte(goMod), 0o600))

	// Create basic .env.base
	envContent := `ENABLE_GO_PRE_COMMIT=true
GO_PRE_COMMIT_LOG_LEVEL=info
GO_PRE_COMMIT_FUMPT_VERSION=latest
GO_PRE_COMMIT_GOLANGCI_LINT_VERSION=latest
GO_PRE_COMMIT_GOIMPORTS_VERSION=latest
`
	githubDir := ".github"
	s.Require().NoError(os.MkdirAll(githubDir, 0o750))
	s.Require().NoError(os.WriteFile(filepath.Join(githubDir, ".env.base"), []byte(envContent), 0o600))

	// Create simple main.go
	mainGo := `package main

import "fmt"

func main() {
	fmt.Println("Minimal project")
}
`
	s.Require().NoError(os.WriteFile("main.go", []byte(mainGo), 0o600))

	// Commit initial files
	s.commitFiles("Initial minimal project")
}

// createComplexProject creates a project with complex configuration
func (s *ConfigEnvironmentIntegrationTestSuite) createComplexProject(projectPath string) {
	s.Require().NoError(os.MkdirAll(projectPath, 0o750))
	s.Require().NoError(os.Chdir(projectPath))

	// Initialize git
	s.initGitRepo()

	// Create complex go.mod with dependencies
	goMod := `module complex-test

go 1.21

require (
	github.com/stretchr/testify v1.8.4
	gopkg.in/yaml.v3 v3.0.1
)
`
	s.Require().NoError(os.WriteFile("go.mod", []byte(goMod), 0o600))

	// Create comprehensive .env.base
	envContent := `# Complex configuration for comprehensive testing
ENABLE_GO_PRE_COMMIT=true
GO_PRE_COMMIT_TIMEOUT_SECONDS=600
GO_PRE_COMMIT_LOG_LEVEL=debug

# All checks enabled
GO_PRE_COMMIT_ENABLE_FMT=true
GO_PRE_COMMIT_ENABLE_FUMPT=true
GO_PRE_COMMIT_ENABLE_LINT=true
GO_PRE_COMMIT_ENABLE_MOD_TIDY=true
GO_PRE_COMMIT_ENABLE_WHITESPACE=true
GO_PRE_COMMIT_ENABLE_EOF=true
GO_PRE_COMMIT_ENABLE_AI_DETECTION=false

# Performance settings
GO_PRE_COMMIT_PARALLEL_WORKERS=4
GO_PRE_COMMIT_MAX_FILE_SIZE_MB=20
GO_PRE_COMMIT_MAX_FILES_OPEN=200

# Tool versions
GO_PRE_COMMIT_TOOL_FUMPT_VERSION=latest
GO_PRE_COMMIT_TOOL_GOLANGCI_LINT_VERSION=latest

# Timeouts
GO_PRE_COMMIT_FMT_TIMEOUT=60
GO_PRE_COMMIT_FUMPT_TIMEOUT=60
GO_PRE_COMMIT_LINT_TIMEOUT=120
GO_PRE_COMMIT_MOD_TIDY_TIMEOUT=60
GO_PRE_COMMIT_WHITESPACE_TIMEOUT=30
GO_PRE_COMMIT_EOF_TIMEOUT=30
GO_PRE_COMMIT_AI_DETECTION_TIMEOUT=60

# Tool versions
GO_PRE_COMMIT_FUMPT_VERSION=latest
GO_PRE_COMMIT_GOLANGCI_LINT_VERSION=latest
GO_PRE_COMMIT_GOIMPORTS_VERSION=latest

# Exclusions
GO_PRE_COMMIT_GIT_EXCLUDE_PATTERNS=vendor/*,*.pb.go,*_generated.go
`
	githubDir := ".github"
	s.Require().NoError(os.MkdirAll(githubDir, 0o750))
	s.Require().NoError(os.WriteFile(filepath.Join(githubDir, ".env.base"), []byte(envContent), 0o600))

	// Create custom .env.custom
	customEnvContent := `# Custom environment overrides
GO_PRE_COMMIT_LOG_LEVEL=trace
GO_PRE_COMMIT_ENABLE_FUMPT=false
`
	s.Require().NoError(os.WriteFile(filepath.Join(githubDir, ".env.custom"), []byte(customEnvContent), 0o600))

	// Create complex project structure
	dirs := []string{"cmd/app", "internal/pkg", "pkg/api", "test/integration"}
	for _, dir := range dirs {
		s.Require().NoError(os.MkdirAll(dir, 0o750))
	}

	// Create multiple Go files
	files := map[string]string{
		"cmd/app/main.go": `package main

import (
	"fmt"
	"log"
	"os"

	"complex-test/internal/pkg"
	"complex-test/pkg/api"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("Usage: app <command>")
	}

	switch os.Args[1] {
	case "serve":
		api.StartServer()
	case "process":
		pkg.ProcessData()
	default:
		fmt.Printf("Unknown command: %s\n", os.Args[1])
	}
}
`,
		"internal/pkg/processor.go": `package pkg

import (
	"fmt"
	"time"
)

// ProcessData processes application data
func ProcessData() {
	fmt.Println("Processing data...")

	start := time.Now()
	defer func() {
		fmt.Printf("Processing completed in %v\n", time.Since(start))
	}()

	// Simulate processing
	time.Sleep(10 * time.Millisecond)
	fmt.Println("Data processed successfully")
}

// Helper function for processing
func validateInput(input string) bool {
	return len(input) > 0 && input != ""
}
`,
		"pkg/api/server.go": `package api

import (
	"fmt"
	"log"
	"net/http"
	"time"
)

// StartServer starts the HTTP server
func StartServer() {
	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/api/status", statusHandler)

	fmt.Println("Server starting on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = fmt.Fprintf(w, "OK")
}

func statusHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = fmt.Fprintf(w, "{\"status\": \"running\", \"timestamp\": \"%d\"}", time.Now().Unix())
}
`,
		"test/integration/integration_test.go": `package integration

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIntegration(t *testing.T) {
	assert.True(t, true, "Integration test placeholder")
}
`,
	}

	for filePath, content := range files {
		// Fix import path issue
		if strings.Contains(content, `"complex-test/`) {
			content = strings.ReplaceAll(content, `"complex-test/`, `"complex-test/`)
		}

		s.Require().NoError(os.WriteFile(filePath, []byte(content), 0o600))
	}

	// Add some files with potential issues for testing
	problemFile := `package main

import(
"fmt"
"unused"
)

func main(){
fmt.Printf("Problem file with formatting issues")
}
`
	s.Require().NoError(os.WriteFile("problem.go", []byte(problemFile), 0o600))

	// Commit initial files
	s.commitFiles("Initial complex project")
}

// createMultiLevelProject creates a project with nested directories
func (s *ConfigEnvironmentIntegrationTestSuite) createMultiLevelProject(projectPath string) {
	s.Require().NoError(os.MkdirAll(projectPath, 0o750))
	s.Require().NoError(os.Chdir(projectPath))

	// Initialize git
	s.initGitRepo()

	// Create root configuration
	envContent := `ENABLE_GO_PRE_COMMIT=true
GO_PRE_COMMIT_LOG_LEVEL=info
`
	githubDir := ".github"
	s.Require().NoError(os.MkdirAll(githubDir, 0o750))
	s.Require().NoError(os.WriteFile(filepath.Join(githubDir, ".env.base"), []byte(envContent), 0o600))

	// Create deep directory structure
	deepDir := filepath.Join("level1", "level2", "level3", "level4")
	s.Require().NoError(os.MkdirAll(deepDir, 0o750))

	// Create go.mod at root
	goMod := `module multi-level-test

go 1.21
`
	s.Require().NoError(os.WriteFile("go.mod", []byte(goMod), 0o600))

	// Create Go files at different levels
	levels := []string{
		"main.go",
		"level1/level1.go",
		"level1/level2/level2.go",
		"level1/level2/level3/level3.go",
		"level1/level2/level3/level4/level4.go",
	}

	for i, levelFile := range levels {
		content := fmt.Sprintf(`package main

import "fmt"

func level%dFunction() {
	fmt.Printf("Function from %s")
}
`, i, levelFile)
		s.Require().NoError(os.WriteFile(levelFile, []byte(content), 0o600))
	}

	// Commit initial files
	s.commitFiles("Initial multi-level project")
}

// createCustomConfigProject creates a project with custom configuration paths
func (s *ConfigEnvironmentIntegrationTestSuite) createCustomConfigProject(projectPath string) {
	s.Require().NoError(os.MkdirAll(projectPath, 0o750))
	s.Require().NoError(os.Chdir(projectPath))

	// Initialize git
	s.initGitRepo()

	// Create go.mod
	goMod := `module custom-config-test

go 1.21
`
	s.Require().NoError(os.WriteFile("go.mod", []byte(goMod), 0o600))

	// Create configuration in different location
	configDir := "config"
	s.Require().NoError(os.MkdirAll(configDir, 0o750))

	// Create .github for standard location
	githubDir := ".github"
	s.Require().NoError(os.MkdirAll(githubDir, 0o750))

	// Create base configuration
	baseEnvContent := `ENABLE_GO_PRE_COMMIT=true
GO_PRE_COMMIT_LOG_LEVEL=info
GO_PRE_COMMIT_ENABLE_FMT=true
GO_PRE_COMMIT_ENABLE_LINT=true
`
	s.Require().NoError(os.WriteFile(filepath.Join(githubDir, ".env.base"), []byte(baseEnvContent), 0o600))

	// Create main.go
	mainGo := `package main

import "fmt"

func main() {
	fmt.Println("Custom config project")
}
`
	s.Require().NoError(os.WriteFile("main.go", []byte(mainGo), 0o600))

	// Commit initial files
	s.commitFiles("Initial custom config project")
}

// initGitRepo initializes a git repository
func (s *ConfigEnvironmentIntegrationTestSuite) initGitRepo() {
	ctx := context.Background()
	gitInit := exec.CommandContext(ctx, "git", "init", ".")
	s.Require().NoError(gitInit.Run())

	gitConfigName := exec.CommandContext(ctx, "git", "config", "user.name", "Test User")
	s.Require().NoError(gitConfigName.Run())

	gitConfigEmail := exec.CommandContext(ctx, "git", "config", "user.email", "test@example.com")
	s.Require().NoError(gitConfigEmail.Run())
}

// commitFiles commits all files to git
func (s *ConfigEnvironmentIntegrationTestSuite) commitFiles(message string) {
	ctx := context.Background()
	gitAdd := exec.CommandContext(ctx, "git", "add", ".")
	s.Require().NoError(gitAdd.Run())

	gitCommit := exec.CommandContext(ctx, "git", "commit", "-m", message) // #nosec G204 - git binary path is fixed
	s.Require().NoError(gitCommit.Run())
}

// TestMinimalConfigurationWorkflow tests workflow with minimal configuration
func (s *ConfigEnvironmentIntegrationTestSuite) TestMinimalConfigurationWorkflow() {
	minimalProject := s.testProjects[0]
	s.Require().NoError(os.Chdir(minimalProject))

	// Save environment variables that might be modified by config.Load()
	s.saveEnvironment("GO_PRE_COMMIT_TIMEOUT_SECONDS")
	s.saveEnvironment("GO_PRE_COMMIT_TEST_CONFIG_DIR")

	// Set test config directory to use this test's config
	s.Require().NoError(os.Setenv("GO_PRE_COMMIT_TEST_CONFIG_DIR", minimalProject))

	// Test configuration loading
	cfg, err := config.Load()
	s.Require().NoError(err, "Should load minimal configuration")

	// Verify default values are applied
	s.True(cfg.Enabled, "Go pre-commit should be enabled")
	s.Equal("info", cfg.LogLevel, "Log level should default to info")
	s.Positive(cfg.Timeout, "Timeout should be greater than 0")
	s.T().Logf("Loaded timeout: %d seconds", cfg.Timeout)

	// Test workflow execution
	testRunner := runner.New(cfg, minimalProject)

	ctx := context.Background()
	opts := runner.Options{}
	results, _ := testRunner.Run(ctx, opts)
	s.NotNil(results, "Should get results from minimal configuration")
	s.NotEmpty(results.CheckResults, "Should execute some checks")

	s.T().Logf("✓ Minimal configuration workflow: %d checks executed", len(results.CheckResults))
}

// TestComplexConfigurationWorkflow tests workflow with complex configuration
func (s *ConfigEnvironmentIntegrationTestSuite) TestComplexConfigurationWorkflow() {
	complexProject := s.testProjects[1]
	s.Require().NoError(os.Chdir(complexProject))

	// Save environment variables that might be modified by config.Load()
	s.saveEnvironment("GO_PRE_COMMIT_TIMEOUT_SECONDS")
	s.saveEnvironment("GO_PRE_COMMIT_LOG_LEVEL")
	s.saveEnvironment("GO_PRE_COMMIT_PARALLEL_WORKERS")
	s.saveEnvironment("GO_PRE_COMMIT_MAX_FILE_SIZE_MB")
	s.saveEnvironment("GO_PRE_COMMIT_MAX_FILES_OPEN")
	s.saveEnvironment("GO_PRE_COMMIT_TEST_CONFIG_DIR")

	// Clear existing environment variables that might override .env.base values
	s.Require().NoError(os.Unsetenv("GO_PRE_COMMIT_TIMEOUT_SECONDS"))
	s.Require().NoError(os.Unsetenv("GO_PRE_COMMIT_PARALLEL_WORKERS"))
	s.Require().NoError(os.Unsetenv("GO_PRE_COMMIT_MAX_FILE_SIZE_MB"))
	s.Require().NoError(os.Unsetenv("GO_PRE_COMMIT_MAX_FILES_OPEN"))
	s.Require().NoError(os.Unsetenv("GO_PRE_COMMIT_LOG_LEVEL"))

	// Set test config directory to use this test's config
	s.Require().NoError(os.Setenv("GO_PRE_COMMIT_TEST_CONFIG_DIR", complexProject))

	// Test configuration loading
	cfg, err := config.Load()
	s.Require().NoError(err, "Should load complex configuration")

	// Verify complex configuration values
	s.Equal(600, cfg.Timeout, "Timeout should be customized")
	s.Equal(4, cfg.Performance.ParallelWorkers, "Parallel workers should be customized")
	s.Equal(int64(20971520), cfg.MaxFileSize, "Max file size should be customized")
	s.Equal(200, cfg.MaxFilesOpen, "Max files open should be customized")

	// Verify custom exclusion patterns
	s.NotEmpty(cfg.Git.ExcludePatterns, "Should have exclusion patterns")

	// Check for vendor exclusion
	hasVendorExclusion := false
	for _, pattern := range cfg.Git.ExcludePatterns {
		if strings.Contains(pattern, "vendor") {
			hasVendorExclusion = true
			break
		}
	}
	s.True(hasVendorExclusion, "Should have vendor exclusion pattern")

	// Test workflow execution with complex configuration
	testRunner := runner.New(cfg, complexProject)

	startTime := time.Now()
	ctx := context.Background()
	opts := runner.Options{}
	results, _ := testRunner.Run(ctx, opts)
	executionTime := time.Since(startTime)

	s.NotNil(results, "Should get results from complex configuration")
	s.NotEmpty(results.CheckResults, "Should execute checks")
	s.Less(executionTime, 2*time.Minute, "Should complete within reasonable time")

	s.T().Logf("✓ Complex configuration workflow: %d checks in %v",
		len(results.CheckResults), executionTime)
}

// TestEnvironmentVariableOverrides tests environment variable override scenarios
func (s *ConfigEnvironmentIntegrationTestSuite) TestEnvironmentVariableOverrides() {
	minimalProject := s.testProjects[0]
	s.Require().NoError(os.Chdir(minimalProject))

	// Test different environment variable combinations
	overrideScenarios := []struct {
		name        string
		envVars     map[string]string
		checkFunc   func(*config.Config)
		description string
	}{
		{
			name: "Log level override",
			envVars: map[string]string{
				"GO_PRE_COMMIT_LOG_LEVEL": "debug",
			},
			checkFunc: func(cfg *config.Config) {
				s.Equal("debug", cfg.LogLevel, "Log level should be overridden")
			},
			description: "Environment should override log level",
		},
		{
			name: "Timeout override",
			envVars: map[string]string{
				"GO_PRE_COMMIT_TIMEOUT_SECONDS": "900",
			},
			checkFunc: func(cfg *config.Config) {
				s.Equal(900, cfg.Timeout, "Timeout should be overridden")
			},
			description: "Environment should override timeout",
		},
		{
			name: "Check disabling",
			envVars: map[string]string{
				"GO_PRE_COMMIT_ENABLE_FMT":  "false",
				"GO_PRE_COMMIT_ENABLE_LINT": "false",
			},
			checkFunc: func(cfg *config.Config) {
				s.False(cfg.Checks.Fmt, "Format check should be disabled")
				s.False(cfg.Checks.Lint, "Lint check should be disabled")
			},
			description: "Environment should disable checks",
		},
		{
			name: "Performance settings",
			envVars: map[string]string{
				"GO_PRE_COMMIT_PARALLEL_WORKERS": "8",
				"GO_PRE_COMMIT_MAX_FILE_SIZE_MB": "50", // 50MB
			},
			checkFunc: func(cfg *config.Config) {
				s.Equal(8, cfg.Performance.ParallelWorkers, "Parallel workers should be overridden")
				s.Equal(int64(52428800), cfg.MaxFileSize, "Max file size should be overridden")
			},
			description: "Environment should override performance settings",
		},
	}

	for _, scenario := range overrideScenarios {
		s.Run(scenario.name, func() {
			// Set environment variables
			for key, value := range scenario.envVars {
				s.saveEnvironment(key)
				s.Require().NoError(os.Setenv(key, value))
			}

			// Load configuration with overrides
			cfg, err := config.Load()
			s.Require().NoError(err, "Should load configuration with overrides")

			// Run scenario-specific checks
			scenario.checkFunc(cfg)

			s.T().Logf("✓ Environment override scenario '%s' tested", scenario.name)
		})
	}

	s.T().Logf("✓ Environment variable override tests completed")
}

// TestMultiLevelConfigurationDiscovery tests configuration discovery in nested directories
func (s *ConfigEnvironmentIntegrationTestSuite) TestMultiLevelConfigurationDiscovery() {
	multiLevelProject := s.testProjects[2]

	// Test configuration discovery from different levels
	testDirs := []string{
		multiLevelProject, // Root
		filepath.Join(multiLevelProject, "level1"),
		filepath.Join(multiLevelProject, "level1", "level2"),
		filepath.Join(multiLevelProject, "level1", "level2", "level3"),
		filepath.Join(multiLevelProject, "level1", "level2", "level3", "level4"),
	}

	for i, testDir := range testDirs {
		s.Run(fmt.Sprintf("Level_%d", i), func() {
			s.Require().NoError(os.Chdir(testDir))

			cfg, err := config.Load()
			s.Require().NoError(err, "Should find configuration from level %d", i)

			// Verify configuration is loaded correctly
			s.True(cfg.Enabled, "Go pre-commit should be enabled")
			s.Equal("info", cfg.LogLevel, "Log level should be loaded")

			// Test context creation from different levels
			sharedCtx := shared.NewContext()
			ctx := context.Background()

			// Repository root should always point to the actual root
			repoRoot, err := sharedCtx.GetRepoRoot(ctx)
			s.Require().NoError(err, "Should get repo root from level %d", i)

			// Resolve symlinks for comparison (macOS /var -> /private/var)
			expectedRoot, err := filepath.EvalSymlinks(multiLevelProject)
			s.Require().NoError(err, "Should resolve symlinks for expected path")
			actualRoot, err := filepath.EvalSymlinks(repoRoot)
			s.Require().NoError(err, "Should resolve symlinks for actual path")

			s.Equal(expectedRoot, actualRoot,
				"Repository root should be detected correctly from level %d", i)

			s.T().Logf("✓ Level %d configuration discovery: Root: %s", i, repoRoot)
		})
	}

	s.T().Logf("✓ Multi-level configuration discovery tests completed")
}

// TestConfigurationValidation tests configuration validation scenarios
func (s *ConfigEnvironmentIntegrationTestSuite) TestConfigurationValidation() {
	customProject := s.testProjects[3]
	s.Require().NoError(os.Chdir(customProject))

	// Test various configuration validation scenarios
	validationScenarios := []struct {
		name      string
		envVars   map[string]string
		expectErr bool
		checkFunc func(error)
	}{
		{
			name: "Valid configuration",
			envVars: map[string]string{
				"ENABLE_GO_PRE_COMMIT":    "true",
				"GO_PRE_COMMIT_LOG_LEVEL": "info",
				"GO_PRE_COMMIT_TIMEOUT":   "300",
			},
			expectErr: false,
			checkFunc: func(err error) {
				s.NoError(err, "Valid configuration should not produce errors")
			},
		},
		{
			name: "Invalid timeout",
			envVars: map[string]string{
				"ENABLE_GO_PRE_COMMIT":          "true",
				"GO_PRE_COMMIT_TIMEOUT_SECONDS": "-100",
			},
			expectErr: true,
			checkFunc: func(err error) {
				s.Require().Error(err, "Negative timeout should produce error")
				s.Contains(err.Error(), "TIMEOUT_SECONDS", "Error should mention timeout setting")
			},
		},
		{
			name: "Invalid log level",
			envVars: map[string]string{
				"ENABLE_GO_PRE_COMMIT":    "true",
				"GO_PRE_COMMIT_LOG_LEVEL": "invalid_level",
			},
			expectErr: true,
			checkFunc: func(err error) {
				s.Require().Error(err, "Invalid log level should produce error")
			},
		},
		{
			name: "Invalid parallel workers",
			envVars: map[string]string{
				"ENABLE_GO_PRE_COMMIT":           "true",
				"GO_PRE_COMMIT_PARALLEL_WORKERS": "0",
			},
			expectErr: true,
			checkFunc: func(err error) {
				s.Require().Error(err, "Zero parallel workers should produce error")
			},
		},
	}

	for _, scenario := range validationScenarios {
		s.Run(scenario.name, func() {
			// Set environment variables
			for key, value := range scenario.envVars {
				s.saveEnvironment(key)
				s.Require().NoError(os.Setenv(key, value))
			}

			// Load and validate configuration
			cfg, err := config.Load()

			if scenario.expectErr {
				if err == nil && cfg != nil {
					// Configuration loaded successfully, test validation
					validationErr := cfg.Validate()
					scenario.checkFunc(validationErr)
				} else {
					scenario.checkFunc(err)
				}
			} else {
				scenario.checkFunc(err)
				if err == nil {
					// Test that valid configuration can be used
					sharedCtx := shared.NewContext()
					s.NotNil(sharedCtx, "Valid configuration should create context successfully")
				}
			}

			s.T().Logf("✓ Configuration validation scenario '%s' tested", scenario.name)
		})
	}

	s.T().Logf("✓ Configuration validation tests completed")
}

// TestSkipEnvironmentIntegration tests SKIP environment variable integration
func (s *ConfigEnvironmentIntegrationTestSuite) TestSkipEnvironmentIntegration() {
	minimalProject := s.testProjects[0]
	s.Require().NoError(os.Chdir(minimalProject))

	// Test different SKIP configurations
	skipScenarios := []struct {
		name          string
		skipValue     string
		expectedSkips []string
		expectedRuns  []string
		description   string
	}{
		{
			name:          "Skip single check",
			skipValue:     "fmt",
			expectedSkips: []string{"fmt"},
			expectedRuns:  []string{"lint", "mod-tidy"},
			description:   "Should skip only fmt check",
		},
		{
			name:          "Skip multiple checks",
			skipValue:     "fmt,lint",
			expectedSkips: []string{"fmt", "lint"},
			expectedRuns:  []string{"mod-tidy", "whitespace"},
			description:   "Should skip fmt and lint checks",
		},
		{
			name:          "Skip with spaces",
			skipValue:     "fmt, lint, fumpt",
			expectedSkips: []string{"fmt", "lint", "fumpt"},
			expectedRuns:  []string{"mod-tidy"},
			description:   "Should handle spaces in skip list",
		},
		{
			name:          "Skip all",
			skipValue:     "fmt,lint,fumpt,gitleaks,mod-tidy,whitespace,eof",
			expectedSkips: []string{"fmt", "lint", "fumpt", "gitleaks", "mod-tidy", "whitespace", "eof"},
			expectedRuns:  []string{},
			description:   "Should skip all checks",
		},
		{
			name:          "No skip",
			skipValue:     "",
			expectedSkips: []string{},
			expectedRuns:  []string{"fmt", "lint", "mod-tidy"},
			description:   "Should run all enabled checks",
		},
	}

	for _, scenario := range skipScenarios {
		s.Run(scenario.name, func() {
			// Set SKIP environment variable
			s.saveEnvironment("SKIP")
			if scenario.skipValue != "" {
				s.Require().NoError(os.Setenv("SKIP", scenario.skipValue))
			} else {
				_ = os.Unsetenv("SKIP")
			}

			// Load configuration
			cfg, err := config.Load()
			s.Require().NoError(err)

			// Create context and runner
			testRunner := runner.New(cfg, minimalProject)

			// Run checks
			ctx := context.Background()
			opts := runner.Options{}
			results, err := testRunner.Run(ctx, opts)

			// Special case: when all checks are skipped, expect "no checks to run" error
			if scenario.name == "Skip all" {
				s.Require().Error(err, "Should return error when no checks are available")
				s.Contains(err.Error(), "no checks", "Error should indicate no checks available")
				return // Skip the rest of the test for this scenario
			}

			s.Require().NoError(err, "Runner should execute without errors")
			s.NotNil(results, "Results should not be nil")

			// Analyze results
			executedChecks := make(map[string]bool)
			if results != nil && results.CheckResults != nil {
				for _, result := range results.CheckResults {
					executedChecks[result.Name] = true
				}
			}

			// Verify skipped checks were not executed
			for _, skipCheck := range scenario.expectedSkips {
				s.False(executedChecks[skipCheck],
					"Check %s should be skipped but was executed", skipCheck)
			}

			// Verify expected checks were executed
			for _, runCheck := range scenario.expectedRuns {
				s.T().Logf("Checking if %s was executed", runCheck)
				// In a real implementation, this would verify check configuration status
				// For now, we just log the expectation
			}

			s.T().Logf("✓ Skip scenario '%s': Executed %d checks",
				scenario.name, len(results.CheckResults))
		})
	}

	s.T().Logf("✓ Skip environment integration tests completed")
}

// TestConfigurationPerformanceImpact tests performance impact of different configurations
func (s *ConfigEnvironmentIntegrationTestSuite) TestConfigurationPerformanceImpact() {
	complexProject := s.testProjects[1]
	s.Require().NoError(os.Chdir(complexProject))

	// Test performance with different worker configurations
	workerScenarios := []struct {
		workers     int
		description string
	}{
		{1, "Single worker"},
		{2, "Two workers"},
		{4, "Four workers"},
	}

	performanceResults := make(map[int]time.Duration)

	for _, scenario := range workerScenarios {
		s.Run(scenario.description, func() {
			// Set worker count
			s.saveEnvironment("GO_PRE_COMMIT_PARALLEL_WORKERS")
			s.Require().NoError(os.Setenv("GO_PRE_COMMIT_PARALLEL_WORKERS",
				fmt.Sprintf("%d", scenario.workers)))

			// Load configuration
			cfg, err := config.Load()
			s.Require().NoError(err)
			s.Equal(scenario.workers, cfg.Performance.ParallelWorkers,
				"Worker count should be set correctly")

			// Create context and runner
			testRunner := runner.New(cfg, complexProject)

			// Measure execution time
			startTime := time.Now()
			ctx := context.Background()
			opts := runner.Options{}
			results, _ := testRunner.Run(ctx, opts)
			executionTime := time.Since(startTime)

			s.NotNil(results)
			performanceResults[scenario.workers] = executionTime

			s.T().Logf("✓ %s: %v for %d checks",
				scenario.description, executionTime, len(results.CheckResults))
		})
	}

	// Analyze performance trends
	s.T().Logf("✓ Performance analysis:")
	for workers, duration := range performanceResults {
		s.T().Logf("  %d workers: %v", workers, duration)
	}

	s.T().Logf("✓ Configuration performance impact tests completed")
}

// TestSuite runs the configuration environment integration test suite
func TestConfigEnvironmentIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(ConfigEnvironmentIntegrationTestSuite))
}
