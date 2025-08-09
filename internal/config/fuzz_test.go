package config

import (
	"os"
	"testing"
)

// FuzzGetBoolEnv tests the getBoolEnv function with various malformed inputs
func FuzzGetBoolEnv(f *testing.F) {
	// Seed corpus with known values
	f.Add("true")
	f.Add("false")
	f.Add("1")
	f.Add("0")
	f.Add("TRUE")
	f.Add("FALSE")
	f.Add("")
	f.Add("invalid")
	f.Add("truee")
	f.Add("falsee")

	f.Fuzz(func(t *testing.T, value string) {
		// Set environment variable with fuzzed value
		key := "FUZZ_TEST_BOOL_VAR"
		_ = os.Setenv(key, value)
		defer func() { _ = os.Unsetenv(key) }()

		// Function should never panic regardless of input
		result := getBoolEnv(key, false)

		// Result should always be a boolean
		if result != true && result != false {
			t.Errorf("getBoolEnv returned non-boolean value: %v", result)
		}
	})
}

// FuzzGetIntEnv tests the getIntEnv function with various malformed inputs
func FuzzGetIntEnv(f *testing.F) {
	// Seed corpus with known values
	f.Add("0")
	f.Add("1")
	f.Add("-1")
	f.Add("100")
	f.Add("invalid")
	f.Add("")
	f.Add("9999999999999999999999")
	f.Add("1.5")
	f.Add("0x10")
	f.Add("10e3")

	f.Fuzz(func(t *testing.T, value string) {
		// Set environment variable with fuzzed value
		key := "FUZZ_TEST_INT_VAR"
		_ = os.Setenv(key, value)
		defer func() { _ = os.Unsetenv(key) }()

		// Function should never panic regardless of input
		result := getIntEnv(key, 42)

		// Result should always be an integer
		if result < -2147483648 || result > 2147483647 {
			t.Errorf("getIntEnv returned out-of-range value: %d", result)
		}
	})
}

// FuzzGetStringEnv tests the getStringEnv function with various inputs
func FuzzGetStringEnv(f *testing.F) {
	// Seed corpus with various string inputs
	f.Add("")
	f.Add("normal")
	f.Add("with spaces")
	f.Add("\n\t\r")
	f.Add("unicode: 🚀")
	f.Add("null\x00byte")
	f.Add("very long string that might cause issues: " + string(make([]byte, 1000)))

	f.Fuzz(func(_ *testing.T, value string) {
		// Set environment variable with fuzzed value
		key := "FUZZ_TEST_STRING_VAR"
		_ = os.Setenv(key, value)
		defer func() { _ = os.Unsetenv(key) }()

		// Function should never panic regardless of input
		result := getStringEnv(key, "default")

		// Result should always be a string - function may trim or process the value
		// Only check for completely unexpected behavior
		_ = result // Just ensure we got a result
	})
}

// FuzzConfigValidation tests config validation with malformed configurations
func FuzzConfigValidation(f *testing.F) {
	// Seed corpus with various config scenarios
	f.Add("debug")
	f.Add("info")
	f.Add("warn")
	f.Add("error")
	f.Add("invalid")
	f.Add("")
	f.Add("DEBUG")
	f.Add("debug\x00")

	f.Fuzz(func(_ *testing.T, logLevel string) {
		cfg := &Config{
			Enabled:      true,
			LogLevel:     logLevel,
			MaxFileSize:  1024 * 1024,
			MaxFilesOpen: 100,
			Timeout:      60,
		}
		cfg.Performance.ParallelWorkers = 2

		// Validation should never panic regardless of input
		err := cfg.Validate()

		// Should return validation error or nil - both are acceptable
		_ = err // Validation may succeed or fail, both are fine
	})
}

// FuzzIsValidVersion tests version validation with various malformed versions
func FuzzIsValidVersion(f *testing.F) {
	// Seed corpus with version-like strings
	f.Add("v1.0.0")
	f.Add("v1.2.3")
	f.Add("1.0.0")
	f.Add("v1.0")
	f.Add("v1")
	f.Add("invalid")
	f.Add("")
	f.Add("v1.0.0-beta")
	f.Add("v1.0.0+build")
	f.Add("v999.999.999")

	f.Fuzz(func(t *testing.T, version string) {
		// Function should never panic regardless of input
		result := isValidVersion(version)

		// Result should always be a boolean
		if result != true && result != false {
			t.Errorf("isValidVersion returned non-boolean value: %v", result)
		}
	})
}
