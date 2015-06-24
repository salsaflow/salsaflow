package client

import (
	"errors"
	"fmt"
	"net/http"
)

var ErrNoTrailingSlash = errors.New("trailing slash missing")

type ErrAPI struct {
	Response *http.Response
}

func (err *ErrAPI) Error() string {
	req := err.Response.Request
	return fmt.Sprintf("%v %v -> %v", req.Method, req.URL, err.Response.Status)
}
