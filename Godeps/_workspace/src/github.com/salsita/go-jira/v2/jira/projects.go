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
	"fmt"
	"net/http"
)

type Project struct {
	Self       string      `json:"self"`
	Id         string      `json:"id"`
	Key        string      `json:"key"`
	Name       string      `json:"name"`
	AvatarURLs *AvatarURLs `json:"avatarUrls,omitempty"`
}

type ProjectService struct {
	client *Client
}

func newProjectService(client *Client) *ProjectService {
	return &ProjectService{client}
}

func (service *ProjectService) List() ([]*Project, *http.Response, error) {
	req, err := service.client.NewRequest("GET", "project", nil)
	if err != nil {
		return nil, nil, err
	}

	var projects []*Project
	resp, err := service.client.Do(req, &projects)
	if err != nil {
		return nil, resp, err
	}

	return projects, resp, nil
}

func (service *ProjectService) ListVersions(projectIdOrKey string) ([]*Version, *http.Response, error) {
	u := fmt.Sprintf("project/%v/versions", projectIdOrKey)
	req, err := service.client.NewRequest("GET", u, nil)
	if err != nil {
		return nil, nil, err
	}

	var versions []*Version
	resp, err := service.client.Do(req, &versions)
	if err != nil {
		return nil, resp, err
	}

	return versions, resp, nil
}
