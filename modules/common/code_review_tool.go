package common

import (
	"github.com/salsaflow/salsaflow/action"
	"github.com/salsaflow/salsaflow/git"
	metaclient "github.com/salsaflow/salsaflow/metastore/client"
	"github.com/salsaflow/salsaflow/version"
)

// CodeReviewTool interface ----------------------------------------------------

type ReviewContext struct {
	Commit *git.Commit
	Meta   *metaclient.CommitData
	Story  Story
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
	// It is supposed to return new array that contains the review metadata as well.
	PostReviewRequests(ctxs []*ReviewContext, opts map[string]interface{}) (map[string]interface{}, error)
	PostReviewFollowupMessage() string
}
