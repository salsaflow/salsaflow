package fileutil

import (
	// Stdlib
	"errors"
	"fmt"
	"os"

	// Internal
	"github.com/salsaflow/salsaflow/action"
	"github.com/salsaflow/salsaflow/errs"
	"github.com/salsaflow/salsaflow/log"
)

func EnsureDirectoryExists(path string) (action.Action, error) {
	// Check whether the directory exists already.
	task := fmt.Sprintf("Check whether '%v' exists and is a directory", path)
	info, err := os.Stat(path)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, errs.NewError(task, err)
		}
	} else {
		// In case the path exists, make sure it is a directory.
		if !info.IsDir() {
			return nil, errs.NewError(task, errors.New("not a directory: "+path))
		}
		// We are done.
		return action.Noop, nil
	}

	// Now we know that path does not exist, so we need to create it.
	createTask := fmt.Sprintf("Create directory '%v'", path)
	log.Run(createTask)
	if err := os.MkdirAll(path, 0755); err != nil {
		return nil, errs.NewError(createTask, err)
	}

	return action.ActionFunc(func() error {
		log.Rollback(createTask)
		task := fmt.Sprintf("Remove directory '%v'", path)
		if err := os.RemoveAll(path); err != nil {
			return errs.NewError(task, err)
		}
		return nil
	}), nil
}
