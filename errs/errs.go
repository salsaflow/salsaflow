package errs

import (
	// Stdlib
	"bytes"
	"fmt"

	// Internal
	"github.com/salsaflow/salsaflow/log"
)

type Err struct {
	taskName string
	err      error
	errHint  *bytes.Buffer
}

func NewError(taskName string, err error, errHint *bytes.Buffer) *Err {
	// The task name and the error must be set, always. Only the error hint is optional.
	switch {
	case taskName == "":
		panic("errs.NewError: argument 'taskName' is empty")
	case err == nil:
		panic("errs.NewError: argument 'err' is empty")
	}

	// We are cool now, return the new Err instance.
	return &Err{taskName, err, errHint}
}

func (err *Err) Log(logger log.Logger) *Err {
	logger.Lock()
	defer logger.Unlock()
	return err.unsafeLog(logger)
}

func (err *Err) LogAndDie(logger log.Logger) {
	logger.Lock()
	defer logger.Unlock()
	err.unsafeLog(logger)
	logger.Fatalln("\nFatal error: " + err.Error())
}

func (err *Err) unsafeLog(logger log.Logger) *Err {
	// Check whether err.err is also an Err.
	// In case it is, print that error first so that format the output
	// in a similar way to an unrolling stack, i.e. deeper error first.
	if err.err != nil {
		if next, ok := err.err.(*Err); ok && next != nil {
			next.unsafeLog(logger)
		}
	}

	// Print the error saved in this Err struct.
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

// RootCause returns the deepest error, in other words, the error that started the error chain.
func (err *Err) RootCause() error {
	if next, ok := err.err.(*Err); ok && next != nil {
		return next.RootCause()
	}
	return err.err
}

func (err *Err) Error() string {
	return err.err.Error()
}

func Log(err error) error {
	if ex, ok := err.(*Err); ok {
		return ex.Log(log.V(log.Info))
	} else {
		return NewError("unknown task", err, nil).Log(log.V(log.Info))
	}
}

func LogError(taskName string, err error, errHint *bytes.Buffer) {
	Log(NewError(taskName, err, errHint))
}

func Fatal(err error) {
	if ex, ok := err.(*Err); ok {
		ex.Log(log.V(log.Info))
	}
	log.Fatalln("\nFatal error: " + err.Error())
}

func RootCause(err error) error {
	ex, ok := err.(*Err)
	if ok {
		return ex.RootCause()
	}
	return err
}
