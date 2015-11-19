package postCmd

import (
	// Stdlib
	"fmt"

	// Internal
	"github.com/salsaflow/salsaflow/errs"
	"github.com/salsaflow/salsaflow/git"
	"github.com/salsaflow/salsaflow/git/gitutil"
)

func postTip() (err error) {
	// Load Git-related config.
	gitConfig, err := git.LoadConfig()
	if err != nil {
		return err
	}
	var (
		remoteName = gitConfig.RemoteName
	)

	// Get the current branch.
	currentBranch, err := gitutil.CurrentBranch()
	if err != nil {
		return err
	}

	// Get the commit to be posted
	task := "Get the commit to be posted for code review"
	commits, err := git.ShowCommit(currentBranch)
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

	// Prompt the user to confirm.
	if err := promptUserToConfirmCommits(commits); err != nil {
		return err
	}

	// Make sure the Story-Id tag is there.
	commits, changed, err := ensureStoryId(commits)
	if err != nil {
		return err
	}

	// Push the current branch if necessary.
	doPush := changed
	if !doPush {
		// Check whether the remote branch actually exists.
		task := fmt.Sprintf(
			"Make sure branch '%v' exists in remote '%v'", currentBranch, remoteName)
		exists, err := git.RemoteBranchExists(currentBranch, remoteName)
		if err != nil {
			return errs.NewError(task, err)
		}
		doPush = !exists
	}
	if !doPush {
		// In case the branch was not modified and it exists remotely,
		// check whether it is up to date.
		task := fmt.Sprintf("Check whether branch '%v' is up to date", currentBranch)
		upToDate, err := git.IsBranchSynchronized(currentBranch, remoteName)
		if err != nil {
			return errs.NewError(task, err)
		}
		doPush = !upToDate
	}
	// Push the branch.
	if doPush {
		if err := push(remoteName, currentBranch); err != nil {
			return err
		}
	}

	// In case the commit was changed, reload.
	if changed {
		commits, err = git.ShowCommit(currentBranch)
		if err != nil {
			return err
		}
	}

	// Post the commit for review.
	if err := postCommitsForReview(commits); err != nil {
		return err
	}

	// Print the followup dialog.
	return printFollowup()
}
