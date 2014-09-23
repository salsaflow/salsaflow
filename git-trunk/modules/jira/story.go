package jira

import (
	// Internal
	"github.com/salsita/SalsaFlow/git-trunk/modules/common"
)

type story struct {
}

func (story *story) GetId() string {
	panic("Not implemented")
}

func (story *story) GetAssignees() []common.User {
	panic("Not implemented")
}
