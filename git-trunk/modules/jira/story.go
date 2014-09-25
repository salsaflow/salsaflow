package jira

import (
	// Internal
	"github.com/salsita/SalsaFlow/git-trunk/errors"
	"github.com/salsita/SalsaFlow/git-trunk/modules/common"
)

type story struct {
}

func (story *story) GetId() string {
	panic("Not implemented")

func (story *story) GetReadableId() string {
	return story.s.Key
}

func (story *story) GetAssignees() []common.User {
	panic("Not implemented")
}

func (story *story) GetTitle() string {
	panic("Not implemented")
}

func (story *story) Start() *errors.Error {
	panic("Not implemented")
}

func (story *story) SetOwners([]common.User) *errors.Error {
	panic("Not implemented")
}
