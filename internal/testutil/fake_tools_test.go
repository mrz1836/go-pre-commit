package testutil

import (
	"os"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsolatedPath_HoldsOnlyGitAndTheScripts(t *testing.T) {
	binDir := IsolatedPath(t, map[string]string{"fake-tool": "#!/bin/sh\necho fake\n"})

	assert.Equal(t, binDir, os.Getenv("PATH"))
	entries, err := os.ReadDir(binDir)
	require.NoError(t, err)
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		names = append(names, entry.Name())
	}
	assert.ElementsMatch(t, []string{"git", "fake-tool"}, names)

	_, err = exec.LookPath("go")
	require.Error(t, err, "no installer may be reachable")
	output, err := exec.CommandContext(t.Context(), "fake-tool").Output()
	require.NoError(t, err)
	assert.Equal(t, "fake\n", string(output))
}
