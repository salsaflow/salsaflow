package common

import (
	"github.com/salsaflow/salsaflow/git"
)

// CodeReviewTool interface ----------------------------------------------------

type CodeReviewTool interface {
	PostReviewRequest(commit *git.Commit, options map[string]interface{}) error
}
