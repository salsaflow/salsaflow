package postCmd

import (
	// Stdlib
	"errors"
	"fmt"

	// Internal
	"github.com/salsaflow/salsaflow/errs"
	"github.com/salsaflow/salsaflow/git"
)

func postRevision(revision string) (err error) {
	// Get the commit to be posted
	task := "Get the commit to be posted for code review"
	commits, err := git.ShowCommit(revision)
	if err != nil {
		return errs.NewError(task, err)
	}

	// Assert that things are consistent.
	if numCommits := len(commits); numCommits != 1 {
		panic(fmt.Sprintf("len(commits): expected 1, got %v", numCommits))
	}

	// Make sure the commit is not a merge commit.
	if err := ensureNoMergeCommits(commits); err != nil {
		return err
	}

	// Make sure the Story-Id tag is not missing.
	task = "Make sure the chosen commit is valid"
	if _, missing := isStoryIdMissing(commits); missing {
		return errs.NewError(task, errors.New("Story-Id tag is missing"))
	}

	// Prompt the user to confirm.
	if err := promptUserToConfirmCommits(commits); err != nil {
		return err
	}

	// Post the review requests, in this case it will be only one.
	if err := postCommitsForReview(commits); err != nil {
		return err
	}

	// In case there is no error, tell the user they can do next.
	return printFollowup()
}
