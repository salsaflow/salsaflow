package errs

import (
	// Stdlib
	"fmt"

	// Internal
	"github.com/salsaflow/salsaflow/log"
)

type Error interface {
	error
	Task() string
	Err() error
	Hint() string
}

type Err struct {
	task string
	err  error
	hint string
}

func NewError(task string, err error) Error {
	return NewErrorWithHint(task, err, "")
}

func NewErrorWithHint(task string, err error, hint string) Error {
	// The task name and the error must be set, always. Only the error hint is optional.
	switch {
	case task == "":
		panic("errs.NewErrorWithHint: argument 'task' is empty")
	case err == nil:
		panic("errs.NewErrorWithHint: argument 'err' is empty")
	}

	// We are cool now, return a new Err instance.
	return &Err{
		task: task,
		err:  err,
		hint: hint,
	}
}

func (err *Err) Error() string {
	return err.err.Error()
}

func (err *Err) Hint() string {
	return err.hint
}

func (err *Err) Task() string {
	return err.task
}

func (err *Err) Err() error {
	return err.err
}

// LogWith logs the given error using the given logger.
func LogWith(err error, logger log.Logger) error {
	logger.Lock()
	defer logger.Unlock()
	return unsafeLogWith(err, logger)
}

func unsafeLogWith(err error, logger log.Logger) error {
	// Implementing Error inteface.
	if ex, ok := err.(Error); ok {
		logger.UnsafeFail(ex.Task())

		hint := ex.Hint()
		if next, ok := ex.Err().(Error); ok {
			logger.UnsafeStderr(hint)
			return unsafeLogWith(next, logger)
		} else {
			last := ex.Err()
			logger.UnsafeNewLine(fmt.Sprintf("(error = %v)", last))
			logger.UnsafeStderr(hint)
			return last
		}
	}

	// Regular errors.
	return err
}

// Log just calls LogWith(err, log.V(log.Info)).
func Log(err error) error {
	return LogWith(err, log.V(log.Info))
}

// LogError is a wrapper that calls Log(NewError(task, err)).
func LogError(task string, err error) error {
	return Log(NewError(task, err))
}

// LogErrorWithHint is a wrapper that calls Log(NewErrorWithHint(task, err, hint)).
func LogErrorWithHint(task string, err error, hint string) error {
	return Log(NewErrorWithHint(task, err, hint))
}

// Fatal logs the error and exists the program with os.Exit(1).
func Fatal(err error) {
	Log(err)
	log.Fatalln("\nFatal error: " + err.Error())
}

// RootCause returns the error deepest in the error chain.
func RootCause(err error) error {
	if ex, ok := err.(Error); ok {
		return RootCause(ex.Err())
	}
	return err
}
