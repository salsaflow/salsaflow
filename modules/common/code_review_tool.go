package common

import (
	"github.com/salsita/salsaflow/git"
)

// CodeReviewTool interface ----------------------------------------------------

type CodeReviewTool interface {
	PostReviewRequest(commit *git.Commit) error
	PrintPostReviewRequestFollowup()
}
