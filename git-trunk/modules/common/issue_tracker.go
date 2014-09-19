package common

import (
	"github.com/salsita/SalsaFlow/git-trunk/version"
)

// IssueTracker interface ------------------------------------------------------

type IssueTracker interface {
	ListActiveStoryIds(ids []string) (activeIds []string, err error)
	NextRelease(*version.Version) (NextRelease, error)
	RunningRelease(*version.Version) (RunningRelease, error)
}

type NextRelease interface {
	PromptUserToConfirmStart() (bool, error)
	Start() (Action, error)
}

type RunningRelease interface {
	ListStoryIds() ([]string, error)
	EnsureDeliverable() error
	Deliver() (Action, error)
}

type Action interface {
	Rollback() error
}
