package modules

import (
	// Stdlib
	"bytes"
	"fmt"

	// Internal
	"github.com/salsita/salsaflow/errs"
	"github.com/salsita/salsaflow/modules/common"

	// Internal: modules
	"github.com/salsita/salsaflow/modules/jira"
)

// IssueTracker instantiation --------------------------------------------------

type IssueTrackerFactory func() (common.IssueTracker, error)

var issueTrackerFactories = map[string]IssueTrackerFactory{
	jira.Id: jira.Factory,
}

func GetIssueTracker() (common.IssueTracker, error) {
	// Load configuration.
	config, err := common.LoadConfig()
	if err != nil {
		return nil, err
	}

	// Choose the issue tracker based on the configuration.
	var task = "Instantiate the selected issue tracker plugin"
	id := config.IssueTrackerId()
	factory, ok := issueTrackerFactories[id]
	if !ok {
		// Collect the available tracker ids.
		ids := make([]string, 0, len(issueTrackerFactories))
		for id := range issueTrackerFactories {
			ids = append(ids, id)
		}

		hint := new(bytes.Buffer)
		fmt.Fprintf(hint, "\nAvailable issue trackers: %v\n\n", ids)
		return nil, errs.NewError(task, fmt.Errorf("unknown issue tracker: %v", id), hint)
	}

	// Try to instantiate the issue tracker.
	tracker, err := factory()
	if err != nil {
		return nil, errs.NewError(task, err, nil)
	}

	return tracker, nil
}
