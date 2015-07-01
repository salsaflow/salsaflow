package common

import (
	// Internal
	"github.com/salsaflow/salsaflow/action"
	"github.com/salsaflow/salsaflow/version"
)

// IssueTracker interface ------------------------------------------------------

type IssueTracker interface {

	// ServiceName returns the name of the service this interface represents.
	ServiceName() string

	// CurrentUser returns the issue tracker account details of the current user.
	// The account ID is taken from the global SalsaFlow configuration file.
	CurrentUser() (User, error)

	// StartableStories returns the list of stories that can be started.
	StartableStories() ([]Story, error)

	// StoriesInDevelopment returns the list of stories that are being developed.
	StoriesInDevelopment() ([]Story, error)

	// ReviewedStories returns the list of stories that have been reviewed already.
	ReviewedStories() ([]Story, error)

	// ListStoriesByTag returns the stories for the given list of Story-Id tags.
	ListStoriesByTag(tags []string) ([]Story, error)

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

	// OpenStory opens the given story in the web browser.
	OpenStory(storyId string) error

	// StoryTagToReadableStoryId parses the Story-Id tag and returns the relevant readable ID.
	StoryTagToReadableStoryId(tag string) (storyId string, err error)

	// ReleaseNotes generates release notes for the given version.
	// In case the given release is not found, ErrReleaseNotFound is returned.
	ReleaseNotes(*version.Version) (*ReleaseNotes, error)
}

type User interface {
	Id() string
}

type StoryState string

const (
	StoryStateNew              StoryState = "new"
	StoryStateApproved         StoryState = "approved"
	StoryStateBeingImplemented StoryState = "being implemented"
	StoryStateImplemented      StoryState = "implemented"
	StoryStateReviewed         StoryState = "reviewed"
	StoryStateBeingTested      StoryState = "being tested"
	StoryStateTested           StoryState = "tested"
	StoryStateStaged           StoryState = "staged"
	StoryStateAccepted         StoryState = "accepted"
	StoryStateRejected         StoryState = "rejected"
	StoryStateClosed           StoryState = "closed"
	StoryStateInvalid          StoryState = "invalid"
)

type Story interface {
	// Id returns the ID of the story.
	Id() string

	// ReadableId returns the human-friendly ID of the story.
	// This ID is used when listing stories to the user.
	ReadableId() string

	// State returns the abstract state the story is in at the moment.
	State() StoryState

	// URL return the URL that can be used to access the story.
	URL() string

	// Tag returns a string that is then used for the Story-Id tag.
	// The tag is supposed to identify the story, but it might be
	// more complicated than just the story ID.
	Tag() string

	// Title returns a short description of the story.
	// It is used to describe stories when listing them to the user.
	Title() string

	// Assignees returns the list of users that are assigned to the story.
	Assignees() []User

	// AddAssignee can be used to add an additional user to the list of assignees.
	AddAssignee(User) error

	// SetAssigness can be used to set the list of assignees,
	// effectively replacing the current list.
	SetAssignees([]User) error

	// Start can be used to start the story in the issue tracker.
	Start() error

	// MarkAsImplemented marks the story as implemented.
	MarkAsImplemented() (action.Action, error)

	// LessThan is being used for sorting stories for output.
	// Stories are printed in the order they are sorted by this function.
	LessThan(Story) bool

	// IssueTracker can be used to get the issue tracker instance
	// that this story is associated with.
	IssueTracker() IssueTracker
}

// Stories implement sort.Interface
type Stories []Story

func (ss Stories) Len() int {
	return len(ss)
}

func (ss Stories) Less(i, j int) bool {
	return ss[i].LessThan(ss[j])
}

func (ss Stories) Swap(i, j int) {
	ss[i], ss[j] = ss[j], ss[i]
}

type NextRelease interface {
	PromptUserToConfirmStart() (bool, error)
	Start() (action.Action, error)
}

type RunningRelease interface {
	Version() *version.Version
	Stories() ([]Story, error)

	// EnsureStageable shall return *ErrNotStageable in case
	// it is not possible to stage the given release.
	EnsureStageable() error
	Stage() (action.Action, error)

	// EnsureReleasable shall return *ErrNotReleasable in case
	// it is not possible to release the given release.
	EnsureReleasable() error
	Release() error
}

// ReleaseNotes represent, well, the release notes for the given version.
type ReleaseNotes struct {
	Version  *version.Version
	Sections []*ReleaseNotesSection
}

// ReleaseNotesSection represents a section of release notes
// that is associated with certain story type.
type ReleaseNotesSection struct {
	StoryType string
	Stories   []Story
}
