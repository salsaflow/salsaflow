package common

import (
	// Internal
	"github.com/salsaflow/salsaflow/errs"
)

type ErrNotReleasable struct {
	*errs.Err
}
