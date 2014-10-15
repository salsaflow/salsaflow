package reviewboard

import (
	// Stdlib
	"fmt"

	// Internal
	"github.com/salsita/salsaflow/config"
	"github.com/salsita/salsaflow/errs"
	"github.com/salsita/salsaflow/git"
	"github.com/salsita/salsaflow/log"
	"github.com/salsita/salsaflow/modules/common"
	"github.com/salsita/salsaflow/shell"
)

const Id = "review_board"

type codeReviewTool struct{}

func Factory() (common.CodeReviewTool, error) {
	return &codeReviewTool{}, nil
}

func (tool *codeReviewTool) PostReviewRequest(commit *git.Commit) error {
	msg := "Post review request for commit " + commit.SHA
	stdout, stderr, err := shell.Run(
		"rbt", "post", "--guess-fields", "yes", "--branch", commit.StoryId, commit.SHA)
	if err != nil {
		return errs.NewError(msg, stderr, err)
	}
	logger := log.V(log.Info)
	logger.Lock()
	logger.UnsafeOk(msg)
	fmt.Print(stdout)
	logger.Unlock()
	return nil
}

func (tool *codeReviewTool) PrintPostReviewRequestFollowup() {
	log.Println(`
Now, please, take some time to go through all the review requests,
check and annotate them for the reviewers to make them more happy (less sad).

If you find any issues you want to fix right before publishing, fix them now,
amend the relevant commits and use:

  $ rbt post -r <RB request id> <commit SHA>

to update the relevant review request.

When you think the review requests are ready to be published,
publish them in Review Board. Then merge your branch into ` + config.TrunkBranch + ` and push.
`)
}
