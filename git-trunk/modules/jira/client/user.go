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

type User struct {
	Self         string
	Key          string
	Name         string
	EmailAddress string `json:"emailAddress"`
	AvatarURLs   struct {
		Size16 string
		Size24 string
		Size32 string
		Size48 string
	} `json:"avatarUrls"`
	DisplayName string `json:"displayName"`
	Active      bool
	TimeZone    string `json:"timeZone"`
	Groups      struct {
		Size  int
		Items []struct {
			Name string
			Self string
		}
	}
}
