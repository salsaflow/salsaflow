package common

import "github.com/salsaflow/salsaflow/git"

// CodeReviewTool interface ----------------------------------------------------

type ReviewContext struct {
	Commit *git.Commit
	Story  Story
}

type CodeReviewTool interface {
	PostReviewRequests(ctxs []*ReviewContext, opts map[string]interface{}) error
	PostReviewFollowupMessage() string
}
