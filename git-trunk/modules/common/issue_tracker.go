package common

import (
	"github.com/salsita/SalsaFlow/git-trunk/version"
)

// IssueTracker interface ------------------------------------------------------

type IssueTracker interface {
	CurrentUser() (User, error)
	ActiveStoryIds(ids []string) (activeIds []string, err error)
	NextRelease(*version.Version) (NextRelease, error)
	RunningRelease(*version.Version) (RunningRelease, error)
}

type User interface {
	GetId() string
}

type Story interface {
	GetId() string
	GetAssignees() []User
}

type NextRelease interface {
	PromptUserToConfirmStart() (bool, error)
	Start() (Action, error)
}

type RunningRelease interface {
	ListStories() ([]Story, error)
	EnsureDeliverable() error
	Deliver() (Action, error)
}
