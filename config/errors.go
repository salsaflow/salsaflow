package config

import (
	"fmt"
)

type ErrFieldNotSet struct {
	fieldName string
}

func (err *ErrFieldNotSet) Error() string {
	return fmt.Sprintf("field '%s' is not set", err.fieldName)
}
