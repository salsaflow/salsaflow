// Copyright (c) 2014 Salsita Software
// Use of this source code is governed by the MIT License.
// The license can be found in the LICENSE file.

package pivotal

import (
	"fmt"
	"net/http"
)

// ErrAPI ----------------------------------------------------------------------

type Error struct {
	Code             string `json:"code"`
	Error            string `json:"error"`
	Requirement      string `json:"requirement"`
	GeneralProblem   string `json:"general_problem"`
	PossibleFix      string `json:"possible_fix"`
	ValidationErrors []struct {
		Field   string `json:"field"`
		Problem string `json:"problem"`
	} `json:"validation_errors"`
}

type ErrAPI struct {
	Response *http.Response
	Err      *Error
}

func (err *ErrAPI) Error() string {
	req := err.Response.Request
	return fmt.Sprintf(
		"%v %v -> %v (error = %+v)",
		req.Method,
		req.URL,
		err.Response.Status,
		err.Err)
}

// ErrFieldNotSet --------------------------------------------------------------

type ErrFieldNotSet struct {
	fieldName string
}

func (err *ErrFieldNotSet) Error() string {
	return fmt.Sprintf("Required field '%s' is not set")
}
