package config

import (
	"fmt"

	"github.com/salsita/SalsaFlow/git-trunk/log"
)

type ErrKeyNotSet struct {
	key string
}

func (err *ErrKeyNotSet) Error() string {
	return fmt.Sprintf("key '%s' is not set", err.key)
}

type ErrKeyInvalid struct {
	key   string
	value interface{}
}

func (err *ErrKeyInvalid) Error() string {
	return fmt.Sprintf("key '%s' is invalid (value = %v)", err.key, err.value)
}

func die(msg string, err error) {
	log.Fail(msg)
	log.Fatalln("\nError:", err)
}
