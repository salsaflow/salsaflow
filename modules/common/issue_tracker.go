package common

import (
	// Stdlib
	"errors"

	// Internal
	"github.com/salsaflow/salsaflow/action"
	"github.com/salsaflow/salsaflow/errs"
	"github.com/salsaflow/salsaflow/version"
)

var ErrReleaseNotFound = errors.New("release not found")

// IssueTracker interface ------------------------------------------------------

type IssueTracker interface {

	// CurrentUser returns the issue tracker account details of the current user.
	// The account ID is taken from the global SalsaFlow configuration file.
	CurrentUser() (User, error)

	// StartableStories returns the list of stories that can be started.
	StartableStories() ([]Story, error)

	// StoriesInDevelopment returns the list of stories that are being developed.
	StoriesInDevelopment() ([]Story, error)

	// NextRelease is a factory method for creating release objects
	// representing the releases that have not been started yet.
	//
	// The current version is the version on the trunk branch,
	// the next version is the next trunk version, i.e. the version that is bumped
	// to the trunk branch once the release branch is created.
	NextRelease(current *version.Version, next *version.Version) (NextRelease, error)

	// RunningRelease is a factory method for creating release objects
	// representing the releases that have been started.
	RunningRelease(*version.Version) (RunningRelease, error)

	// SelectActiveStoryIds returns the IDs associated with the stories
	// that are being actively worked on, i.e. they are not closed yet.
	SelectActiveStoryIds(ids []string) (activeIds []string, err error)

	// OpenStory opens the given story in the web browser.
	OpenStory(storyId string) error
}

type User interface {
	Id() string
}

type Story interface {
	Id() string
	ReadableId() string
	Title() string

	Assignees() []User
	AddAssignee(User) *errs.Error
	SetAssignees([]User) *errs.Error

	Start() *errs.Error
}

type NextRelease interface {
	PromptUserToConfirmStart() (bool, error)
	Start() (action.Action, error)
}

type RunningRelease interface {
	Version() *version.Version
	Stories() ([]Story, error)
	EnsureStageable() error
	Stage() (action.Action, error)
	CheckReleasable() (unreleasable []Story, err error)
	Release() error
}
