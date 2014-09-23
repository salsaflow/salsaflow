package jira

import (
	// Internal
	"github.com/salsita/SalsaFlow/git-trunk/modules/common"
	"github.com/salsita/SalsaFlow/git-trunk/version"
)

type runningRelease struct {
}

func newRunningRelease(ver *version.Version) (*runningRelease, error) {
	panic("Not implemented")
}

func (release *runningRelease) ListStories() ([]common.Story, error) {
	panic("Not implemented")
}

func (release *runningRelease) EnsureDeliverable() error {
	panic("Not implemented")
}

func (release *runningRelease) Deliver() (common.Action, error) {
	panic("Not implemented")
}
