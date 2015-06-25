package common

import (
	"github.com/salsaflow/salsaflow/action"
	"github.com/salsaflow/salsaflow/git"
	"github.com/salsaflow/salsaflow/metastore"
	"github.com/salsaflow/salsaflow/version"
)

// CodeReviewTool interface ----------------------------------------------------

type ReviewContext struct {
	Commit *git.Commit
	Meta   *metastore.CommitData
	Story  Story
}

type Map map[string]interface{}

type CodeReviewTool interface {
	// InitialiseRelease is called during `release start`
	// when a new version string is committed into trunk.
	InitialiseRelease(v *version.Version) (rollback action.Action, err error)

	// FinaliseRelease is called during `release stage`
	// when the release is being closed. In can abort the process
	// by returning an error.
	FinaliseRelease(v *version.Version) (rollback action.Action, err error)

	// PostReviewRequests posts review requests for the given review contexts.
	// It is supposed to return an array of metadata that is to be stored on the server
	// such as res[i] is to be associated with ctxs[i].
	PostReviewRequests(ctxs []*ReviewContext, opts Map) (res []*metastore.Resource, err error)

	// PostReviewFollowupMessage returns the message to be printed to the user
	// once the review requests are posted into the code review tool.
	PostReviewFollowupMessage() string
}
