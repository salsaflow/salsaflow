/*
   Copyright (C) 2014  Salsita s.r.o.

   This program is free software: you can redistribute it and/or modify
   it under the terms of the GNU General Public License as published by
   the Free Software Foundation, either version 3 of the License, or
   (at your option) any later version.

   This program is distributed in the hope that it will be useful,
   but WITHOUT ANY WARRANTY; without even the implied warranty of
   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
   GNU General Public License for more details.

   You should have received a copy of the GNU General Public License
   along with this program. If not, see {http://www.gnu.org/licenses/}.
*/

package client

import (
	"fmt"
	"net/http"
)

type Error struct {
	ErrorMessages []string          `json:"error_messages"`
	Errors        map[string]string `json:"errors"`
}

// ErrAPI ----------------------------------------------------------------------

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