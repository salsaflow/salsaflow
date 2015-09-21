package jira

import "github.com/salsita/go-jira/v2/jira"

type user struct {
	*jira.User
}

func (u *user) Id() string {
	return u.User.Name
}
