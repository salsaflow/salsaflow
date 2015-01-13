package sprintly

import (
	"fmt"
)

type ErrDeploys400 struct {
	Err *ErrAPI
}

func (err *ErrDeploys400) Error() string {
	return fmt.Sprintf("%v (items not found)", err.Err)
}

type ErrDeploys403 struct {
	Err *ErrAPI
}

func (err *ErrDeploys403) Error() string {
	return fmt.Sprintf("%v (sender not a member of the given product)", err.Err)
}

type ErrDeploys404 struct {
	Err *ErrAPI
}

func (err *ErrDeploys404) Error() string {
	return fmt.Sprintf("%v (product ID invalid or unknown)", err.Err)
}
