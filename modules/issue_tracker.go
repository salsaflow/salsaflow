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
	"github.com/salsita/salsaflow/modules/pivotaltracker"
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
		err.Fatal(logger)
	}
}

func initIssueTracker() *errs.Error {
	// Register all available issue trackers here.
	factories := map[string]IssueTrackerFactory{
		jira.Id:           jira.Factory,
		pivotaltracker.Id: pivotaltracker.Factory,
	}

	// Choose the issue tracker based on the configuration.
	var (
		taskName = "Instantiate the selected issue tracker plugin"
	)
	factory, ok := factories[config.IssueTrackerName]
	if !ok {
		// Collect the available tracker ids.
		ids := make([]string, 0, len(factories))
		for id := range factories {
			ids = append(ids, id)
		}

		var b bytes.Buffer
		fmt.Fprintf(&b, "(unknown issue tracker: %v)", config.IssueTrackerName)
		fmt.Fprintf(&b, "(available issue trackers: %v)", ids)
		fmt.Fprintf(&b, "\nError: failed to instantiate the issue tracker plugin")
		return errs.NewError(taskName, &b, nil)
	}

	// Try to instantiate the issue tracker.
	tracker, err := factory()
	if err != nil {
		return errs.NewError(taskName, nil, err)
	}

	// Set the global issue tracker instance, at last.
	issueTracker = tracker

	return nil
}
