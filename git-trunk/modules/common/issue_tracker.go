package common

import (
	"github.com/salsita/SalsaFlow/git-trunk/errors"
	"github.com/salsita/SalsaFlow/git-trunk/version"
)

// IssueTracker interface ------------------------------------------------------

type IssueTracker interface {
	ActiveStoryIds(ids []string) (activeIds []string, err error)
	CurrentUser() (User, error)
	GetStartableStories() ([]Story, error)
	NextRelease(*version.Version) (NextRelease, error)
	RunningRelease(*version.Version) (RunningRelease, error)
}

type User interface {
	GetId() string
}

type Story interface {
	GetId() string
	GetAssignees() []User
	GetTitle() string
	Start() *errors.Error
	SetOwners([]User) *errors.Error
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
