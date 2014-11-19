package prompt

import (
	// Stdlib
	"errors"

	// Internal
	"github.com/salsita/salsaflow/log"
)

var ErrCanceled = errors.New("operation canceled")

func PanicCancel() {
	panic(ErrCanceled)
}

func RecoverCancel() {
	if r := recover(); r != nil {
		if r == ErrCanceled {
			log.Println("\nOperation canceled. You are welcome to come back any time!")
		} else {
			panic(r)
		}
	}
}
