package common

import (
	// Internal
	"github.com/salsaflow/salsaflow/errs"
)

// ErrNotStageable shall be returned from EnsureStageable()
// when the given release cannot be staged yet.
type ErrNotStageable struct {
	errs.Err
}

// ErrNotReleasable shall be returned from EnsureReleasable()
// when the given release cannot be released yet.
type ErrNotReleasable struct {
	errs.Err
}
