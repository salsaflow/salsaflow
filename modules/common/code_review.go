package common

import (
	// Internal
	"github.com/salsaflow/salsaflow/action"
	"github.com/salsaflow/salsaflow/config/loader"
	"github.com/salsaflow/salsaflow/git"
	"github.com/salsaflow/salsaflow/version"
)

// CodeReviewModule interface --------------------------------------------------

type CodeReviewModule interface {
	loader.Module
	NewCodeReviewTool() (CodeReviewTool, error)
}

// CodeReviewTool interface ----------------------------------------------------

type ReviewContext struct {
	Commit *git.Commit
	Story  Story
}

type CodeReviewTool interface {
	NewRelease(v *version.Version) Release
	PostReviewRequests(ctxs []*ReviewContext, opts map[string]interface{}) error
	PostReviewFollowupMessage() string
}

type Release interface {
	// Initialise is called during `release start`
	// when a new version string is committed into trunk.
	Initialise() (rollback action.Action, err error)

	// The following methods are used during `release deploy`
	// to perform optional cleanup after the release is finished.

	// EnsureClosable makes sure the release can be closed.
	// It returns ErrNotClosable as the root cause in case that is not the case.
	EnsureClosable() error

	// Close closes the given release in the code review tool.
	Close() (rollback action.Action, err error)
}
