package errs

import (
	// Stdlib
	"bytes"
	"fmt"

	// Internal
	"github.com/salsita/salsaflow/log"
)

type Error struct {
	taskName string
	err      error
	errHint  *bytes.Buffer
}

func NewError(taskName string, err error, errHint *bytes.Buffer) *Error {
	// The task name and the error must be set, always. Only the error hint is optional.
	switch {
	case taskName == "":
		panic("errs.NewError: argument 'taskName' is empty")
	case err == nil:
		panic("errs.NewError: argument 'err' is empty")
	}

	// We are cool now, return the new Error instance.
	return &Error{taskName, err, errHint}
}

func (err *Error) Log(logger log.Logger) *Error {
	logger.Lock()
	defer logger.Unlock()
	return err.unsafeLog(logger)
}

func (err *Error) LogAndDie(logger log.Logger) {
	logger.Lock()
	defer logger.Unlock()
	err.unsafeLog(logger)
	logger.Fatalln("\nFatal error: " + err.Error())
}

func (err *Error) unsafeLog(logger log.Logger) *Error {
	// Check whether err.err is also an Error.
	// In case it is, print that error first so that format the output
	// in a similar way to an unrolling stack, i.e. deeper error first.
	if err.err != nil {
		if next, ok := err.err.(*Error); ok && next != nil {
			next.Log(logger)
		}
	}

	// Print the error saved in this Error struct.
	logger.UnsafeFail(err.taskName)
	if err.err != nil {
		logger.UnsafeNewLine(fmt.Sprintf("(error = %v)", err.err))
	}
	if err.errHint != nil {
		logger.UnsafeStderr(err.errHint)
	}

	// Return self to be able to chain calls or return.
	return err
}

// Trigger returns the deepest error, in other words, the error that started the error chain.
func (err *Error) Trigger() error {
	if next, ok := err.err.(*Error); ok && next != nil {
		return next.Trigger()
	}
	return err.err
}

func (err *Error) Error() string {
	return err.err.Error()
}

func Log(err error) error {
	if ex, ok := err.(*Error); ok {
		return ex.Log(log.V(log.Info))
	} else {
		return NewError("unknown task", err, nil).Log(log.V(log.Info))
	}
}

func LogError(taskName string, err error, errHint *bytes.Buffer) {
	Log(NewError(taskName, err, errHint))
}

func Fatal(err error) {
	if ex, ok := err.(*Error); ok {
		ex.Log(log.V(log.Info))
	}
	log.Fatalln("\nFatal error: " + err.Error())
}
