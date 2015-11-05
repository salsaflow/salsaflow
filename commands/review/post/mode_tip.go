package postCmd

func postTip() (err error) {
	// Get the commit to be posted
	task := "Get the commit to be posted for code review"
	commits, err := git.ShowCommit("HEAD")
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

	// Push the current branch in case the commit was modified.
	if changed {
		// In case the commit was changed, push in any case.
	} else {
		// Otherwise only push in case the branch is not up to date.
	}

	// In case the commit was changed, reload.
	if changed {
		commits, err = git.ShowCommit("HEAD")
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
