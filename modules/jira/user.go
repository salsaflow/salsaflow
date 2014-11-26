package jira

import "github.com/salsaflow/salsaflow/modules/jira/client"

type user struct {
	*client.User
}

func (u *user) Id() string {
	return u.User.Name
}
