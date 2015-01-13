// Copyright 2013 The go-github AUTHORS. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package sprintly

import (
	"io"
	"net/url"
	"reflect"
	"strings"

	"github.com/google/go-querystring/query"
)

func appendArgs(u *url.URL, args interface{}) error {
	if args == nil {
		return nil
	}

	v := reflect.ValueOf(args)
	if v.Kind() == reflect.Ptr && v.IsNil() {
		return nil
	}

	values, err := query.Values(args)
	if err != nil {
		return err
	}

	u.RawQuery = values.Encode()
	return nil
}

func encodeArgs(args interface{}) (io.Reader, error) {
	if args == nil {
		return nil, nil
	}

	v := reflect.ValueOf(args)
	if v.Kind() == reflect.Ptr && v.IsNil() {
		return nil, nil
	}

	values, err := query.Values(args)
	if err != nil {
		return nil, err
	}

	return strings.NewReader(values.Encode()), nil
}
