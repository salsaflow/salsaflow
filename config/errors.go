package config

import (
	"fmt"

	"github.com/salsita/salsaflow/log"
)

type ErrKeyNotSet struct {
	Key string
}

func (err *ErrKeyNotSet) Error() string {
	return fmt.Sprintf("key '%s' is not set", err.Key)
}

type ErrKeyInvalid struct {
	Key   string
	Value interface{}
}

func (err *ErrKeyInvalid) Error() string {
	return fmt.Sprintf("key '%s' is invalid (value = %v)", err.Key, err.Value)
}

func die(msg string, err error) {
	log.Fail(msg)
	log.Fatalln("\nError:", err)
}
