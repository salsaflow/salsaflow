package jira

import (
	"github.com/salsita/salsaflow/action"
	"github.com/salsita/salsaflow/modules/common"
	"github.com/salsita/salsaflow/version"
)

type runningRelease struct {
}

func newRunningRelease(ver *version.Version) (*runningRelease, error) {
	panic("Not implemented")
}

func (release *runningRelease) Stories() ([]common.Story, error) {
	panic("Not implemented")
}

func (release *runningRelease) EnsureStageable() error {
	panic("Not implemented")
}

func (release *runningRelease) Stage() (action.Action, error) {
	panic("Not implemented")
}
