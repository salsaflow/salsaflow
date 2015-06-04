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

// Resources -------------------------------------------------------------------

type Version struct {
	Id            string `json:"id,omitempty"`
	Self          string `json:"self,omitempty"`
	Name          string `json:"name,omitempty"`
	Description   string `json:"description,omitempty"`
	Project       string `json:"project,omitempty"`
	ProjectId     int    `json:"projectId,omitempty"`
	Released      bool   `json:"released,omitempty"`
	Archived      bool   `json:"archived,omitempty"`
	StartDate     string `json:"startDate,omitempty"`
	UserStartDate string `json:"userStartDate,omitempty"`
}

// The service -----------------------------------------------------------------

type VersionService struct {
	client *Client
}

func newVersionService(client *Client) *VersionService {
	return &VersionService{client}
}

// Create creates a new version.
func (service *VersionService) Create(version *Version) (*Version, *http.Response, error) {
	switch {
	case version.Name == "":
		return nil, nil, &ErrFieldNotSet{"Version.Name"}
	case version.Project == "":
		return nil, nil, &ErrFieldNotSet{"Version.Project"}
	}

	req, err := service.client.NewRequest("POST", "version", version)
	if err != nil {
		return nil, nil, err
	}

	var createdVersion Version
	resp, err := service.client.Do(req, &createdVersion)
	if err != nil {
		return nil, resp, err
	}
	return &createdVersion, resp, nil
}

// Update updates the version with the specified ID as specified in the change request.
func (service *VersionService) Update(id string, change *Version) (*http.Response, error) {
	u := fmt.Sprintf("version/%v", id)
	req, err := service.client.NewRequest("PUT", u, change)
	if err != nil {
		return nil, err
	}
	return service.client.Do(req, nil)
}

// Delete deleted the version with the specified ID.
func (service *VersionService) Delete(id string) (*http.Response, error) {
	u := fmt.Sprintf("version/%v", id)
	req, err := service.client.NewRequest("DELETE", u, nil)
	if err != nil {
		return nil, err
	}

	return service.client.Do(req, nil)
}
