package github

import (
	// Stdlib
	"fmt"

	// Internal
	"github.com/salsaflow/salsaflow/version"
)

type ErrMilestoneNotFound struct {
	v *version.Version
}

func (err *ErrMilestoneNotFound) Error() string {
	return fmt.Sprintf("review milestone not found for release %v", err.v)
}
