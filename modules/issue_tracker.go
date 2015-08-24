package modules

import (
	// Stdlib
	"fmt"

	// Internal
	"github.com/salsaflow/salsaflow/errs"
	"github.com/salsaflow/salsaflow/modules/common"

	// Internal: modules
	"github.com/salsaflow/salsaflow/modules/jira"
	"github.com/salsaflow/salsaflow/modules/pivotaltracker"
)

// IssueTracker instantiation --------------------------------------------------

var issueTrackerFactories = map[string]common.IssueTrackerFactory{
	jira.Id:           jira.NewFactory(),
	pivotaltracker.Id: pivotaltracker.NewFactory(),
}

func AvailableIssueTrackerKeys() []string {
	keys := make([]string, 0, len(issueTrackerFactories))
	for key := range issueTrackerFactories {
		keys = append(keys, key)
	}
	return keys
}

func GetIssueTrackerFactory(id string) (common.IssueTrackerFactory, error) {
	// Choose the issue tracker based on the configuration.
	var task = "Instantiate the selected issue tracker plugin"
	factory, ok := issueTrackerFactories[id]
	if !ok {
		// Collect the available tracker ids.
		ids := make([]string, 0, len(issueTrackerFactories))
		for id := range issueTrackerFactories {
			ids = append(ids, id)
		}

		hint := fmt.Sprintf("\nAvailable issue trackers: %v\n\n", ids)
		return nil, errs.NewErrorWithHint(
			task, fmt.Errorf("unknown issue tracker: '%v'", id), hint)
	}

	return factory, nil
}

func GetIssueTracker() (common.IssueTracker, error) {
	// Load configuration.
	config, err := common.LoadConfig()
	if err != nil && config == nil {
		return nil, err
	}

	// Get the factory.
	factory, err := GetIssueTrackerFactory(config.IssueTrackerId())
	if err != nil {
		return nil, err
	}

	// Return a new module instance.
	return factory.NewIssueTracker()
}
