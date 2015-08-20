package modules

import (
	"github.com/salsaflow/salsaflow/modules/issue_trackers/pivotaltracker"
)

var issueTrackerFactories = map[string]IssueTrackerFactory{
	pivotaltracker.Id: pivotaltracker.Factory,
}
