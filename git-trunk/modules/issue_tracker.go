package modules

import (
	// Stdlib
	"fmt"

	// Internal
	"github.com/salsita/SalsaFlow/git-trunk/config"
	"github.com/salsita/SalsaFlow/git-trunk/errors"
	"github.com/salsita/SalsaFlow/git-trunk/log"
	"github.com/salsita/SalsaFlow/git-trunk/modules/common"

	// Internal: modules
	"github.com/salsita/SalsaFlow/git-trunk/modules/pivotaltracker"
)

// IssueTracker instantiation --------------------------------------------------

type IssueTrackerFactory func() (common.IssueTracker, error)

func GetIssueTracker() common.IssueTracker {
	return issueTracker
}

var issueTracker common.IssueTracker

func mustInitIssueTracker() {
	// Register all available issue trackers here.
	factories := map[string]IssueTrackerFactory{
		pivotaltracker.Id: pivotaltracker.Factory,
	}

	// Choose the issue tracker based on the configuration.
	var (
		taskName = "Instantiate the selected issue tracker plugin"
		logger   = log.V(log.Info)
	)
	factory, ok := factories[config.IssueTrackerName]
	if !ok {
		// Collect the available tracker ids.
		ids := make([]string, 0, len(factories))
		for id := range factories {
			ids = append(ids, id)
		}

		// Print the error output into the console.
		logger.Lock()
		defer logger.Unlock()
		logger.Fail(taskName)
		logger.NewLine(
			fmt.Sprintf("(unknown issue tracker: %v)", config.IssueTrackerName))
		logger.NewLine(
			fmt.Sprintf("(available issue trackers: %v)", ids))
		logger.Fatalln("\nError: failed to instantiate the issue tracker plugin")
	}

	// Try to instantiate the issue tracker.
	tracker, err := factory()
	if err != nil {
		errors.NewError(taskName, nil, err).Fatal(logger)
	}

	// Set the global issue tracker instance, at last.
	issueTracker = tracker
}
