package modules

import (
	// Stdlib
	"bytes"
	"fmt"

	// Internal
	"github.com/salsita/salsaflow/config"
	"github.com/salsita/salsaflow/errs"
	"github.com/salsita/salsaflow/log"
	"github.com/salsita/salsaflow/modules/common"

	// Internal: modules
	"github.com/salsita/salsaflow/modules/jira"
	// "github.com/salsita/salsaflow/modules/pivotaltracker" - DISABLED
)

// IssueTracker instantiation --------------------------------------------------

type IssueTrackerFactory func() (common.IssueTracker, error)

func GetIssueTracker() common.IssueTracker {
	return issueTracker
}

var issueTracker common.IssueTracker

func mustInitIssueTracker() {
	var logger = log.V(log.Info)
	if err := initIssueTracker(); err != nil {
		err.LogAndDie(logger)
	}
}

func initIssueTracker() *errs.Error {
	// Register all available issue trackers here.
	factories := map[string]IssueTrackerFactory{
		jira.Id: jira.Factory,
		// pivotaltracker.Id: pivotaltracker.Factory, - DISABLED
	}

	// Choose the issue tracker based on the configuration.
	var task = "Instantiate the selected issue tracker plugin"
	factory, ok := factories[config.IssueTrackerId()]
	if !ok {
		// Collect the available tracker ids.
		ids := make([]string, 0, len(factories))
		for id := range factories {
			ids = append(ids, id)
		}

		hint := new(bytes.Buffer)
		fmt.Fprintf(hint, "\nAvailable issue trackers: %v\n\n", ids)
		return errs.NewError(
			task,
			fmt.Errorf("unknown issue tracker: %v", config.IssueTrackerId()),
			hint)
	}

	// Try to instantiate the issue tracker.
	tracker, err := factory()
	if err != nil {
		return errs.NewError(task, err, nil)
	}

	// Set the global issue tracker instance, at last.
	issueTracker = tracker

	return nil
}
