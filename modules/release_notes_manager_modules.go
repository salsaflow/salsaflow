package modules

import (
	"github.com/salsaflow/salsaflow/modules/release_notes_managers/github"
)

var releaseNotesManagerFactories = map[string]ReleaseNotesManagerFactory{
	github.Id: github.Factory,
}
