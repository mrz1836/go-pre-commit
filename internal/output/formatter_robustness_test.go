package output

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestFormatExecutionStats_ColorEnabled(t *testing.T) {
	f := New(Options{ColorEnabled: true})

	// All three count branches plus the file-count branch are exercised.
	out := f.FormatExecutionStats(3, 2, 1, 1500*time.Millisecond, 7)
	assert.Contains(t, out, "passed")
	assert.Contains(t, out, "failed")
	assert.Contains(t, out, "skipped")
	assert.Contains(t, out, "7 file(s)")
}

func TestFormatExecutionStats_EdgeCases(t *testing.T) {
	f := New(Options{ColorEnabled: false})

	t.Run("all zero with zero duration", func(t *testing.T) {
		out := f.FormatExecutionStats(0, 0, 0, 0, 0)
		assert.Contains(t, out, "in ")
		assert.NotContains(t, out, "file(s)")
	})

	t.Run("very large counts", func(t *testing.T) {
		out := f.FormatExecutionStats(999999, 888888, 777777, time.Hour, 1234567)
		assert.Contains(t, out, "999999 passed")
		assert.Contains(t, out, "888888 failed")
		assert.Contains(t, out, "777777 skipped")
		assert.Contains(t, out, "1234567 file(s)")
	})

	t.Run("only failures, no files", func(t *testing.T) {
		out := f.FormatExecutionStats(0, 4, 0, time.Second, 0)
		assert.Contains(t, out, "4 failed in ")
		assert.NotContains(t, out, "passed")
		assert.NotContains(t, out, "file(s)")
	})
}
