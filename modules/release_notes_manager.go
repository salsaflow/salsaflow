package modules

import (
	// Stdlib
	"fmt"

	// Internal
	"github.com/salsaflow/salsaflow/errs"
	"github.com/salsaflow/salsaflow/modules/common"

	// Internal: modules
	github "github.com/salsaflow/salsaflow/modules/github/releasenotes"
)

// ReleaseNotesManager instantiation --------------------------------------------------

var notesManagerFactories = map[string]common.ReleaseNotesManagerFactory{
	github.Id: github.NewFactory(),
}

func AvailableReleaseNotesManagerKeys() []string {
	keys := make([]string, 0, len(notesManagerFactories))
	for key := range notesManagerFactories {
		keys = append(keys, key)
	}
	return keys
}

func GetReleaseNotesManagerFactory(id string) (common.ReleaseNotesManagerFactory, error) {
	// Choose the release notes manager based on the configuration.
	var task = "Instantiate the selected release notes manager plugin"
	// In case the id is not set, we simply return nil.
	// This means that this module is disabled.
	if id == "" {
		return nil, nil
	}
	factory, ok := notesManagerFactories[id]
	if !ok {
		// Collect the available tracker ids.
		ids := make([]string, 0, len(notesManagerFactories))
		for id := range notesManagerFactories {
			ids = append(ids, id)
		}

		hint := fmt.Sprintf("\nAvailable release notes managers: %v\n\n", ids)
		return nil, errs.NewErrorWithHint(
			task, fmt.Errorf("unknown release notes manager: '%v'", id), hint)
	}

	return factory, nil
}

func GetReleaseNotesManager() (common.ReleaseNotesManager, error) {
	// Load configuration.
	config, err := common.LoadConfig()
	if err != nil && config == nil {
		return nil, err
	}

	// Get the factory.
	factory, err := GetReleaseNotesManagerFactory(config.ReleaseNotesManagerId())
	if err != nil {
		return nil, err
	}
	if factory == nil {
		return nil, nil
	}

	// Return a new module instance.
	return factory.NewReleaseNotesManager()
}
