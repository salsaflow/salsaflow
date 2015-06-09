package common

import (
	"github.com/salsaflow/salsaflow/action"
	"github.com/salsaflow/salsaflow/git"
	"github.com/salsaflow/salsaflow/version"
)

// CodeReviewTool interface ----------------------------------------------------

type ReviewContext struct {
	Commit *git.Commit
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

	PostReviewRequests(ctxs []*ReviewContext, opts map[string]interface{}) error
	PostReviewFollowupMessage() string
}
