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

package jira

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"
)

const (
	LibraryVersion = "0.0.1"

	DefaultMaxPendingRequests = 10

	defaultUserAgent = "go-jira/" + LibraryVersion
)

type (
	L []interface{}
	M map[string]interface{}
)

type Client struct {
	// HTTP client to be used to send all the HTTP requests.
	httpClient *http.Client

	// Base URL of the Jira API that is to be used to form API requests.
	BaseURL *url.URL

	// User-Agent header to be set for every request.
	UserAgent string

	// Me service.
	Myself *MyselfService

	// Project service.
	Projects *ProjectService

	// Issue service.
	Issues *IssueService

	// Remote Issue Link service.
	RemoteIssueLinks *RemoteIssueLinkService

	// Version service.
	Versions *VersionService

	// requestCh is used to limit the number of pending requests.
	requestCh chan struct{}

	// Options
	optMaxPendingRequests int
}

func NewClient(baseURL *url.URL, httpClient *http.Client, options ...func(*Client)) *Client {
	// Create a Client object.
	client := &Client{
		httpClient:            httpClient,
		BaseURL:               baseURL,
		UserAgent:             defaultUserAgent,
		optMaxPendingRequests: DefaultMaxPendingRequests,
	}

	// Set up the API services.
	client.Myself = newMyselfService(client)
	client.Projects = newProjectService(client)
	client.Issues = newIssueService(client)
	client.RemoteIssueLinks = newRemoteIssueLinkService(client)
	client.Versions = newVersionService(client)

	// Set custom options.
	for _, option := range options {
		option(client)
	}

	// Finish initialising the client.
	client.requestCh = make(chan struct{}, client.optMaxPendingRequests)

	// Return the new Client instance.
	return client
}

// SetOptMaxPendingRequests can be used to set a custom queue size
// for the requests that are to be sent to JIRA.
//
// It only makes sense to call this method from an option function.
// Calling it later on will have no effect whatsoever.
func (c *Client) SetOptMaxPendingRequests(limit int) {
	c.optMaxPendingRequests = limit
}

func (c *Client) NewRequest(method, urlPath string, body interface{}) (*http.Request, error) {
	path, err := url.Parse(urlPath)
	if err != nil {
		return nil, err
	}

	u := c.BaseURL.ResolveReference(path)

	var rawBody bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&rawBody).Encode(body); err != nil {
			return nil, err
		}
	}

	req, err := http.NewRequest(method, u.String(), &rawBody)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", c.UserAgent)
	return req, nil
}

func (c *Client) Do(req *http.Request, responseResource interface{}) (*http.Response, error) {
	// Acquire a request slot by sending to the request channel.
	c.requestCh <- struct{}{}
	defer func() {
		// Release the request slot by receiving from the request channel.
		<-c.requestCh
	}()

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode > 299 {
		// Try to parse the body as the error object.
		var errObject Error
		err := json.NewDecoder(resp.Body).Decode(&errObject)
		if err == nil {
			// Fill in the error object on success.
			return resp, &ErrAPI{
				Response: resp,
				Err:      &errObject,
			}
		} else {
			// Otherwise leave the error object empty.
			return resp, &ErrAPI{
				Response: resp,
			}
		}
	}

	if responseResource != nil {
		err = json.NewDecoder(resp.Body).Decode(responseResource)
	}

	return resp, err
}
