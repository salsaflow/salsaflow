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
	"github.com/salsita/salsaflow/modules/reviewboard"
)

// CodeReviewTool instantiation ------------------------------------------------

type CodeReviewToolFactory func() (common.CodeReviewTool, error)

func GetCodeReviewTool() common.CodeReviewTool {
	return codeReviewTool
}

var codeReviewTool common.CodeReviewTool

func mustInitCodeReviewTool() {
	var logger = log.V(log.Info)
	if err := initCodeReviewTool(); err != nil {
		err.LogAndDie(logger)
	}
}

func initCodeReviewTool() *errs.Error {
	// Register all available code review tools here.
	factories := map[string]CodeReviewToolFactory{
		reviewboard.Id: reviewboard.Factory,
	}

	// Choose the code review tool based on the configuration.
	var task = "Instantiate the selected code review plugin"
	factory, ok := factories[config.CodeReviewToolId()]
	if !ok {
		// Collect the available code review tool ids.
		ids := make([]string, 0, len(factories))
		for id := range factories {
			ids = append(ids, id)
		}

		hint := new(bytes.Buffer)
		fmt.Fprintf(hint, "\nAvailable code review tools: %v\n\n", ids)
		return errs.NewError(
			task,
			fmt.Errorf("unknown code review tool: %v", config.CodeReviewToolId()),
			hint)
	}

	// Try to instantiate the code review tool.
	tool, err := factory()
	if err != nil {
		return errs.NewError(task, err, nil)
	}

	// Set the global code review tool instance, at last.
	codeReviewTool = tool

	return nil
}
