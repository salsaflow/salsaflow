package modules

import (
	// Stdlib
	"fmt"

	// Internal
	"github.com/salsaflow/salsaflow/errs"
	"github.com/salsaflow/salsaflow/modules/common"
)

// ReleaseNotesManager instantiation --------------------------------------------------

type ReleaseNotesManagerFactory func() (common.ReleaseNotesManager, error)

func AvailableReleaseNotesManagerKeys() []string {
	keys := make([]string, 0, len(releaseNotesManagerFactories))
	for key := range releaseNotesManagerFactories {
		keys = append(keys, key)
	}
	return keys
}

func GetReleaseNotesManager() (common.ReleaseNotesManager, error) {
	// Load configuration.
	config, err := common.LoadConfig()
	if err != nil && config == nil {
		return nil, err
	}

	// Choose the release notes manager based on the configuration.
	var task = "Instantiate the selected release notes manager plugin"
	id := config.ReleaseNotesManagerId()
	// In case the id is not set, we simply return nil.
	// This means that this module is disabled.
	if id == "" {
		return nil, nil
	}
	factory, ok := releaseNotesManagerFactories[id]
	if !ok {
		// Collect the available tracker ids.
		ids := make([]string, 0, len(releaseNotesManagerFactories))
		for id := range releaseNotesManagerFactories {
			ids = append(ids, id)
		}

		hint := fmt.Sprintf("\nAvailable release notes managers: %v\n\n", ids)
		return nil, errs.NewErrorWithHint(
			task, fmt.Errorf("unknown release notes manager: '%v'", id), hint)
	}

	// Try to instantiate the release notes manager.
	rnm, err := factory()
	if err != nil {
		return nil, errs.NewError(task, err)
	}

	return rnm, nil
}
