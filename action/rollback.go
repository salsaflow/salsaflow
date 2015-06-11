package action

import (
	// Internal
	"github.com/salsaflow/salsaflow/errs"
	"github.com/salsaflow/salsaflow/log"
)

// RollbackOnError is supposed to be used together with defer:
//
//     defer action.RollbackOnError(&err, action)
//
// Then it is visible why *error is being passed in. The args are bound
// when this line is encountered, so it would not be set to a non-nil error
// in case the error is set later. That is why we pass a pointer so that
// err can be checked when the deferred function is actually called.
func RollbackOnError(err *error, action Action) {
	RollbackTaskOnError(err, "", action)
}

// RollbackTaskOnError does the same thing as RollbackOnError,
// but it also informs the user that the given task is being reverted.
func RollbackTaskOnError(err *error, task string, action Action) {
	// Do nothing unless there is an error.
	if *err == nil {
		return
	}

	// Tell the user what is happening.
	if task != "" {
		log.Rollback(task)
	}

	// Call the rollback function.
	if ex := action.Rollback(); ex != nil {
		errs.Log(ex)
	}
}
