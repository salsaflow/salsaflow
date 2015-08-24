package common

import (
	"github.com/salsaflow/salsaflow/action"
	"github.com/salsaflow/salsaflow/version"
)

// ReleaseNotesManagerFactory interface ----------------------------------------

type ReleaseNotesManagerFactory interface {
	LocalConfigTemplate() string
	NewReleaseNotesManager() (ReleaseNotesManager, error)
}

// ReleaseNotesManager interface -----------------------------------------------

// ReleaseNotesManager is used to post release notes once a release is closed.
type ReleaseNotesManager interface {

	// PostReleaseNotes post the given release notes.
	PostReleaseNotes(*ReleaseNotes) (action.Action, error)
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

// Implement sort.Interface to sort story sections alphabetically.
type ReleaseNotesSections []*ReleaseNotesSection

func (sections ReleaseNotesSections) Len() int {
	return len(sections)
}

func (sections ReleaseNotesSections) Less(i, j int) bool {
	return sections[i].StoryType < sections[j].StoryType
}

func (sections ReleaseNotesSections) Swap(i, j int) {
	sections[i], sections[j] = sections[j], sections[i]
}
