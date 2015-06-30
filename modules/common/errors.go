package common

import (
	// Stdlib
	"errors"
	"fmt"

	// Internal
	"github.com/salsaflow/salsaflow/version"
)

// ErrNotStageable shall be returned from EnsureStageable()
// when the given release cannot be staged yet.
var ErrNotStageable = errors.New("release cannot be staged")

// ErrNotReleasable shall be returned from EnsureReleasable()
// when the given release cannot be released yet.
var ErrNotReleasable = errors.New("release cannot be released")

// ErrReleaseNotFound shall be returned from GenerateReleaseNotes
// or perhaps any other function when the given release was not found.
type ErrReleaseNotFound struct {
	Version *version.Version
}

func (err *ErrReleaseNotFound) Error() string {
	return fmt.Sprintf("release %v not found", err.Version.BaseString())
}
