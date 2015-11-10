package postCmd

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
	task := "Make sure the chosen commit is valid"
	if isStoryIdMissing(commits) {
		return errs.NewError(task, errors.New("Story-Id tag is missing"))
	}

	// Prompt the user to confirm.
	if err := promptUserToConfirmCommits(commits); err != nil {
		return err
	}

	// Post the review requests, in this case it will be only one.
	act, err := postCommitsForReview(commits)
	if err != nil {
		return err
	}
	defer action.RollbackOnError(&err, act)

	// In case there is no error, tell the user they can do next.
	return printFollowup()
}
