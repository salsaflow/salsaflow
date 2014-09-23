package jira

import (
	// Internal
	"github.com/salsita/SalsaFlow/git-trunk/modules/common"
	"github.com/salsita/SalsaFlow/git-trunk/version"
)

type nextRelease struct {
}

func newNextRelease(ver *version.Version) (*nextRelease, error) {
	panic("Not implemented")
}

func (release *nextRelease) PromptUserToConfirmStart() (bool, error) {
	panic("Not implemented")
}

func (release *nextRelease) Start() (common.Action, error) {
	panic("Not implemented")
}
