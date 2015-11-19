package noop

import (
	// Internal
	"github.com/salsaflow/salsaflow/action"
	"github.com/salsaflow/salsaflow/log"
	"github.com/salsaflow/salsaflow/modules/common"
	"github.com/salsaflow/salsaflow/version"
)

func newCodeReviewTool() (common.CodeReviewTool, error) {
	return &codeReviewTool{}, nil
}

type codeReviewTool struct{}

func (tool *codeReviewTool) InitialiseRelease(v *version.Version) (action.Action, error) {
	return action.Noop, nil
}

func (tool *codeReviewTool) FinaliseRelease(v *version.Version) (action.Action, error) {
	return action.Noop, nil
}

func (tool *codeReviewTool) PostReviewRequests(
	ctxs []*common.ReviewContext,
	opts map[string]interface{},
) error {
	log.Log("NOOP code review module active, not doing anything")
	return nil
}

func (tool *codeReviewTool) PostReviewFollowupMessage() string {
	return `
No review requests created, this is a NOOP code review module after all!
`
}
