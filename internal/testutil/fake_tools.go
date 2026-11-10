package testutil

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// IsolatedPath replaces PATH with a new directory holding only git (as a
// symlink) and the given executable scripts, keyed by command name, and
// returns that directory. A check then finds the fake tools, and no
// installer (go, curl, sh) can be found, so a test can never install a real
// tool or reach the network. It skips the test where symlinks are not
// supported. Callers that run checks should also clear the tools package's
// install cache (tools.CleanCache), which remembers earlier lookups.
func IsolatedPath(tb testing.TB, scripts map[string]string) string {
	tb.Helper()
	gitPath, err := exec.LookPath("git")
	if err != nil {
		tb.Skip("git not available")
	}

	binDir := tb.TempDir()
	if err := os.Symlink(gitPath, filepath.Join(binDir, filepath.Base(gitPath))); err != nil {
		tb.Skipf("symlinks not supported: %v", err)
	}
	for name, script := range scripts {
		if err := os.WriteFile(filepath.Join(binDir, name), []byte(script), 0o700); err != nil { // #nosec G306 - a fake tool must be executable
			tb.Fatalf("failed to write fake %s: %v", name, err)
		}
	}
	tb.Setenv("PATH", binDir)
	return binDir
}

// FakeGofumpt is a stand-in for gofumpt, for IsolatedPath. Like gofumpt, it
// lists (-l) and rewrites (-w) the .go files it would reformat, which are the
// ones containing the word "messy". It uses only shell builtins, so it runs on
// a PATH that holds nothing else.
const FakeGofumpt = `#!/bin/sh
list=false
write=false
for arg in "$@"; do
	case "$arg" in
	-l) list=true ;;
	-w) write=true ;;
	esac
done
for arg in "$@"; do
	case "$arg" in
	*.go)
		messy=false
		while IFS= read -r line || [ -n "$line" ]; do
			case "$line" in *messy*) messy=true ;; esac
		done < "$arg"
		if [ "$messy" = true ]; then
			if [ "$list" = true ]; then echo "$arg"; fi
			if [ "$write" = true ]; then printf 'package main\n' > "$arg"; fi
		fi
		;;
	esac
done
`

// FakeGofumptSyntaxError is a stand-in for gofumpt, for IsolatedPath, that
// fails the way gofumpt does on a file it cannot parse
const FakeGofumptSyntaxError = `#!/bin/sh
echo "broken.go:1:8: syntax error: unexpected newline" >&2
exit 2
`
