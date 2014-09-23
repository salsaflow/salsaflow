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
	// Stdlib
	"fmt"
	"net/http"

	// Other
	"github.com/google/go-querystring/query"
)

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
		Assignee *User `json:"assignee,omitempty"`
	} `json:"fields,omitempty"`
}

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
	u := "api/2/search"
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

	var issues []*Issue
	resp, err := service.client.Do(req, &issues)
	if err != nil {
		return nil, nil, err
	}

	return issues, resp, nil
}

// PerformTransition will perform the requested transition for the chosen issue.
func (service *IssueService) PerformTransition(issueIdOrKey, transitionId string) (*http.Response, error) {
	u := fmt.Sprintf("api/2/issue/%v/transitions", issueIdOrKey)
	var p struct {
		Transition struct {
			Id string `json:"id"`
		} `json:"transition"`
	}
	p.Transition.Id = transitionId

	req, err := service.client.NewRequest("POST", u, &p)
	if err != nil {
		return nil, err
	}

	return service.client.Do(req, nil)
}
