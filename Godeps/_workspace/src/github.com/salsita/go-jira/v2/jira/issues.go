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

	// Vendor
	"github.com/google/go-querystring/query"
)

// Resources -------------------------------------------------------------------

// IssueList represents a list of issues, what a surprise.
type IssueList struct {
	Expand     string   `json:"expand,omitempty"`
	StartAt    int      `json:"startAt,omitempty"`
	MaxResults int      `json:"maxResults,omitempty"`
	Total      int      `json:"total,omitempty"`
	Issues     []*Issue `json:"issues,omitempty"`
}

// Issue represents the issue resource as produces by the REST API.
type Issue struct {
	Id     string `json:"id,omitempty"`
	Self   string `json:"self,omitempty"`
	Key    string `json:"key,omitempty"`
	Fields struct {
		Summary   string `json:"summary,omitempty"`
		IssueType struct {
			Id          string `json:"id,omitempty"`
			Self        string `json:"self,omitempty"`
			Name        string `json:"name,omitempty"`
			Description string `json:"description,omitempty"`
			Subtask     bool   `json:"subtask,omitempty"`
			IconURL     string `json:"iconUrl,omitempty"`
		} `json:"issuetype,omitempty"`
		Parent      *Issue           `json:"parent,omitempty"`
		Subtasks    []*Issue         `json:"subtasks,omitempty"`
		Assignee    *User            `json:"assignee,omitempty"`
		FixVersions []*Version       `json:"fixVersions,omitempty"`
		Labels      []string         `json:"labels,omitempty"`
		Status      *IssueStatus     `json:"status,omitempty"`
		Resolution  *IssueResolution `json:"resolution,omitempty"`
	} `json:"fields,omitempty"`
}

// IssueStatus represents an issue status, e.g. "Scheduled".
type IssueStatus struct {
	Id          string `json:"id,omitempty"`
	Self        string `json:"self,omitempty"`
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
	IconURL     string `json:"iconUrl,omitempty"`
}

// IssueResolution represents an issue resolution, e.g. "Cannot Reproduce".
type IssueResolution struct {
	Id          string `json:"id,omitempty"`
	Self        string `json:"self,omitempty"`
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
}

// The service -----------------------------------------------------------------

type IssueService struct {
	client *Client
}

func newIssueService(client *Client) *IssueService {
	return &IssueService{client}
}

type SearchOptions struct {
	JQL           string `url:"jql,omitempty"`
	StartAt       int    `url:"startAt,omitempty"`
	MaxResults    int    `url:"maxResults,omitempty"`
	ValidateQuery bool   `url:"validateQuery,omitempty"`
}

func (service *IssueService) Search(opts *SearchOptions) ([]*Issue, *http.Response, error) {
	u := "search"
	if opts != nil {
		vs, err := query.Values(opts)
		if err != nil {
			panic(err)
		}
		u += "?" + vs.Encode()
	}

	req, err := service.client.NewRequest("GET", u, nil)
	if err != nil {
		return nil, nil, err
	}

	var issueList IssueList
	resp, err := service.client.Do(req, &issueList)
	if err != nil {
		return nil, nil, err
	}

	// TODO: Deal with pagination.
	return issueList.Issues, resp, nil
}

// Get returns the chosen issue.
func (service *IssueService) Get(issueIdOrKey string) (*Issue, *http.Response, error) {
	u := fmt.Sprintf("issue/%v", issueIdOrKey)
	req, err := service.client.NewRequest("GET", u, nil)
	if err != nil {
		return nil, nil, err
	}

	var issue Issue
	resp, err := service.client.Do(req, &issue)
	if err != nil {
		return nil, nil, err
	}
	return &issue, resp, nil
}

// Update updates the chosen issue.
func (service *IssueService) Update(issueIdOrKey string, body interface{}) (*http.Response, error) {
	u := fmt.Sprintf("issue/%v", issueIdOrKey)
	req, err := service.client.NewRequest("PUT", u, body)
	if err != nil {
		return nil, err
	}
	return service.client.Do(req, nil)
}

// PerformTransition performs the requested transition for the chosen issue.
func (service *IssueService) PerformTransition(
	issueIdOrKey string,
	transition interface{},
) (*http.Response, error) {

	u := fmt.Sprintf("issue/%v/transitions", issueIdOrKey)
	req, err := service.client.NewRequest("POST", u, transition)
	if err != nil {
		return nil, err
	}

	return service.client.Do(req, nil)
}
