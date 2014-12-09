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

package pivotal

import (
	"net/http"
	"time"
)

type Me struct {
	Id                         int        `json:"id"`
	Name                       string     `json:"name"`
	Initials                   string     `json:"initials"`
	Username                   string     `json:"username"`
	TimeZone                   *TimeZone  `json:"time_zone"`
	ApiToken                   string     `json:"api_token"`
	HasGoogleIdentity          bool       `json:"has_google_identity"`
	ProjectIds                 []int      `json:"project_ids"`
	WorkspaceIds               []int      `json:"workspace_ids"`
	Email                      string     `json:"email"`
	ReceivedInAppNotifications bool       `json:"receives_in_app_notifications"`
	CreatedAt                  *time.Time `json:"created_at"`
	UpdatedAt                  *time.Time `json:"updated_at"`
}

type MeService struct {
	client *Client
}

func newMeService(client *Client) *MeService {
	return &MeService{client}
}

func (service *MeService) Get() (*Me, *http.Response, error) {
	req, err := service.client.NewRequest("GET", "me", nil)
	if err != nil {
		return nil, nil, err
	}

	var me Me
	resp, err := service.client.Do(req, &me)
	if err != nil {
		return nil, resp, err
	}

	return &me, resp, nil
}
