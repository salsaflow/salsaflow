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

// ErrNotClosable shall be returned from EnsureReleasable()
// when the given release cannot be released yet.
var ErrNotClosable = errors.New("release cannot be closed")

// ErrReleaseNotFound shall be returned from GenerateReleaseNotes
// or perhaps any other function when the given release was not found.
type ErrReleaseNotFound struct {
	Version *version.Version
}

func (err *ErrReleaseNotFound) Error() string {
	return fmt.Sprintf("release %v not found", err.Version.BaseString())
}
