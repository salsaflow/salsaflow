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

import "net/http"

type MyselfService struct {
	client *Client
}

func newMyselfService(client *Client) *MyselfService {
	return &MyselfService{client}
}

func (service *MyselfService) Get() (*User, *http.Response, error) {
	req, err := service.client.NewRequest("GET", "myself", nil)
	if err != nil {
		return nil, nil, err
	}

	var myself User
	resp, err := service.client.Do(req, &myself)
	if err != nil {
		return nil, resp, err
	}

	return &myself, resp, nil
}
