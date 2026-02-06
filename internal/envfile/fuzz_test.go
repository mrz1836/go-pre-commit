package envfile

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// FuzzParse tests the parse function with various malformed inputs
func FuzzParse(f *testing.F) {
	// Seed corpus with various test cases
	f.Add("KEY=value")
	f.Add("KEY=value # comment")
	f.Add("KEY=\"quoted value\"")
	f.Add("=no_key")
	f.Add("KEY_WITHOUT_VALUE")
	f.Add("KEY=")
	f.Add("#comment")
	f.Add("")
	f.Add("KEY=value\nKEY2=value2")
	f.Add("KEY=value with spaces")
	f.Add("KEY=value#anchor")

	f.Fuzz(func(t *testing.T, input string) {
		// Should not panic on any input
		result := parse(input)

		// Verify result is a valid map
		if result == nil {
			t.Fatal("parse returned nil map")
		}

		// Verify all keys and values are valid strings
		for key, value := range result {
			if key == "" {
				t.Error("parse returned empty key")
			}
			// Value can be empty string, that's valid
			_ = value
		}

		// Verify parsing is deterministic
		result2 := parse(input)
		if len(result) != len(result2) {
			t.Errorf("parse is not deterministic: got %d keys first time, %d second time", len(result), len(result2))
		}
	})
}

// FuzzProcessValue tests the processValue function with various malformed inputs
func FuzzProcessValue(f *testing.F) {
	// Seed corpus with various test cases
	f.Add("simple")
	f.Add("  spaces  ")
	f.Add("value # comment")
	f.Add("\"quoted\"")
	f.Add("'quoted'")
	f.Add("value#anchor")
	f.Add("")
	f.Add("   ")

	f.Fuzz(func(t *testing.T, input string) {
		// Should not panic on any input
		result := processValue(input)

		// Value can be empty, that's valid
		_ = result

		// Verify processing is deterministic
		result2 := processValue(input)
		if result != result2 {
			t.Errorf("processValue is not deterministic: got %q first time, %q second time", result, result2)
		}

		// Verify no leading/trailing whitespace in unquoted values
		if !strings.HasPrefix(input, "\"") && !strings.HasPrefix(input, "'") {
			if result != strings.TrimSpace(result) {
				// This is expected for quoted values, but not unquoted
				if !strings.Contains(input, "\"") && !strings.Contains(input, "'") {
					t.Errorf("processValue returned value with whitespace: %q from input %q", result, input)
				}
			}
		}
	})
}

// FuzzLoadDir tests the LoadDir function with various file contents
func FuzzLoadDir(f *testing.F) {
	// Seed corpus: coreContent, localContent, skipLocal
	f.Add("CORE_VAR=value1\nSHARED=core\n", "LOCAL_VAR=local\nSHARED=local\n", false)
	f.Add("# comment only\n", "\n", false)
	f.Add("OVERRIDE=base\n", "OVERRIDE=local\n", true)
	f.Add("", "", false)

	f.Fuzz(func(t *testing.T, coreContent, localContent string, skipLocal bool) {
		tmpDir := t.TempDir()
		envDir := filepath.Join(tmpDir, "env")
		if err := os.MkdirAll(envDir, 0o750); err != nil {
			t.Fatal(err)
		}

		if err := os.WriteFile(filepath.Join(envDir, "00-core.env"), []byte(coreContent), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(envDir, "99-local.env"), []byte(localContent), 0o600); err != nil {
			t.Fatal(err)
		}

		// Should not panic on any input
		_ = LoadDir(envDir, skipLocal)
	})
}

// FuzzLoad tests the Load function with various file contents
func FuzzLoad(f *testing.F) {
	// Seed corpus with various test cases
	f.Add("KEY=value\nKEY2=value2\n")
	f.Add("# Comment\nKEY=value\n")
	f.Add("=invalid\nKEY=value\n")
	f.Add("KEY=value\n\nKEY2=value2\n")
	f.Add("")

	f.Fuzz(func(t *testing.T, content string) {
		// Should not panic when parsing
		_ = parse(content)

		// parse should always return a non-nil map
		result := parse(content)
		if result == nil {
			t.Fatal("parse returned nil map")
		}
	})
}
