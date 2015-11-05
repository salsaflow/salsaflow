package postCmd

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
	currentBranch, err := git.CurrentBranch()
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

	// Prompt the user to confirm.
	if err := confirmCommits(commits); err != nil {
		return err
	}

	// Make sure the Story-Id tag is there.
	act, changed, err := ensureStoryId(commits)
	if err != nil {
		return err
	}
	defer action.RollbackOnError(&err, act)

	// Push the current branch in case it was modified
	// or it is not up to date at all.
	doPush := changed
	if !doPush {
		// In case the branch was not modified,
		// check whether it is up to date.
		upToDate, err := git.IsBranchSynchronized(currentBranch, remoteName)
		if err != nil {
			return err
		}
		doPush = upToDate
	}
	// Push the branch.
	if doPush {
		if err := push(currentBranch); err != nil {
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
	act, err = postCommitsForReview(commits)
	if err != nil {
		return err
	}
	defer action.Rollback(&err, act)

	// Print the followup dialog.
	return printFollowup()
}
