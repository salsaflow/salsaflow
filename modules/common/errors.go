package common

import (
	// Stdlib
	"errors"
)

// ErrNotStageable shall be returned from EnsureStageable()
// when the given release cannot be staged yet.
var ErrNotStageable = errors.New("release cannot be staged")

// ErrNotReleasable shall be returned from EnsureReleasable()
// when the given release cannot be released yet.
var ErrNotReleasable = errors.New("release cannot be released")
