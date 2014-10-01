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
)

// Resources -------------------------------------------------------------------

type Version struct {
	Id            string `json:"id,omitempty"`
	Self          string `json:"self,omitempty"`
	Name          string `json:"name,omitempty"`
	Description   string `json:"description,omitempty"`
	Released      bool   `json:"released,omitempty"`
	Archived      bool   `json:"archived,omitempty"`
	StartDate     string `json:"startDate,omitempty"`
	UserStartDate string `json:"userStartDate,omitempty"`
	ProjectId     int    `json:"projectId,omitempty"`
}

// The service -----------------------------------------------------------------

type VersionService struct {
	client *Client
}

func newVersionService(client *Client) *VersionService {
	return &VersionService{client}
}

// Update updates the version with the specified ID as specified in the change request.
func (service *VersionService) Update(id string, change *Version) (*http.Response, error) {
	u := fmt.Sprintf("api/2/version/%v", id)
	req, err := service.client.NewRequest("PUT", u, change)
	if err != nil {
		return nil, err
	}
	return service.client.Do(req, nil)
}
