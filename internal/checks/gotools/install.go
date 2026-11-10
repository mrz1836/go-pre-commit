package gotools

import (
	"errors"
	"fmt"

	prerrors "github.com/mrz1836/go-pre-commit/internal/errors"
	"github.com/mrz1836/go-pre-commit/internal/tools"
)

// toolInstallError reports a tool that is not installed and that
// tools.EnsureInstalled did not install, either because automatic installs
// are off or because the install failed. The install error passes through as
// Output, so its cause (the tool, version, and failed step) is never hidden,
// and the suggestion is the exact command that installs the configured
// version.
func toolInstallError(tool string, err error) *prerrors.CheckError {
	message := fmt.Sprintf("%s not found, and the automatic install did not make it available", tool)
	if errors.Is(err, tools.ErrAutoInstallDisabled) {
		message = fmt.Sprintf("%s not found, and automatic installs are off (%s=false)", tool, tools.EnvAutoInstallTools)
	}
	return &prerrors.CheckError{
		Err:        fmt.Errorf("%w: %w", prerrors.ErrToolNotFound, err),
		Message:    message,
		Suggestion: fmt.Sprintf("Install %s with: %s", tool, tools.InstallCommand(tool)),
		Output:     err.Error(),
	}
}
