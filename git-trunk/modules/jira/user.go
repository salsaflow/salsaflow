package jira

import "github.com/salsita/SalsaFlow/git-trunk/modules/jira/client"

type myself struct {
	*client.User
}

func (me *myself) GetId() string {
	return me.Name
}
