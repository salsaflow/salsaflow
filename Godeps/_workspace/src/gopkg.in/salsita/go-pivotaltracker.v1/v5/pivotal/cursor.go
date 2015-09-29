/*

   Copyright (C) 2015 Scott Devoid
   Copyright (C) 2015 Salsita Software

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
	end       bool
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
	// The cursor has reached the end, return io.EOF
	if c.end {
		return nil, io.EOF
	}

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
	resp, err = c.client.Do(req, v)
	if err != nil {
		return nil, err
	}

	// Get limit, offset, total and returned headers for pagination
	limit := getIntHeader(&err, resp, "X-Tracker-Pagination-Limit")
	offset := getIntHeader(&err, resp, "X-Tracker-Pagination-Offset")
	total := getIntHeader(&err, resp, "X-Tracker-Pagination-Total")
	returned := getIntHeader(&err, resp, "X-Tracker-Pagination-Returned")
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

	// Return EOF on the next call in case we have reached the end.
	if c.offset >= total {
		c.end = true
	}

	return resp, nil
}

func (c *cursor) all(v interface{}) error {
	// Get the total number of items.
	total, err := c.total()
	if err != nil {
		return err
	}

	// Get the request object.
	req := c.requestFn()

	// Set limit=total, offset=0 to get all data at once.
	values, err := url.ParseQuery(req.URL.RawQuery)
	if err != nil {
		return err
	}
	values.Set("limit", strconv.Itoa(total))
	values.Set("offset", "0")
	req.URL.RawQuery = values.Encode()

	// Do the request, decode JSON to v.
	_, err = c.client.Do(req, v)
	return err
}

func (c *cursor) total() (int, error) {
	// Get the request object.
	req := c.requestFn()

	// Set limit=0, offset=0 to just get the pagination information.
	values, err := url.ParseQuery(req.URL.RawQuery)
	if err != nil {
		return 0, err
	}
	values.Set("limit", "0")
	values.Set("offset", "0")
	req.URL.RawQuery = values.Encode()

	// Do the request, decode JSON to v
	resp, err := c.client.Do(req, nil)
	if err != nil {
		return 0, err
	}

	total := getIntHeader(&err, resp, "X-Tracker-Pagination-Total")
	if err != nil {
		return 0, err
	}
	return total, nil
}

// Helper to extract and convert Header values that are Int's
func getIntHeader(err *error, resp *http.Response, header string) int {
	if *err != nil {
		return 0
	}
	v := resp.Header.Get(header)
	if v == "" {
		return 0
	}
	i, ex := strconv.Atoi(v)
	if ex != nil {
		*err = ex
		return 0
	}
	return i
}
