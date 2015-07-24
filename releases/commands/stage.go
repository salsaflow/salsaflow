package commands

import (
	// Stdlib
	"fmt"
	"os"

	// Internal
	"github.com/salsaflow/salsaflow/action"
	"github.com/salsaflow/salsaflow/changes"
	"github.com/salsaflow/salsaflow/errs"
	"github.com/salsaflow/salsaflow/git"
	"github.com/salsaflow/salsaflow/log"
	"github.com/salsaflow/salsaflow/modules"
	"github.com/salsaflow/salsaflow/modules/common"
	"github.com/salsaflow/salsaflow/prompt"
	"github.com/salsaflow/salsaflow/releases"
	"github.com/salsaflow/salsaflow/version"
)

type StageOptions struct {
	SkipFetch bool
}

var DefaultStageOptions = &StageOptions{
	SkipFetch: false,
}

func Stage(options *StageOptions) (act action.Action, err error) {
	// Rollback machinery.
	chain := action.NewActionChain()
	defer chain.RollbackOnError(&err)

	// Make sure opts are not nil.
	if options == nil {
		options = DefaultStageOptions
	}

	// Load git config.
	gitConfig, err := git.LoadConfig()
	if err != nil {
		return nil, err
	}
	var (
		remoteName    = gitConfig.RemoteName()
		releaseBranch = gitConfig.ReleaseBranchName()
		stagingBranch = gitConfig.StagingBranchName()
	)

	// Instantiate the issue tracker.
	tracker, err := modules.GetIssueTracker()
	if err != nil {
		return nil, err
	}

	// Get the current branch.
	task := "Get the current branch"
	currentBranch, err := git.CurrentBranch()
	if err != nil {
		return nil, err
	}

	// Cannot be on the release branch, it will be deleted.
	task = fmt.Sprintf("Make sure that branch '%v' is not checked out", releaseBranch)
	if currentBranch == releaseBranch {
		return nil, errs.NewError(
			task, fmt.Errorf("cannot stage the release while on branch '%v'", releaseBranch))
	}

	// Fetch the remote repository.
	if !options.SkipFetch {
		task = "Fetch the remote repository"
		log.Run(task)
		if err := git.UpdateRemotes(remoteName); err != nil {
			return nil, errs.NewError(task, err)
		}
	}

	// Make sure that the local release branch exists and is up to date.
	task = fmt.Sprintf("Make sure that branch '%v' is up to date", releaseBranch)
	log.Run(task)
	if err := git.CheckOrCreateTrackingBranch(releaseBranch, remoteName); err != nil {
		return nil, errs.NewError(task, err)
	}

	// Read the current release version.
	task = "Read the current release version"
	releaseVersion, err := version.GetByBranch(releaseBranch)
	if err != nil {
		return nil, errs.NewError(task, err)
	}

	// Make sure the release is stageable.
	release, err := tracker.RunningRelease(releaseVersion)
	if err != nil {
		return nil, err
	}
	if err := release.EnsureStageable(); err != nil {
		return nil, err
	}

	// Make sure there are no commits being left behind,
	// e.g. make sure no commits are forgotten on the trunk branch,
	// i.e. make sure that everything necessary was cherry-picked.
	if err := checkCommits(tracker, release, releaseBranch); err != nil {
		return nil, err
	}

	// Reset the staging branch to point to the newly created tag.
	task = fmt.Sprintf("Reset branch '%v' to point to branch '%v'", stagingBranch, releaseBranch)
	log.Run(task)
	act, err = git.CreateOrResetBranch(stagingBranch, releaseBranch)
	if err != nil {
		return nil, errs.NewError(task, err)
	}
	chain.PushTask(task, act)

	// Delete the local release branch.
	task = fmt.Sprintf("Delete branch '%v'", releaseBranch)
	log.Run(task)
	if err := git.Branch("-D", releaseBranch); err != nil {
		return nil, errs.NewError(task, err)
	}
	chain.PushTask(task, action.ActionFunc(func() error {
		task := fmt.Sprintf("Recreate branch '%v'", releaseBranch)

		// In case the release branch exists locally, do nothing.
		// This might look like an extra and useless check, but it looks like
		// the final git push at the end of the command function actually creates
		// the release branch locally when it is aborted from the pre-push hook.
		// Not sure why and how that is happening.
		exists, err := git.LocalBranchExists(releaseBranch)
		if err != nil {
			return errs.NewError(task, err)
		}
		if exists {
			return nil
		}

		// In case the branch indeed does not exist, create it.
		if err := git.Branch(releaseBranch, remoteName+"/"+releaseBranch); err != nil {
			return errs.NewError(task, err)
		}
		return nil
	}))

	// Update the version string on the staging branch.
	stagingVersion, err := releaseVersion.ToStageVersion()
	if err != nil {
		return nil, err
	}

	task = fmt.Sprintf("Bump version (branch '%v' -> %v)", stagingBranch, stagingVersion)
	log.Run(task)
	act, err = version.SetForBranch(stagingVersion, stagingBranch)
	if err != nil {
		return nil, errs.NewError(task, err)
	}
	chain.PushTask(task, act)

	// Finalise the release in the code review tool.
	codeReviewTool, err := modules.GetCodeReviewTool()
	if err != nil {
		return nil, err
	}
	act, err = codeReviewTool.FinaliseRelease(releaseVersion)
	if err != nil {
		return nil, err
	}
	// No need to pass any task string, the module rollback functions
	// are expected to take care of printing messages on their own.
	chain.Push(act)

	// Stage the release in the issue tracker.
	act, err = release.Stage()
	if err != nil {
		return nil, err
	}
	chain.Push(act)

	// Push to create the tag, reset client and delete release in the remote repository.
	task = "Push changes to the remote repository"
	log.Run(task)
	err = git.Push(remoteName,
		"-f",                            // Use the Force, Luke.
		":"+releaseBranch,               // Delete the release branch.
		stagingBranch+":"+stagingBranch) // Push the staging branch.
	if err != nil {
		return nil, err
	}
	return chain, nil
}

func checkCommits(
	tracker common.IssueTracker,
	release common.RunningRelease,
	releaseBranch string,
) error {

	var task = "Make sure no changes are being left behind"
	log.Run(task)

	stories, err := release.Stories()
	if err != nil {
		return errs.NewError(task, err)
	}
	if len(stories) == 0 {
		return nil
	}

	groups, err := changes.StoryChanges(stories)
	if err != nil {
		return errs.NewError(task, err)
	}

	toCherryPick, err := releases.StoryChangesToCherryPick(groups)
	if err != nil {
		return errs.NewError(task, err)
	}

	// In case there are some changes being left behind,
	// ask the user to confirm whether to proceed or not.
	if len(toCherryPick) == 0 {
		return nil
	}

	fmt.Println(`
Some changes are being left behind!

In other words, some changes that are assigned to the current release
have not been cherry-picked onto the release branch yet.
	`)
	if err := changes.DumpStoryChanges(os.Stdout, toCherryPick, tracker, false); err != nil {
		panic(err)
	}
	fmt.Println()

	confirmed, err := prompt.Confirm("Are you sure you really want to stage the release?", false)
	if err != nil {
		return errs.NewError(task, err)
	}
	if !confirmed {
		return prompt.ErrCanceled
	}
	fmt.Println()

	return nil
}
