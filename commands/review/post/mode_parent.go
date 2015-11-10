package postCmd

func postBranch(parentBranch string) (err error) {
	// Load the git-related config.
	gitConfig, err := git.LoadConfig()
	if err != nil {
		return err
	}
	var (
		remoteName  = gitConfig.RemoteName
		trunkBranch = gitConfig.TrunkBranchName
	)

	// Get the current branch name.
	currentBranch, err := gitutil.CurrentBranch()
	if err != nil {
		return err
	}

	if !flagNoFetch {
		// Fetch the remote repository.
		task := "Fetch the remote repository"
		log.Run(task)

		if err := git.UpdateRemotes(remoteName); err != nil {
			return errs.NewError(task, err)
		}
	}

	// Make sure the parent branch is up to date.
	task := fmt.Sprintf("Make sure reference '%v' is up to date", parentBranch)
	log.Run(task)
	if err := git.EnsureBranchSynchronized(parentBranch, remoteName); err != nil {
		return errs.NewError(task, err)
	}

	// Make sure the current branch is up to date.
	task = fmt.Sprintf("Make sure branch '%v' is up to date", currentBranch)
	log.Run(task)
	if err = git.EnsureBranchSynchronized(currentBranch, remoteName); err != nil {
		return errs.NewError(task, err)
	}

	// Get the commits to be posted
	task = "Get the commits to be posted for code review"
	commits, err := git.ShowCommitRange(parentBranch + "..")
	if err != nil {
		return errs.NewError(task, err)
	}

	// Make sure there are no merge commits.
	if err := ensureNoMergeCommits(commits); err != nil {
		return err
	}

	// Prompt the user to confirm.
	if err := confirmCommits(commits); err != nil {
		return err
	}

	// Rebase the current branch on top the parent branch.
	if !flagNoRebase {
		task := fmt.Sprintf("Rebase branch '%v' onto '%v'", currentBranch, parentBranch)
		log.Run(task)
		if err := git.Rebase(parentBranch); err != nil {
			ex := errs.Log(errs.NewError(task, err))
			asciiart.PrintGrimReaper("GIT REBASE FAILED")
			fmt.Printf(`Git failed to rebase your branch onto '%v'.

The repository might have been left in the middle of the rebase process.
In case you do not know how to handle this, just execute

  $ git rebase --abort

to make your repository clean again.

In any case, you have to rebase your current branch onto '%v'
if you want to continue and post a review request. In the edge cases
you can as well use -no_rebase to skip this step, but try not to do it.
`, parentBranch, parentBranch)
			return ex
		}

		// We need to get the commits again, the hash has changed.
		task = "Get the commits to be posted for code review, again"
		commits, err := git.ShowCommitRange(parentBranch + "..")
		if err != nil {
			return errs.NewError(task, err)
		}
	}

	// Ensure the Story-Id tag is there.
	act, err := ensureStoryId(commits)
	if err != nil {
		return err
	}
	defer action.RollbackOnError(&err, act)

	// Merge the current branch into the parent branch unless -no_merge.
	if flagNoMerge {
		// In case the user doesn't want to merge,
		// we need to push the current branch.
		err = push(currentBranch)
	} else {
		// Otherwise we merge the branch into the parent branch
		// and then we push the parent branch itself.
		act, err = mergeDialog(currentBranch, parentBranch)
		if err != nil {
			return err
		}
		defer action.RollbackOnError(&err, act)

		err = push(parentBranch)
	}
	if err != nil {
		return err
	}

	// Post the review requests.
	act, err := postCommitsForReview(commits)
	if err != nil {
		return err
	}
	defer action.RollbackOnError(&err, act)

	// In case there is no error, tell the user they can do next.
	return printFollowup()
}
