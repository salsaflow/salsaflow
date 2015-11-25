package noop

import (
	// Internal
	"github.com/salsaflow/salsaflow/action"
)

type release struct{}

func (r *release) Initialise() (action.Action, error) {
	return action.Noop, nil
}

func (r *release) EnsureClosable() error {
	return nil
}

func (r *release) Close() (action.Action, error) {
	return action.Noop, nil
}
