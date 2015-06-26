package common

import (
	"github.com/salsaflow/salsaflow/action"
	"github.com/salsaflow/salsaflow/git"
	"github.com/salsaflow/salsaflow/metastore"
	"github.com/salsaflow/salsaflow/version"
)

// CodeReviewTool interface ----------------------------------------------------

type ReviewContext struct {
	Commit        *git.Commit
	ReviewRequest *metastore.Resource
	Story         Story
}

type CodeReviewTool interface {
	// InitialiseRelease is called during `release start`
	// when a new version string is committed into trunk.
	InitialiseRelease(v *version.Version) (rollback action.Action, err error)

	// FinaliseRelease is called during `release stage`
	// when the release is being closed. In can abort the process
	// by returning an error.
	FinaliseRelease(v *version.Version) (rollback action.Action, err error)

	// PostReviewRequests posts review requests for the given review contexts.
	// It is supposed to return an array of metadata that is to be stored on the server.
	PostReviewRequests(ctxs []*ReviewContext, opts map[string]interface{}) ([]*ReviewContext, error)

	// PostReviewFollowupMessage returns the message to be printed to the user
	// once the review requests are posted into the code review tool.
	PostReviewFollowupMessage() string
}
