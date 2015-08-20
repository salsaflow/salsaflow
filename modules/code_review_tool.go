package modules

import (
	// Stdlib
	"fmt"

	// Internal
	"github.com/salsaflow/salsaflow/errs"
	"github.com/salsaflow/salsaflow/modules/common"
)

// CodeReviewTool instantiation ------------------------------------------------

type CodeReviewToolFactory func() (common.CodeReviewTool, error)

func AvailableCodeReviewToolKeys() []string {
	keys := make([]string, 0, len(codeReviewToolFactories))
	for key := range codeReviewToolFactories {
		keys = append(keys, key)
	}
	return keys
}

func GetCodeReviewTool() (common.CodeReviewTool, error) {
	// Load configuration.
	config, err := common.LoadConfig()
	if err != nil && config == nil {
		return nil, err
	}

	// Choose the code review tool based on the configuration.
	var task = "Instantiate the selected code review plugin"
	id := config.CodeReviewToolId()
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

	// Try to instantiate the code review tool.
	tool, err := factory()
	if err != nil {
		return nil, errs.NewError(task, err)
	}

	return tool, nil
}
