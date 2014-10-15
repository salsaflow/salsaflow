package reviewboard

import (
	// Stdlib
	"bytes"
	"errors"
	"fmt"
	"regexp"
	"strconv"

	// Internal
	"github.com/salsita/salsaflow/config"
	"github.com/salsita/salsaflow/errs"
	"github.com/salsita/salsaflow/git"
	"github.com/salsita/salsaflow/log"
	"github.com/salsita/salsaflow/modules/common"
	"github.com/salsita/salsaflow/repo"
	"github.com/salsita/salsaflow/shell"
)

const Id = "review_board"

type codeReviewTool struct{}

func Factory() (common.CodeReviewTool, error) {
	repo.AddInitHook(ensureRbtVersion)
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

func ensureRbtVersion() error {
	msg := "Check the RBTools version being used"
	log.Run(msg)

	// rbt prints the version string to stderr. WHY? Know knows...
	_, stderr, err := shell.Run("rbt", "--version")
	if err != nil {
		return errs.NewError(msg, stderr, err)
	}

	pattern := regexp.MustCompile("^RBTools (([0-9]+)[.]([0-9]+).*)")
	parts := pattern.FindStringSubmatch(stderr.String())
	if len(parts) != 4 {
		err := errors.New("failed to parse 'rbt --version' output: " + stderr.String())
		return errs.NewError(msg, nil, err)
	}
	rbtVersion := parts[1]
	// No need to check errors, we know the format is correct.
	major, _ := strconv.Atoi(parts[2])
	minor, _ := strconv.Atoi(parts[3])

	if !(major == 0 && minor == 6) {
		hint := `
You need RBTools version 0.6. Please run

  $ pip install rbtool~=0.6 --allow-external rbtool --allow-unverified rbtool

to install the correct version.

`
		return errs.NewError(
			msg,
			bytes.NewBufferString(hint),
			errors.New("unsupported rbt version detected: "+rbtVersion))
	}

	return nil
}
