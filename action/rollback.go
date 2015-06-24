package action

import (
	// Stdlib
	"errors"

	// Internal
	"github.com/salsaflow/salsaflow/errs"
	"github.com/salsaflow/salsaflow/log"
)

var ErrRollbackFailed = errs.NewError(
	"Roll back changes", errors.New("failed to roll back changes"))

// RollbackOnError is equivalent to RollbackTaskOnError(err, "", action).
func RollbackOnError(err *error, action Action) {
	RollbackTaskOnError(err, "", action)
}

// RollbackTaskOnError wraps an ActionChain to perform a rollback on error.
func RollbackTaskOnError(err *error, task string, action Action) {
	chain := NewActionChain()
	chain.PushTask(task, action)
	chain.RollbackOnError(err)
}

type actionRecord struct {
	task   string
	action Action
}

type ActionChain struct {
	actions []*actionRecord
}

func NewActionChain() *ActionChain {
	return &ActionChain{}
}

func (chain *ActionChain) Push(action Action) {
	chain.PushTask("", action)
}

func (chain *ActionChain) PushTask(task string, action Action) {
	if action != nil {
		chain.actions = append(chain.actions, &actionRecord{task, action})
	}
}

func (chain *ActionChain) Rollback() error {
	var ex error
	for i := range chain.actions {
		act := chain.actions[len(chain.actions)-1-i]

		// Inform the user what is happening.
		if task := act.task; task != "" {
			log.Rollback(task)
		}

		// Run the rollback function registered.
		if err := act.action.Rollback(); err != nil {
			errs.Log(err)
			ex = ErrRollbackFailed
		}
	}
	return ex
}

// RollbackOnError is supposed to be called using defer:
//
//     defer chain.RollbackOnError(&err)
//
// Then it is visible why *error is being passed in. The args are bound
// when this line is encountered, so it would not be set to a non-nil error
// in case the error is set later. That is why we pass a pointer so that
// err can be checked when the deferred function is actually called.
func (chain *ActionChain) RollbackOnError(err *error) {
	// Run the rollback function in case there is an error.
	if *err != nil {
		chain.Rollback()
	}
}
