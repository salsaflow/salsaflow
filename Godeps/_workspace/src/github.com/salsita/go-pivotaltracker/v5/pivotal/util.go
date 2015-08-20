/*

   Copyright (C) 2015 Scott Devoid

*/
package pivotal

import (
	"io"
	"net/http"
	"net/url"
	"strconv"
)

// requestFn is a function that returns a new *http.Request object.
type requestFn func() (req *http.Request)

// cursor tracks response headers from paginated API responses.
// And sets the appropriate URI variables in the next request.
type cursor struct {
	client    *Client
	requestFn requestFn
	limit     int
	offset    int
}

// newCursor creates a new cursor to interate over an endpoint that
// supports limit and offest request parameters.
func newCursor(client *Client, fn requestFn, limit int) (c *cursor, err error) {
	return &cursor{client: client, requestFn: fn, limit: limit}, nil
}

// next is called with a pointer to an []*Type, which will be correctly
// unmarshalled. next() returns the http.Response, where the Body is
// already closed and an error. When next() reaches the end of a paginated
// endpoint, it returns io.EOF as the error.
func (c *cursor) next(v interface{}) (resp *http.Response, err error) {

	req := c.requestFn()

	// Set the URL limit=X,offset=Y
	values, err := url.ParseQuery(req.URL.RawQuery)
	if err != nil {
		return nil, err
	}
	values.Set("limit", strconv.Itoa(c.limit))
	values.Set("offset", strconv.Itoa(c.offset))
	req.URL.RawQuery = values.Encode()

	// Do the request, decode JSON to v
	resp, err = c.client.Do(req, &v)
	if err != nil {
		return nil, err
	}

	// Helper to extract and convert Header values that are Int's
	getIntHeader := func(resp *http.Response, k string) int {
		if err != nil {
			return 0
		}
		i, cerr := strconv.Atoi(resp.Header.Get(k))
		if cerr != nil {
			err = cerr
		}
		return i
	}

	// Get limit, offset, total and returned headers for pagination
	limit := getIntHeader(resp, "X-Tracker-Pagination-Limit")
	offset := getIntHeader(resp, "X-Tracker-Pagination-Offset")
	total := getIntHeader(resp, "X-Tracker-Pagination-Total")
	returned := getIntHeader(resp, "X-Tracker-Pagination-Returned")
	if err != nil {
		return nil, err
	}

	// Calculate the new offset, which is the old offset plus
	// the minimum of (returned, limit)
	if returned < limit {
		c.offset = offset + returned
	} else {
		c.offset = offset + limit
	}

	// Return EOF if we have reached the end.
	if c.offset >= total {
		err = io.EOF
	}
	return resp, err
}
