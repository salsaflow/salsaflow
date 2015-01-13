package sprintly

import (
	"fmt"
)

type ErrItems400 struct {
	Err *ErrAPI
}

func (err *ErrItems400) Error() string {
	return fmt.Sprintf("%v (invalid type, status or order_by)", err.Err)
}

type ErrItems404 struct {
	Err *ErrAPI
}

func (err *ErrItems404) Error() string {
	return fmt.Sprintf("%v (assigned_to or created_by users unknown or invalid)", err.Err)
}
