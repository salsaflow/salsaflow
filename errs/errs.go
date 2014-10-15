package errs

import (
	// Stdlib
	"bytes"
	"fmt"

	// Internal
	"github.com/salsita/salsaflow/log"
)

type Error struct {
	TaskName string
	Stderr   *bytes.Buffer
	Err      error
}

func NewError(taskName string, stderr *bytes.Buffer, err error) *Error {
	return &Error{taskName, stderr, err}
}

func (err *Error) Log(logger log.Logger) {
	logger.Lock()
	logger.UnsafeFail(err.TaskName)
	if err.Err != nil {
		logger.UnsafeNewLine(fmt.Sprintf("(%v)", err.Err))
	}
	if err.Stderr != nil {
		logger.UnsafeStderr(err.Stderr)
	}
	logger.Unlock()
	// Call myself recursively in case err.Err is also an Error.
	if ex, ok := err.Err.(*Error); ok {
		ex.Log(logger)
	}
}

func (err *Error) Fatal(logger log.Logger) {
	logger.Lock()
	defer logger.Unlock()
	logger.UnsafeFail(err.TaskName)
	if err.Err != nil {
		logger.UnsafeNewLine(fmt.Sprintf("(%v)", err.Err))
	}
	if err.Stderr != nil {
		logger.UnsafeStderr(err.Stderr)
	}
	if err.Err != nil {
		logger.UnsafeFatalln("\nError: " + err.Err.Error())
	} else {
		logger.UnsafeFatalln("\nError: task failed")
	}
}

func (err *Error) Error() string {
	return err.Err.Error()
}

func Log(err error) error {
	if ex, ok := err.(*Error); ok {
		ex.Log(log.V(log.Info))
	}
	return err
}

func LogFail(task string, err error) error {
	if ex, ok := err.(*Error); ok {
		ex.Log(log.V(log.Info))
		if task != ex.TaskName {
			log.Fail(task)
		}
	} else {
		logger := log.V(log.Info)
		logger.Lock()
		logger.Fail(task)
		logger.NewLine(fmt.Sprintf("(err = %v)", err))
		logger.Unlock()
	}
	return err
}
