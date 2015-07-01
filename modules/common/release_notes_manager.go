package common

import (
	"github.com/salsaflow/salsaflow/action"
)

// ReleaseNotesManager is used to post release notes once a release is closed.
type ReleaseNotesManager interface {

	// PostReleaseNotes post the given release notes.
	PostReleaseNotes(*ReleaseNotes) (action.Action, error)
}
