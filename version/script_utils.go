package version

import (
	// Stdlib
	"fmt"

	// Internal
	"github.com/salsaflow/salsaflow/action"
	"github.com/salsaflow/salsaflow/errs"
	"github.com/salsaflow/salsaflow/git"
	"github.com/salsaflow/salsaflow/log"
)

func GetByBranch(branch string) (ver *Version, err error) {
	// Remember the current branch.
	currentBranch, err := git.CurrentBranch()
	if err != nil {
		return nil, err
	}

	// Checkout the target branch.
	if err := git.Checkout(branch); err != nil {
		return nil, err
	}
	defer func() {
		// Checkout the original branch on return.
		if ex := git.Checkout(currentBranch); ex != nil {
			if err == nil {
				err = ex
			} else {
				errs.Log(ex)
			}
		}
	}()

	// Get the version.
	return Get()
}

func SetForBranch(ver *Version, branch string) (act action.Action, err error) {
	var mainTask = fmt.Sprintf("Bump version to %v for branch '%v'", ver, branch)

	// Make sure the repository is clean (don't check untracked files).
	task := "Make sure the repository is clean"
	if err := git.EnsureCleanWorkingTree(false); err != nil {
		return nil, errs.NewError(task, err, nil)
	}

	// Remember the current branch.
	currentBranch, err := git.CurrentBranch()
	if err != nil {
		return nil, err
	}

	// Remember the current position of the target branch.
	task = fmt.Sprintf("Remember the position of branch '%v'", branch)
	originalPosition, err := git.Hexsha("refs/heads/" + branch)
	if err != nil {
		return nil, errs.NewError(task, err, nil)
	}

	// Checkout the target branch.
	task = fmt.Sprintf("Checkout branch '%v'", branch)
	if err := git.Checkout(branch); err != nil {
		return nil, errs.NewError(task, err, nil)
	}
	defer func() {
		// Checkout the original branch on return.
		task := fmt.Sprintf("Checkout branch '%v'", currentBranch)
		if ex := git.Checkout(currentBranch); ex != nil {
			if err == nil {
				err = ex
			} else {
				errs.LogError(task, ex, nil)
			}
		}
	}()

	// Set the project version to the desired value.
	if err := Set(ver); err != nil {
		return nil, err
	}

	// Commit changes.
	_, err = git.RunCommand("commit", "-a",
		"-m", fmt.Sprintf("Bump version to %v", ver),
		"-m", fmt.Sprintf("Story-Id: %v", git.StoryIdUnassignedTagValue))
	if err != nil {
		task := "Reset the working tree to the original state"
		if err := git.Reset("--keep"); err != nil {
			errs.LogError(task, err, nil)
		}
		return nil, err
	}

	return action.ActionFunc(func() (err error) {
		// On rollback, reset the target branch to the original position.
		log.Rollback(mainTask)
		return git.ResetKeep(branch, originalPosition)
	}), nil
}
