package cmd

import "testing"

// preserveWorkingDir restores the process working directory when the test
// ends. Running the root command in-process calls initConfig, which changes
// directory to the nearest parent holding .github configuration (the
// repository root); without this, later tests (and -count iterations) would
// run from that directory instead of the package directory.
func preserveWorkingDir(t *testing.T) {
	t.Helper()
	t.Chdir(".")
}
