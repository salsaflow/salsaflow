package common

import (
	"github.com/salsita/salsaflow/errs"
	"github.com/salsita/salsaflow/version"
)

// IssueTracker interface ------------------------------------------------------

type IssueTracker interface {

	// SelectActiveStoryIds returns the IDs associated with the stories
	// that are being actively worked on, i.e. they are not closed yet.
	SelectActiveStoryIds(ids []string) (activeIds []string, err error)

	// CurrentUser returns the issue tracker account details of the current user.
	// The account ID is taken from the global SalsaFlow configuration file.
	CurrentUser() (User, error)

	// StartableStories returns the list of stories that can be started.
	StartableStories() ([]Story, error)

	// NextRelease is a factory method for creating release objects
	// representing the releases that have not been started yet.
	NextRelease(*version.Version) (NextRelease, error)

	// RunningRelease is a factory method for creating release objects
	// representing the releases that have been started.
	RunningRelease(*version.Version) (RunningRelease, error)
}

type User interface {
	Id() string
}

type Story interface {
	Id() string
	ReadableId() string
	Assignees() []User
	Title() string

	Start() *errs.Error

	SetOwners([]User) *errs.Error
}

type NextRelease interface {
	PromptUserToConfirmStart() (bool, error)
	Start() (Action, error)
}

type RunningRelease interface {
	Stories() ([]Story, error)
	EnsureDeliverable() error
	Deliver() (Action, error)
}
