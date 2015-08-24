package modules

import (
	// Stdlib
	"fmt"

	// Internal
	"github.com/salsaflow/salsaflow/errs"
	"github.com/salsaflow/salsaflow/modules/common"

	// Internal: modules
	github "github.com/salsaflow/salsaflow/modules/github/codereview"
	"github.com/salsaflow/salsaflow/modules/reviewboard"
)

// CodeReviewTool instantiation ------------------------------------------------

var codeReviewToolFactories = map[string]common.CodeReviewToolFactory{
	github.Id:      github.NewFactory(),
	reviewboard.Id: reviewboard.NewFactory(),
}

func AvailableCodeReviewToolKeys() []string {
	keys := make([]string, 0, len(codeReviewToolFactories))
	for key := range codeReviewToolFactories {
		keys = append(keys, key)
	}
	return keys
}

func GetCodeReviewToolFactory(id string) (common.CodeReviewToolFactory, error) {
	// Choose the code review tool based on the configuration.
	var task = "Instantiate the selected code review plugin"
	factory, ok := codeReviewToolFactories[id]
	if !ok {
		// Collect the available code review tool ids.
		ids := make([]string, 0, len(codeReviewToolFactories))
		for id := range codeReviewToolFactories {
			ids = append(ids, id)
		}

		hint := fmt.Sprintf("\nAvailable code review tools: %v\n\n", ids)
		return nil, errs.NewErrorWithHint(
			task, fmt.Errorf("unknown code review tool: '%v'", id), hint)
	}

	return factory, nil
}

func GetCodeReviewTool() (common.CodeReviewTool, error) {
	// Load configuration.
	config, err := common.LoadConfig()
	if err != nil && config == nil {
		return nil, err
	}

	// Get the factory.
	factory, err := GetCodeReviewToolFactory(config.CodeReviewToolId())
	if err != nil {
		return nil, err
	}

	// Return a new module instance.
	return factory.NewCodeReviewTool()
}
