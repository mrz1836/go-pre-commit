package integration

import (
	"fmt"
	"os"
	"testing"

	"github.com/mrz1836/go-pre-commit/internal/testutil"
)

// TestMain clears go-pre-commit configuration inherited from the environment
// (for example exported by magex from .github/env) so tests only see the
// configuration they set up themselves.
func TestMain(m *testing.M) {
	if err := testutil.ClearConfigEnv(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to clear inherited configuration: %v\n", err)
		os.Exit(1)
	}
	os.Exit(m.Run())
}
