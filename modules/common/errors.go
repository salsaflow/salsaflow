package common

import (
	// Internal
	"github.com/salsaflow/salsaflow/errs"
)

type ErrNotReleasable struct {
	// Cannot simply embed because then the name collides
	// with the Error method. Sucks.
	E errs.Error
}

func (err *ErrNotReleasable) Error() string {
	return err.E.Error()
}

func (err *ErrNotReleasable) Hint() string {
	return err.E.Hint()
}

func (err *ErrNotReleasable) Task() string {
	return err.E.Task()
}

func (err *ErrNotReleasable) Err() error {
	return err.E.Err()
}
