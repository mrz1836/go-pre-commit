package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	prerrors "github.com/mrz1836/go-pre-commit/internal/errors"
	"github.com/mrz1836/go-pre-commit/internal/git"
	"github.com/mrz1836/go-pre-commit/internal/runner"
	"github.com/mrz1836/go-pre-commit/internal/testutil"
	"github.com/mrz1836/go-pre-commit/internal/tools"
)

// testVersion is the binary version the JSON tests build the CLI with
const testVersion = "1.2.3-test"

// errTestUncoded is an error that carries no JSON error code
var errTestUncoded = errors.New("an uncoded error")

// The keys every JSON document must have, exactly: adding, renaming, or
// removing a field fails these tests, so the documented shape cannot drift.
func statusReportKeys() []string {
	return []string{"config", "enabled", "hooks", "hooks_dir", "repository_root", "version"}
}

func statusConfigKeys() []string { return []string{"mode", "path"} }

func statusHookKeys() []string {
	return []string{"executable", "name", "outdated", "path", "state"}
}

func errorReportKeys() []string { return []string{"error", "version"} }

func errorObjectKeys() []string { return []string{"code", "message"} }

func runReportKeys() []string {
	return []string{"checks", "enabled", "files", "result", "totals", "version", "warnings"}
}

func runCheckKeys() []string {
	return []string{"changed_files", "duration_ms", "files", "name", "result"}
}

func runTotalsKeys() []string {
	return []string{"checks", "duration_ms", "failed", "files", "passed", "skipped"}
}

// jsonTestRepo creates a git repository (with an isolated git, see
// testutil.NewGitRepo) holding go-pre-commit configuration, changes into it, and
// returns its root. modular selects .github/env/00-test.env over the legacy
// .github/.env.base. The test sees no go-pre-commit configuration but this
// file (see isolateConfigEnv), and nothing but git on PATH, with empty GOPATH
// and GOBIN, so no check can use or install a tool from the developer's
// machine; a test that needs a tool puts a fake one on PATH with isolatePath.
func jsonTestRepo(t *testing.T, modular bool, settings string) string {
	t.Helper()
	isolateConfigEnv(t)
	preserveWorkingDir(t)
	repo := testutil.NewGitRepo(t)
	isolatePath(t, nil)
	t.Setenv("GOPATH", t.TempDir())
	t.Setenv("GOBIN", t.TempDir())

	configPath := filepath.Join(repo, ".github", ".env.base")
	if modular {
		configPath = filepath.Join(repo, ".github", "env", "00-test.env")
	}
	require.NoError(t, os.MkdirAll(filepath.Dir(configPath), 0o750))
	require.NoError(t, os.WriteFile(configPath, []byte(settings), 0o600))

	t.Chdir(repo)
	return repo
}

// isolateConfigEnv restores the process environment when the test ends, and
// starts the test with no go-pre-commit configuration in it. config.Load
// exports every value it reads, and the legacy .github/.env.base never
// overrides a value already set, so settings another test in this package
// left behind would otherwise replace the test's own configuration.
func isolateConfigEnv(t *testing.T) {
	t.Helper()
	testutil.IsolateEnv(t)
	require.NoError(t, testutil.ClearConfigEnv())
}

// builtinChecksOnly enables just the checks that need no external tool, and
// turns automatic installs off, so a tool check enabled by mistake fails
// rather than installs
const builtinChecksOnly = `ENABLE_GO_PRE_COMMIT=true
GO_PRE_COMMIT_AUTO_INSTALL_TOOLS=false
GO_PRE_COMMIT_ENABLE_FUMPT=false
GO_PRE_COMMIT_ENABLE_LINT=false
GO_PRE_COMMIT_ENABLE_MOD_TIDY=false
GO_PRE_COMMIT_ENABLE_GITLEAKS=false
GO_PRE_COMMIT_ENABLE_WHITESPACE=true
GO_PRE_COMMIT_ENABLE_EOF=true
GO_PRE_COMMIT_WHITESPACE_AUTO_STAGE=false
`

// executeCLI runs the CLI in-process with args, capturing the process's real
// stdout and stderr (not just cobra's writers), so the tests see everything
// any part of the program prints
func executeCLI(t *testing.T, args ...string) (stdout, stderr string, err error) {
	t.Helper()
	builder := NewCommandBuilder(NewCLIApp(testVersion, "test-commit", "test-date"))
	root := builder.BuildRootCmd()
	root.AddCommand(builder.BuildStatusCmd(), builder.BuildRunCmd())
	root.SetArgs(args)

	stdout, stderr = captureStdio(t, func() { err = root.Execute() })
	return stdout, stderr, err
}

// captureStdio runs fn with os.Stdout and os.Stderr replaced by pipes and
// returns what was written to each
func captureStdio(t *testing.T, fn func()) (stdout, stderr string) {
	t.Helper()
	outR, outW, err := os.Pipe()
	require.NoError(t, err)
	errR, errW, err := os.Pipe()
	require.NoError(t, err)

	var outBuf, errBuf bytes.Buffer
	var wg sync.WaitGroup
	wg.Add(2)
	go func() { defer wg.Done(); _, _ = io.Copy(&outBuf, outR) }()
	go func() { defer wg.Done(); _, _ = io.Copy(&errBuf, errR) }()

	func() {
		oldStdout, oldStderr := os.Stdout, os.Stderr
		os.Stdout, os.Stderr = outW, errW
		defer func() { os.Stdout, os.Stderr = oldStdout, oldStderr }()
		fn()
	}()

	_ = outW.Close()
	_ = errW.Close()
	wg.Wait()
	return outBuf.String(), errBuf.String()
}

// decodeSingleJSON decodes stdout as exactly one JSON object with nothing else
// around it
func decodeSingleJSON(t *testing.T, stdout string) map[string]any {
	t.Helper()
	decoder := json.NewDecoder(strings.NewReader(stdout))
	var doc map[string]any
	require.NoError(t, decoder.Decode(&doc), "stdout must be a JSON object, got:\n%s", stdout)
	_, err := decoder.Token()
	require.ErrorIs(t, err, io.EOF, "stdout must hold exactly one JSON document, got:\n%s", stdout)
	return doc
}

// keysOf returns the sorted keys of a JSON object
func keysOf(t *testing.T, value any) []string {
	t.Helper()
	object, ok := value.(map[string]any)
	require.True(t, ok, "expected a JSON object, got %T", value)
	return slices.Sorted(maps.Keys(object))
}

// objects returns a JSON array of objects
func objects(t *testing.T, value any) []map[string]any {
	t.Helper()
	array, ok := value.([]any)
	require.True(t, ok, "expected a JSON array, got %T (%v)", value, value)
	out := make([]map[string]any, 0, len(array))
	for _, item := range array {
		object, isObject := item.(map[string]any)
		require.True(t, isObject, "expected a JSON object, got %T", item)
		out = append(out, object)
	}
	return out
}

// requireErrorReport checks that stdout is the error document with code
func requireErrorReport(t *testing.T, stdout, code string) {
	t.Helper()
	doc := decodeSingleJSON(t, stdout)
	assert.Equal(t, errorReportKeys(), keysOf(t, doc))
	assert.Equal(t, testVersion, doc["version"])
	assert.Equal(t, errorObjectKeys(), keysOf(t, doc["error"]))
	errorObject, ok := doc["error"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, code, errorObject["code"])
	assert.NotEmpty(t, errorObject["message"])
}

// ---- status --json ----

func TestStatusJSONHookStates(t *testing.T) {
	ourScript := git.NewInstaller("", "").GenerateHookScript()
	oldScript := "#!/bin/bash\n# Go Pre-commit Hook\n# written by an older release\nexec go-pre-commit run\n"

	tests := []struct {
		name           string
		content        string
		mode           os.FileMode
		wantState      string
		wantExecutable bool
		wantOutdated   bool
	}{
		{name: "missing"},
		{name: "ready", content: ourScript, mode: 0o755, wantState: hookStateReady, wantExecutable: true},
		{name: "outdated", content: oldScript, mode: 0o755, wantState: hookStateOutdated, wantExecutable: true, wantOutdated: true},
		{name: "not executable", content: ourScript, mode: 0o644, wantState: hookStateNotExecutable},
		{name: "outdated and not executable", content: oldScript, mode: 0o644, wantState: hookStateNotExecutable, wantOutdated: true},
		{name: "another tool's hook", content: "#!/bin/sh\nexec other-tool\n", mode: 0o755, wantState: hookStateOther, wantExecutable: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := jsonTestRepo(t, true, "ENABLE_GO_PRE_COMMIT=true\n")
			hooksDir := filepath.Join(repo, ".git", "hooks")
			if tt.content != "" {
				hookPath := filepath.Join(hooksDir, hookTypePreCommit)
				require.NoError(t, os.WriteFile(hookPath, []byte(tt.content), 0o600))
				require.NoError(t, os.Chmod(hookPath, tt.mode))
			}

			stdout, stderr, err := executeCLI(t, "status", "--json")
			require.NoError(t, err, "stderr: %s", stderr)

			doc := decodeSingleJSON(t, stdout)
			assert.Equal(t, statusReportKeys(), keysOf(t, doc))
			assert.Equal(t, testVersion, doc["version"])
			assert.Equal(t, true, doc["enabled"])
			assert.Equal(t, repo, doc["repository_root"])
			assert.Equal(t, hooksDir, doc["hooks_dir"])

			hooks := objects(t, doc["hooks"])
			require.Len(t, hooks, 4)
			for i, name := range []string{hookTypePreCommit, hookTypePrePush, "commit-msg", "post-commit"} {
				hook := hooks[i]
				assert.Equal(t, statusHookKeys(), keysOf(t, hook))
				assert.Equal(t, name, hook["name"])
				assert.Equal(t, filepath.Join(hooksDir, name), hook["path"])
				if name != hookTypePreCommit || tt.content == "" {
					assert.Equal(t, hookStateMissing, hook["state"], name)
					assert.Equal(t, false, hook["executable"], name)
					assert.Equal(t, false, hook["outdated"], name)
					continue
				}
				assert.Equal(t, tt.wantState, hook["state"])
				assert.Equal(t, tt.wantExecutable, hook["executable"])
				assert.Equal(t, tt.wantOutdated, hook["outdated"])
			}
		})
	}
}

func TestStatusJSONUnreadableHook(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root can read any file")
	}
	repo := jsonTestRepo(t, true, "ENABLE_GO_PRE_COMMIT=true\n")
	hookPath := filepath.Join(repo, ".git", "hooks", hookTypePreCommit)
	require.NoError(t, os.WriteFile(hookPath, []byte("#!/bin/sh\n"), 0o600))
	require.NoError(t, os.Chmod(hookPath, 0o000))
	t.Cleanup(func() { _ = os.Chmod(hookPath, 0o600) }) // so TempDir cleanup can remove it

	stdout, stderr, err := executeCLI(t, "status", "--json")
	require.NoError(t, err, "stderr: %s", stderr)

	hook := objects(t, decodeSingleJSON(t, stdout)["hooks"])[0]
	assert.Equal(t, slices.Sorted(slices.Values(append(statusHookKeys(), "error"))), keysOf(t, hook))
	assert.Equal(t, hookStateError, hook["state"])
	assert.Contains(t, hook["error"], "permission denied")
}

func TestStatusJSONConfigSource(t *testing.T) {
	for _, tt := range []struct {
		name     string
		modular  bool
		wantMode string
		wantPath string
	}{
		{name: "modular", modular: true, wantMode: "modular", wantPath: filepath.Join(".github", "env")},
		{name: "legacy", modular: false, wantMode: "legacy", wantPath: filepath.Join(".github", ".env.base")},
	} {
		t.Run(tt.name, func(t *testing.T) {
			repo := jsonTestRepo(t, tt.modular, "ENABLE_GO_PRE_COMMIT=true\n")

			stdout, stderr, err := executeCLI(t, "status", "--json")
			require.NoError(t, err, "stderr: %s", stderr)

			doc := decodeSingleJSON(t, stdout)
			assert.Equal(t, statusConfigKeys(), keysOf(t, doc["config"]))
			config, ok := doc["config"].(map[string]any)
			require.True(t, ok)
			assert.Equal(t, tt.wantMode, config["mode"])
			assert.Equal(t, filepath.Join(repo, tt.wantPath), config["path"])
		})
	}
}

func TestStatusJSONDisabled(t *testing.T) {
	jsonTestRepo(t, true, "ENABLE_GO_PRE_COMMIT=false\n")

	stdout, stderr, err := executeCLI(t, "status", "--json")
	require.NoError(t, err, "stderr: %s", stderr)

	doc := decodeSingleJSON(t, stdout)
	assert.Equal(t, statusReportKeys(), keysOf(t, doc))
	assert.Equal(t, false, doc["enabled"])
	assert.Len(t, objects(t, doc["hooks"]), 4, "hooks are reported even when the system is disabled")
}

func TestStatusJSONErrors(t *testing.T) {
	t.Run("no configuration", func(t *testing.T) {
		isolateConfigEnv(t)
		preserveWorkingDir(t)
		t.Chdir(testutil.NewGitRepo(t))

		stdout, _, err := executeCLI(t, "status", "--json")
		require.Error(t, err, "a missing configuration still fails the command")
		require.ErrorIs(t, err, prerrors.ErrEnvFileNotFound)
		requireErrorReport(t, stdout, errCodeConfigNotFound)
	})

	t.Run("invalid configuration", func(t *testing.T) {
		jsonTestRepo(t, true, "GO_PRE_COMMIT_TIMEOUT_SECONDS=0\n")

		stdout, _, err := executeCLI(t, "status", "--json")
		require.Error(t, err)
		requireErrorReport(t, stdout, errCodeConfigInvalid)
	})

	t.Run("not a repository", func(t *testing.T) {
		isolateConfigEnv(t)
		testutil.IsolateGit(t)
		preserveWorkingDir(t)
		dir := t.TempDir()
		require.NoError(t, os.MkdirAll(filepath.Join(dir, ".github"), 0o750))
		require.NoError(t, os.WriteFile(filepath.Join(dir, ".github", ".env.base"), []byte("ENABLE_GO_PRE_COMMIT=true\n"), 0o600))
		t.Setenv("GIT_CEILING_DIRECTORIES", filepath.Dir(dir))
		t.Chdir(dir)

		stdout, _, err := executeCLI(t, "status", "--json")
		require.Error(t, err)
		requireErrorReport(t, stdout, errCodeNotARepository)
	})
}

func TestStatusWithoutJSONIsUnchanged(t *testing.T) {
	jsonTestRepo(t, true, "ENABLE_GO_PRE_COMMIT=true\n")

	stdout, stderr, err := executeCLI(t, "--no-color", "status")
	require.NoError(t, err)

	assert.NotContains(t, stdout, "{")
	assert.Contains(t, stdout, "Go Pre-commit System Status")
	assert.Contains(t, stderr, "No hooks are currently installed")
}

// ---- run --json ----

// runJSONSettings enables the built-in checks and fumpt, which the tests back
// with a fake gofumpt on an isolated PATH (see isolatePath)
const runJSONSettings = `ENABLE_GO_PRE_COMMIT=true
GO_PRE_COMMIT_AUTO_INSTALL_TOOLS=false
GO_PRE_COMMIT_ENABLE_FUMPT=true
GO_PRE_COMMIT_ENABLE_LINT=false
GO_PRE_COMMIT_ENABLE_MOD_TIDY=false
GO_PRE_COMMIT_ENABLE_GITLEAKS=false
GO_PRE_COMMIT_ENABLE_WHITESPACE=true
GO_PRE_COMMIT_ENABLE_EOF=true
GO_PRE_COMMIT_WHITESPACE_AUTO_STAGE=false
`

// isolatePath replaces PATH with one holding only git and the given fake
// tools, keyed by command name, so no check can install a real tool
func isolatePath(t *testing.T, scripts map[string]string) {
	t.Helper()
	testutil.IsolatedPath(t, scripts)
	tools.CleanCache()
	t.Cleanup(tools.CleanCache)
}

// checksByName indexes a run report's checks by name, checking each one's keys
func checksByName(t *testing.T, doc map[string]any, extraKeys map[string][]string) map[string]map[string]any {
	t.Helper()
	byName := make(map[string]map[string]any)
	for _, check := range objects(t, doc["checks"]) {
		name, ok := check["name"].(string)
		require.True(t, ok)
		wantKeys := slices.Sorted(slices.Values(append(runCheckKeys(), extraKeys[name]...)))
		assert.Equal(t, wantKeys, keysOf(t, check), "keys of check %s", name)
		byName[name] = check
	}
	return byName
}

func TestRunJSONReportsEachCheck(t *testing.T) {
	repo := jsonTestRepo(t, true, runJSONSettings)
	isolatePath(t, map[string]string{"gofumpt": testutil.FakeGofumpt})
	require.NoError(t, os.WriteFile("clean.go", []byte("package main\n"), 0o600))
	require.NoError(t, os.WriteFile("messy.go", []byte("package main // messy\n"), 0o600))
	require.NoError(t, os.WriteFile("notes.md", []byte("trailing   \n"), 0o600))

	stdout, stderr, err := executeCLI(t, "run", "--json", "--parallel", "1", "--files", "clean.go,messy.go,notes.md")

	require.ErrorIs(t, err, prerrors.ErrChecksFailed, "the exit status still reports the failed check")
	doc := decodeSingleJSON(t, stdout)
	assert.Equal(t, runReportKeys(), keysOf(t, doc))
	assert.Equal(t, testVersion, doc["version"])
	assert.Equal(t, true, doc["enabled"])
	assert.Equal(t, runResultFailed, doc["result"])
	assert.Equal(t, []any{"clean.go", "messy.go", "notes.md"}, doc["files"])
	assert.Equal(t, []any{}, doc["warnings"])

	checks := objects(t, doc["checks"])
	names := make([]any, 0, len(checks))
	for _, check := range checks {
		names = append(names, check["name"])
	}
	assert.Equal(t, []any{"eof", "fumpt", "whitespace"}, names, "checks are sorted by name")

	byName := checksByName(t, doc, map[string][]string{"whitespace": {"error"}})

	eof := byName["eof"]
	assert.Equal(t, checkResultPassed, eof["result"])
	assert.Equal(t, []any{"clean.go", "messy.go", "notes.md"}, eof["files"])
	assert.Equal(t, []any{}, eof["changed_files"], "a fixing check that changed nothing reports an empty list")

	fumpt := byName["fumpt"]
	assert.Equal(t, checkResultPassed, fumpt["result"])
	assert.Equal(t, []any{"clean.go", "messy.go"}, fumpt["files"])
	assert.Equal(t, []any{"messy.go"}, fumpt["changed_files"], "fumpt passes but reports the file it rewrote")

	whitespace := byName["whitespace"]
	assert.Equal(t, checkResultFailed, whitespace["result"])
	assert.Equal(t, []any{"notes.md"}, whitespace["changed_files"])
	assert.Contains(t, whitespace["error"], "whitespace issues found")
	assert.IsType(t, float64(0), whitespace["duration_ms"])

	assert.Equal(t, runTotalsKeys(), keysOf(t, doc["totals"]))
	totals, ok := doc["totals"].(map[string]any)
	require.True(t, ok)
	assert.InDelta(t, 3, totals["checks"], 0)
	assert.InDelta(t, 2, totals["passed"], 0)
	assert.InDelta(t, 1, totals["failed"], 0)
	assert.InDelta(t, 0, totals["skipped"], 0)
	assert.InDelta(t, 3, totals["files"], 0)

	// The fixes really happened, and the human report went to stderr
	content, readErr := os.ReadFile(filepath.Join(repo, "notes.md")) // #nosec G304 - the test's own temporary file
	require.NoError(t, readErr)
	assert.Equal(t, "trailing\n", string(content))
	assert.Contains(t, stderr, "Check Results")
	assert.Contains(t, stderr, "whitespace failed")
}

func TestRunJSONReportsSuggestionAndOutput(t *testing.T) {
	jsonTestRepo(t, true, runJSONSettings)
	isolatePath(t, map[string]string{"gofumpt": testutil.FakeGofumptSyntaxError})
	require.NoError(t, os.WriteFile("broken.go", []byte("package\n"), 0o600))

	stdout, _, err := executeCLI(t, "run", "--json", "--only", "fumpt", "--files", "broken.go")

	require.ErrorIs(t, err, prerrors.ErrChecksFailed)
	doc := decodeSingleJSON(t, stdout)
	fumpt := checksByName(t, doc, map[string][]string{"fumpt": {"error", "output", "suggestion"}})["fumpt"]
	assert.Equal(t, checkResultFailed, fumpt["result"])
	assert.Contains(t, fumpt["error"], "gofumpt")
	assert.Contains(t, fumpt["suggestion"], "syntax errors")
	assert.Contains(t, fumpt["output"], "broken.go:1:8: syntax error")
	assert.Equal(t, []any{}, fumpt["changed_files"])
}

func TestRunJSONAllPassed(t *testing.T) {
	// A setting another test left in the process, which the legacy
	// configuration below would not override: the test must not see it
	t.Setenv("GO_PRE_COMMIT_ENABLE_FUMPT", "true")
	jsonTestRepo(t, false, builtinChecksOnly)
	require.NoError(t, os.WriteFile("clean.go", []byte("package main\n"), 0o600))
	testutil.RunGit(t, ".", "add", "clean.go")

	stdout, stderr, err := executeCLI(t, "run", "--json", "--quiet")

	require.NoError(t, err, "stderr: %s", stderr)
	doc := decodeSingleJSON(t, stdout)
	assert.Equal(t, runResultPassed, doc["result"])
	assert.Equal(t, []any{"clean.go"}, doc["files"], "staged files are checked by default")
	names := make([]any, 0, 2)
	for _, check := range objects(t, doc["checks"]) {
		names = append(names, check["name"])
		assert.Equal(t, checkResultPassed, check["result"])
	}
	assert.Equal(t, []any{"eof", "whitespace"}, names, "only the configured checks run")
}

func TestRunJSONNothingToRun(t *testing.T) {
	t.Run("disabled", func(t *testing.T) {
		jsonTestRepo(t, true, "ENABLE_GO_PRE_COMMIT=false\n")

		stdout, stderr, err := executeCLI(t, "run", "--json")

		require.NoError(t, err)
		doc := decodeSingleJSON(t, stdout)
		assert.Equal(t, runReportKeys(), keysOf(t, doc))
		assert.Equal(t, false, doc["enabled"])
		assert.Equal(t, runResultSkipped, doc["result"])
		assert.Equal(t, []any{}, doc["files"])
		assert.Equal(t, []any{}, doc["checks"])
		assert.Equal(t, []any{"Pre-commit system is disabled in configuration (ENABLE_GO_PRE_COMMIT=false)"}, doc["warnings"])
		assert.Contains(t, stderr, "disabled")
	})

	t.Run("no files", func(t *testing.T) {
		jsonTestRepo(t, true, builtinChecksOnly)

		stdout, _, err := executeCLI(t, "run", "--json")

		require.NoError(t, err)
		doc := decodeSingleJSON(t, stdout)
		assert.Equal(t, runReportKeys(), keysOf(t, doc))
		assert.Equal(t, true, doc["enabled"])
		assert.Equal(t, runResultSkipped, doc["result"])
		assert.Equal(t, []any{}, doc["files"])
		assert.Equal(t, []any{}, doc["checks"])
		assert.Equal(t, runTotalsKeys(), keysOf(t, doc["totals"]))
	})
}

func TestRunJSONErrors(t *testing.T) {
	t.Run("no configuration", func(t *testing.T) {
		isolateConfigEnv(t)
		preserveWorkingDir(t)
		t.Chdir(testutil.NewGitRepo(t))

		stdout, _, err := executeCLI(t, "run", "--json")
		require.ErrorIs(t, err, prerrors.ErrEnvFileNotFound)
		requireErrorReport(t, stdout, errCodeConfigNotFound)
	})

	t.Run("no check matches", func(t *testing.T) {
		jsonTestRepo(t, true, builtinChecksOnly)
		require.NoError(t, os.WriteFile("clean.go", []byte("package main\n"), 0o600))

		stdout, _, err := executeCLI(t, "run", "--json", "--only", "lint", "--files", "clean.go")
		require.ErrorIs(t, err, prerrors.ErrNoChecksToRun)
		requireErrorReport(t, stdout, errCodeNoChecksToRun)
	})

	t.Run("json cannot be combined with show-checks", func(t *testing.T) {
		jsonTestRepo(t, true, builtinChecksOnly)

		stdout, _, err := executeCLI(t, "run", "--json", "--show-checks")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "show-checks")
		assert.Empty(t, stdout)
	})
}

func TestRunWithoutJSONIsUnchanged(t *testing.T) {
	jsonTestRepo(t, true, builtinChecksOnly)
	require.NoError(t, os.WriteFile("notes.md", []byte("trailing   \n"), 0o600))

	stdout, stderr, err := executeCLI(t, "--no-color", "run", "--files", "notes.md")

	require.ErrorIs(t, err, prerrors.ErrChecksFailed)
	assert.NotContains(t, stdout, "{")
	assert.Contains(t, stdout, "Check Results")
	assert.Contains(t, stderr, "✗ whitespace failed")
}

// TestRunReportClassifiesResults tests how runner results map to the report,
// including a check skipped by --graceful, which the runner records as a
// success carrying the reason it could not run
func TestRunReportClassifiesResults(t *testing.T) {
	results := &runner.Results{
		CheckResults: []runner.CheckResult{
			{Name: "whitespace", Success: true, Duration: 2500 * time.Microsecond, Files: []string{"a.go"}, ChangedFiles: []string{}},
			{
				Name: "lint", Success: true, CanSkip: true, Duration: time.Millisecond, Files: []string{"a.go"},
				Error: "golangci-lint not found", Suggestion: "Install golangci-lint",
			},
			{
				Name: "fumpt", Success: false, Duration: 3 * time.Millisecond, Files: []string{"a.go"}, ChangedFiles: []string{},
				Error: "tool execution failed", Suggestion: "Fix syntax", Output: "\x1b[31ma.go:1:1: syntax error\x1b[0m",
			},
		},
		Passed:        2,
		Failed:        1,
		TotalDuration: 7 * time.Millisecond,
		TotalFiles:    1,
	}

	report := newRunReport(testVersion, &runRecord{enabled: true, files: []string{"a.go"}, results: results})

	assert.Equal(t, runResultFailed, report.Result)
	require.Len(t, report.Checks, 3)
	assert.Equal(t, []string{"fumpt", "lint", "whitespace"}, []string{report.Checks[0].Name, report.Checks[1].Name, report.Checks[2].Name})

	fumpt, lint, whitespace := report.Checks[0], report.Checks[1], report.Checks[2]
	assert.Equal(t, checkResultFailed, fumpt.Result)
	assert.Equal(t, "a.go:1:1: syntax error", fumpt.Output, "color codes are stripped")
	assert.Equal(t, checkResultSkipped, lint.Result)
	assert.Equal(t, "golangci-lint not found", lint.Error)
	assert.Equal(t, "Install golangci-lint", lint.Suggestion)
	assert.Nil(t, lint.ChangedFiles)
	assert.Equal(t, checkResultPassed, whitespace.Result)
	assert.Empty(t, whitespace.Error)
	assert.Equal(t, int64(2), whitespace.DurationMs)

	assert.Equal(t, runTotals{Checks: 3, Passed: 1, Failed: 1, Skipped: 1, Files: 1, DurationMs: 7}, report.Totals)
	assert.Equal(t, []string{"lint skipped: golangci-lint not found"}, report.Warnings)
}

func TestRouteHumanOutputToStderr(t *testing.T) {
	previous := tools.SetOutput(nil)
	t.Cleanup(func() { tools.SetOutput(previous) })

	restore := routeHumanOutputToStderr()
	assert.Equal(t, os.Stderr, tools.Output(), "tool installation messages go to stderr")

	restore()
	assert.Equal(t, os.Stdout, tools.Output())
}

func TestJSONHelpDocumentsTheValues(t *testing.T) {
	builder := NewCommandBuilder(NewCLIApp(testVersion, "test-commit", "test-date"))
	statusHelp := builder.BuildStatusCmd().Long
	runHelp := builder.BuildRunCmd().Long

	for _, state := range []string{hookStateReady, hookStateOutdated, hookStateNotExecutable, hookStateOther, hookStateMissing, hookStateError} {
		assert.Contains(t, statusHelp, state, "status help lists hook state %s", state)
	}
	for _, key := range append(append(statusReportKeys(), statusHookKeys()...), statusConfigKeys()...) {
		assert.Contains(t, statusHelp, key, "status help documents %s", key)
	}
	for _, key := range append(append(runReportKeys(), runCheckKeys()...), runTotalsKeys()...) {
		assert.Contains(t, runHelp, key, "run help documents %s", key)
	}
	for _, value := range []string{checkResultPassed, checkResultFailed, checkResultSkipped} {
		assert.Contains(t, runHelp, value)
	}
	for _, code := range []string{errCodeConfigNotFound, errCodeConfigInvalid, errCodeNotARepository} {
		assert.Contains(t, statusHelp, code)
		assert.Contains(t, runHelp, code)
	}
	for _, code := range []string{errCodeGitFailed, errCodeNoChecksToRun, errCodeRunFailed} {
		assert.Contains(t, runHelp, code)
	}
}

// TestErrorCodes tests the codes the JSON error document carries
func TestErrorCodes(t *testing.T) {
	coded := withCode(errCodeGitFailed, prerrors.ErrNotGitRepository)
	assert.Equal(t, prerrors.ErrNotGitRepository.Error(), coded.Error(), "the text of the error is unchanged")
	require.ErrorIs(t, coded, prerrors.ErrNotGitRepository)
	assert.Equal(t, errCodeGitFailed, errorCode(coded))
	assert.Equal(t, errCodeRunFailed, errorCode(errTestUncoded))

	assert.Equal(t, errCodeConfigNotFound, configErrorCode(prerrors.ErrEnvFileNotFound))
	assert.Equal(t, errCodeConfigInvalid, configErrorCode(errTestUncoded))
}

// errTestWrite is returned by failingWriter
var errTestWrite = errors.New("write refused")

// failingWriter refuses every write, like a closed pipe
type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) { return 0, errTestWrite }

// TestJSONWriteFailures tests that a failed write of the JSON document is
// reported, and never hides the error the command already had
func TestJSONWriteFailures(t *testing.T) {
	require.ErrorIs(t, writeJSON(failingWriter{}, map[string]string{}), errTestWrite)

	err := writeErrorReport(failingWriter{}, testVersion, withCode(errCodeNotARepository, prerrors.ErrNotGitRepository))
	require.ErrorIs(t, err, errTestWrite)
	require.ErrorIs(t, err, prerrors.ErrNotGitRepository)

	jsonTestRepo(t, true, "ENABLE_GO_PRE_COMMIT=false\n")
	builder := NewCommandBuilder(NewCLIApp(testVersion, "test-commit", "test-date"))
	runCmd := builder.BuildRunCmd()
	runCmd.SetOut(failingWriter{})
	require.ErrorIs(t, builder.runChecksWithConfig(RunConfig{JSON: true}, runCmd, nil), errTestWrite)
}

func TestStdoutOf(t *testing.T) {
	assert.Equal(t, os.Stdout, stdoutOf(nil))

	var buf bytes.Buffer
	runCmd := NewCommandBuilder(NewCLIApp(testVersion, "test-commit", "test-date")).BuildRunCmd()
	runCmd.SetOut(&buf)
	assert.Equal(t, &buf, stdoutOf(runCmd))
}

// TestRunJSONReportsToolInstallFailures tests that a tool that is missing and
// not installed is a failed (or, with --graceful, skipped) check in the report,
// with the install error as its output, and that the installer's messages,
// including the golangci-lint go install fallback, go to stderr, never to
// stdout. PATH holds only git (see jsonTestRepo), so the go install fallback
// cannot run, and the release install stops at its version check, before any
// download.
func TestRunJSONReportsToolInstallFailures(t *testing.T) {
	tests := []struct {
		name           string
		settings       string
		args           []string
		wantResult     string
		wantError      string
		wantSuggestion string
		wantOutput     []string
		wantStderr     []string
		wantWarning    string
	}{
		{
			name:           "automatic installs are off",
			settings:       "ENABLE_GO_PRE_COMMIT=true\nGO_PRE_COMMIT_AUTO_INSTALL_TOOLS=false\n",
			args:           []string{"--only", "fumpt"},
			wantResult:     checkResultFailed,
			wantError:      "gofumpt not found, and automatic installs are off (GO_PRE_COMMIT_AUTO_INSTALL_TOOLS=false)",
			wantSuggestion: "Install gofumpt with: go install mvdan.cc/gofumpt@",
			wantOutput:     []string{"GO_PRE_COMMIT_AUTO_INSTALL_TOOLS=false"},
		},
		{
			name:           "the release install and the go install fallback fail",
			settings:       "ENABLE_GO_PRE_COMMIT=true\nGO_PRE_COMMIT_GOLANGCI_LINT_VERSION=v2.15\n",
			args:           []string{"--only", "lint"},
			wantResult:     checkResultFailed,
			wantError:      "golangci-lint not found, and the automatic install did not make it available",
			wantSuggestion: "Install golangci-lint with: go install ",
			wantOutput:     []string{"golangci-lint v2.15: check version", "go install fallback"},
			wantStderr:     []string{"Installing golangci-lint@v2.15", "Trying go install ", "lint failed"},
		},
		{
			name:        "a missing linter skipped with --graceful",
			settings:    "ENABLE_GO_PRE_COMMIT=true\nGO_PRE_COMMIT_GOLANGCI_LINT_VERSION=v2.15\n",
			args:        []string{"--only", "lint", "--graceful"},
			wantResult:  checkResultSkipped,
			wantError:   "golangci-lint not found, and the automatic install did not make it available",
			wantStderr:  []string{"Trying go install "},
			wantWarning: "lint skipped: golangci-lint not found, and the automatic install did not make it available",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jsonTestRepo(t, true, tt.settings)
			require.NoError(t, os.WriteFile("go.mod", []byte("module example.com/app\n\ngo 1.22\n"), 0o600))
			require.NoError(t, os.WriteFile("main.go", []byte("package main\n\nfunc main() {}\n"), 0o600))

			stdout, stderr, err := executeCLI(t, append([]string{"run", "--json", "--files", "main.go"}, tt.args...)...)

			doc := decodeSingleJSON(t, stdout)
			require.Len(t, objects(t, doc["checks"]), 1)
			check := objects(t, doc["checks"])[0]
			assert.Equal(t, tt.wantResult, check["result"])
			assert.Equal(t, tt.wantError, check["error"])
			if tt.wantSuggestion != "" {
				assert.Contains(t, check["suggestion"], tt.wantSuggestion)
			}
			for _, want := range tt.wantOutput {
				assert.Contains(t, check["output"], want)
			}
			for _, want := range tt.wantStderr {
				assert.Contains(t, stderr, want)
			}

			if tt.wantResult == checkResultSkipped {
				require.NoError(t, err)
				assert.Equal(t, []any{tt.wantWarning}, doc["warnings"])
				return
			}
			require.ErrorIs(t, err, prerrors.ErrChecksFailed)
		})
	}
}
