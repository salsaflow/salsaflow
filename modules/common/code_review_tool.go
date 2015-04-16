package common

import "github.com/salsaflow/salsaflow/git"

// CodeReviewTool interface ----------------------------------------------------

type CommitReviewContext struct {
	Commit *git.Commit
	Story  Story
}

type CodeReviewTool interface {
	PostReviewRequestForCommit(ctx *CommitReviewContext, opts map[string]interface{}) error
	PostReviewRequestForBranch(branch string, ctxs []*CommitReviewContext, opts map[string]interface{}) error
	PostReviewFollowupMessage() string
}
