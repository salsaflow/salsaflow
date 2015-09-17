package config

import (
	// Stdlib
	"fmt"
)

// ErrKeyNotSet is returned when a configuration key is not set.
type ErrKeyNotSet struct {
	Key string
}

func (err *ErrKeyNotSet) Error() string {
	return fmt.Sprintf("key '%s' is not set", err.Key)
}

// ErrKeyInvalid can be returned when a configuration value is not valid.
type ErrKeyInvalid struct {
	Key   string
	Value interface{}
}

func (err *ErrKeyInvalid) Error() string {
	return fmt.Sprintf("key '%s' is invalid (value = %v)", err.Key, err.Value)
}

// ErrConfigRecordNotFound is returned from ConfigurationsSection.ConfigRecord
// when the section specified by the given module kind cannot be found.
type ErrConfigRecordNotFound struct {
	configKey string
}

func (err *ErrConfigRecordNotFound) Error() string {
	return fmt.Sprintf(
		"configuration record not found for key '%v'", err.configKey)
}
