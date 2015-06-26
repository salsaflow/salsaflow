package errs

import (
	// Stdlib
	"fmt"

	// Internal
	"github.com/salsaflow/salsaflow/log"
)

type Err interface {
	error
	Task() string
	Err() error
	Hint() string
}

type errImpl struct {
	task string
	err  error
	hint string
}

func NewError(task string, err error) Err {
	return NewErrorWithHint(task, err, "")
}

func NewErrorWithHint(task string, err error, hint string) Err {
	// The task name and the error must be set, always. Only the error hint is optional.
	switch {
	case task == "":
		panic("errs.NewErrorWithHint: argument 'task' is empty")
	case err == nil:
		panic("errs.NewErrorWithHint: argument 'err' is empty")
	}

	// We are cool now, return a new Err instance.
	return &errImpl{
		task: task,
		err:  err,
		hint: hint,
	}
}

func (err *errImpl) Error() string {
	return err.err.Error()
}

func (err *errImpl) Hint() string {
	return err.hint
}

func (err *errImpl) Task() string {
	return err.task
}

func (err *errImpl) Err() error {
	return err.err
}

// LogWith logs the given error using the given logger.
func LogWith(err error, logger log.Logger) error {
	logger.Lock()
	defer logger.Unlock()
	return unsafeLogWith(err, logger)
}

func unsafeLogWith(err error, logger log.Logger) error {
	// Handle errors implementing Err interface.
	if ex, ok := err.(Err); ok {
		logger.UnsafeFail(ex.Task())

		hint := ex.Hint()
		if next, ok := ex.Err().(Err); ok {
			// The next error is also Err, call recursively
			// after printing the hint when that is set.
			logger.UnsafeStderr(hint)
			return unsafeLogWith(next, logger)
		} else {
			// The next error is the root cause error.
			// It is not implementing Err, so we can print
			// the root cause error and stop the recursion.
			last := ex.Err()
			logger.UnsafeNewLine(fmt.Sprintf("(error = %v)", last))
			logger.UnsafeStderr(hint)
			return last
		}
	}

	// Handle regular errors.
	//
	// This block is only executed when this function is called
	// for the first time with an error not implementing Err.
	logger.UnsafeFail("Unknown task")
	logger.UnsafeNewLine(fmt.Sprintf("(error = %v)", err))
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
	if ex, ok := err.(Err); ok {
		return RootCause(ex.Err())
	}
	return err
}
