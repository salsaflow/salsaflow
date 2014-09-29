package jira

import "github.com/salsita/SalsaFlow/git-trunk/modules/jira/client"

type user struct {
	*client.User
}

func (u *user) Id() string {
	return u.User.Name
}
