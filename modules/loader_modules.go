package modules

import (
	// Internal
	"github.com/salsaflow/salsaflow/config/loader"
	githubCodeReview "github.com/salsaflow/salsaflow/modules/github/codereview"
	githubReleaseNotes "github.com/salsaflow/salsaflow/modules/github/releasenotes"
	"github.com/salsaflow/salsaflow/modules/jira"
	"github.com/salsaflow/salsaflow/modules/pivotaltracker"
	"github.com/salsaflow/salsaflow/modules/reviewboard"
)

var registeredModules = []loader.Module{
	githubCodeReview.NewModule(),
	githubReleaseNotes.NewModule(),
	jira.NewModule(),
	pivotaltracker.NewModule(),
	reviewboard.NewModule(),
}
