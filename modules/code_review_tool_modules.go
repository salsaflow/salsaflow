package modules

import (
	"github.com/salsaflow/salsaflow/modules/code_review_tools/github"
)

var codeReviewToolFactories = map[string]CodeReviewToolFactory{
	github.Id: github.Factory,
}
