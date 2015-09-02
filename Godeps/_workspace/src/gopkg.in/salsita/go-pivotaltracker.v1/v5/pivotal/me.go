// Copyright (c) 2014 Salsita Software
// Use of this source code is governed by the MIT License.
// The license can be found in the LICENSE file.

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
	ProjectIds                 *[]int     `json:"project_ids"`
	WorkspaceIds               *[]int     `json:"workspace_ids"`
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
