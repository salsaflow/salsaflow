package common

import (
	"github.com/salsita/salsaflow/git"
)

// CodeReviewTool interface ----------------------------------------------------

type CodeReviewTool interface {
	PostReviewRequest(commit *git.Commit, options map[string]interface{}) error
	PrintPostReviewRequestFollowup()
}
