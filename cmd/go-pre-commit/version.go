package main

// Build-time variables injected via ldflags
//
//nolint:gochecknoglobals // These are build-time injected variables
var (
	Version   = "dev"
	Commit    = "none"
	BuildDate = "unknown"
)
