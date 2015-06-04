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
	// Stdlib
	"fmt"
	"net/http"
	"net/url"
)

// Resources -------------------------------------------------------------------

// RemoteIssueLink represents a JIRA issue remote link.
type RemoteIssueLink struct {
	GlobalId     string `json:"globalId,omitempty"`
	Relationship string `json:"relationship,omitempty"`
	Object       struct {
		URL     string `json:"url,omitempty"`
		Title   string `json:"title,omitempty"`
		Summary string `json:"summary,omitempty"`
		Icon    struct {
			URL   string `json:"url16x16,omitempty"`
			Title string `json:"title,omitempty"`
		} `json:"icon,omitempty"`
		Status struct {
			Resolved bool `json:"resolved"`
			Icon     struct {
				URL   string `json:"url16x16,omitempty"`
				Title string `json:"title,omitempty"`
				Link  string `json:"link,omitempty"`
			} `json:"icon,omitempty"`
		} `json:"status,omitempty"`
	} `json:"object,omitempty"`
	Application struct {
		Name string `json:"name,omitempty"`
		Type string `json:"type,omitempty"`
	} `json:"application,omitempty"`
}

// The service -----------------------------------------------------------------

type RemoteIssueLinkService struct {
	client *Client
}

func newRemoteIssueLinkService(client *Client) *RemoteIssueLinkService {
	return &RemoteIssueLinkService{client}
}

// List returns the remote issue links associated with the given issue.
func (service *RemoteIssueLinkService) List(
	issueIdOrKey string,
) ([]*RemoteIssueLink, *http.Response, error) {

	u := fmt.Sprintf("issue/%v/remotelink", issueIdOrKey)
	req, err := service.client.NewRequest("GET", u, nil)
	if err != nil {
		return nil, nil, err
	}

	var links []*RemoteIssueLink
	resp, err := service.client.Do(req, &links)
	if err != nil {
		return nil, nil, err
	}
	return links, resp, nil
}

// Get returns the remote issue link identified by the given issue ID or key and global ID.
func (service *RemoteIssueLinkService) Get(
	issueIdOrKey string,
	globalId string,
) (*RemoteIssueLink, *http.Response, error) {

	u := fmt.Sprintf("issue/%v/remotelink?globalId=%v", issueIdOrKey, url.QueryEscape(globalId))
	req, err := service.client.NewRequest("GET", u, nil)
	if err != nil {
		return nil, nil, err
	}

	var link RemoteIssueLink
	resp, err := service.client.Do(req, &link)
	if err != nil {
		return nil, nil, err
	}
	return &link, resp, nil
}

// Create creates a new remote issue link for the given issue.
// Global ID needs to be filled in if the remote issue link is to be updated in the future.
func (service *RemoteIssueLinkService) Create(
	issueIdOrKey string,
	link *RemoteIssueLink,
) (*http.Response, error) {

	u := fmt.Sprintf("issue/%v/remotelink", issueIdOrKey)
	req, err := service.client.NewRequest("POST", u, link)
	if err != nil {
		return nil, err
	}
	return service.client.Do(req, nil)
}

// Update updates the remote issue link identified by the given issue ID.
// Global ID needs to be filled in for this API call to succeed.
func (service *RemoteIssueLinkService) Update(
	issueIdOrKey string,
	link *RemoteIssueLink,
) (*http.Response, error) {

	return service.Create(issueIdOrKey, link)
}

// Delete deletes the remote issue link identified by the given issue ID or key and global ID.
func (service *RemoteIssueLinkService) Delete(
	issueIdOrKey string,
	globalId string,
) (*http.Response, error) {

	u := fmt.Sprintf("issue/%v/remotelink?globalId=%v", issueIdOrKey, url.QueryEscape(globalId))
	req, err := service.client.NewRequest("DELETE", u, nil)
	if err != nil {
		return nil, err
	}
	return service.client.Do(req, nil)
}
