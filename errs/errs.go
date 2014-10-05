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
	defer logger.Unlock()
	logger.UnsafeFail(err.TaskName)
	logger.UnsafeNewLine(fmt.Sprintf("(%v)", err.Err))
	logger.UnsafeStderr(err.Stderr)
}

func (err *Error) Fatal(logger log.Logger) {
	logger.Lock()
	defer logger.Unlock()
	logger.UnsafeFail(err.TaskName)
	logger.UnsafeNewLine(fmt.Sprintf("(%v)", err.Err))
	logger.UnsafeStderr(err.Stderr)
	logger.UnsafeFatalln("\nError: task failed")
}

func (err *Error) Error() string {
	return err.Err.Error()
}

func LogAndReturn(err error) error {
	if ex, ok := err.(*Error); ok {
		ex.Log(log.V(log.Info))
	}
	return err
}
