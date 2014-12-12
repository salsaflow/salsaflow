package stageCmd

import (
	// Stdlib
	"errors"
	"fmt"
	"os"

	// Internal
	"github.com/salsaflow/salsaflow/action"
	"github.com/salsaflow/salsaflow/app"
	"github.com/salsaflow/salsaflow/changes"
	"github.com/salsaflow/salsaflow/errs"
	"github.com/salsaflow/salsaflow/git"
	"github.com/salsaflow/salsaflow/log"
	"github.com/salsaflow/salsaflow/modules"
	"github.com/salsaflow/salsaflow/modules/common"
	"github.com/salsaflow/salsaflow/prompt"
	"github.com/salsaflow/salsaflow/releases"
	"github.com/salsaflow/salsaflow/version"

	// Other
	"gopkg.in/tchap/gocli.v2"
)

var Command = &gocli.Command{
	UsageLine: `
  stage`,
	Short: "stage and close the current release",
	Long: `
  Stage and close the release that is currently running, i.e.

    1) Make sure the release can be staged (the stories are reviewed and tested)
    2) Tag the release branch and delete it.
    3) Move the staging branch to point to the tag.
    4) Push the changes.
	`,
	Action: run,
}

func run(cmd *gocli.Command, args []string) {
	if len(args) != 0 {
		cmd.Usage()
		os.Exit(2)
	}

	app.InitOrDie()

	defer prompt.RecoverCancel()

	if err := runMain(); err != nil {
		errs.Fatal(err)
	}
}

func runMain() (err error) {
	// Remember the current branch.
	task := "Remember the current branch"
	currentBranch, err := git.CurrentBranch()
	if err != nil {
		return err
	}
	defer func(branch string) {
		// Checkout the original branch on return.
		log.Run(fmt.Sprintf("Checkout the original branch (%v)", branch))
		if ex := git.Checkout(branch); ex != nil {
			if err == nil {
				err = ex
			} else {
				errs.Log(ex)
			}
		}
	}(currentBranch)

	// Load git config.
	config, err := git.LoadConfig()
	if err != nil {
		return err
	}
	var (
		remoteName    = config.RemoteName()
		releaseBranch = config.ReleaseBranchName()
		stagingBranch = config.StagingBranchName()
	)

	// Instantiate the issue tracker.
	tracker, err := modules.GetIssueTracker()
	if err != nil {
		return err
	}

	// Cannot be on the release branch, it will be deleted.
	task = "Make sure that the release branch is not checked out"
	if currentBranch == releaseBranch {
		return errs.NewError(
			task, errors.New("cannot stage the release while on the release branch"), nil)
	}

	// Fetch the remote repository.
	task = "Fetch the remote repository"
	log.Run(task)
	if err := git.UpdateRemotes(remoteName); err != nil {
		return errs.NewError(task, err, nil)
	}

	// Make sure that the local release branch exists.
	task = "Make sure that the local release branch exists"
	if err := git.CreateTrackingBranchUnlessExists(releaseBranch, remoteName); err != nil {
		return errs.NewError(task, err, nil)
	}

	// Make sure that the release branch is up to date.
	task = fmt.Sprintf("Make sure branch '%v' is up to date", releaseBranch)
	log.Run(task)
	if err := git.EnsureBranchSynchronized(releaseBranch, remoteName); err != nil {
		return errs.NewError(task, err, nil)
	}

	// Read the current release version.
	task = "Read the current release version"
	releaseVersion, err := version.GetByBranch(releaseBranch)
	if err != nil {
		return errs.NewError(task, err, nil)
	}

	// Make sure the release is stageable.
	release, err := tracker.RunningRelease(releaseVersion)
	if err != nil {
		return err
	}
	if err := release.EnsureStageable(); err != nil {
		return err
	}

	// Make sure there are no commits being left behind,
	// e.g. make sure no commits are forgotten on the trunk branch,
	// i.e. make sure that everything necessary was cherry-picked.
	if err := checkCommits(release, releaseBranch); err != nil {
		return err
	}

	// Tag the release branch with the associated version string.
	tag := releaseVersion.ReleaseTagString()
	task = fmt.Sprintf("Tag branch '%v' (tag = %v)", releaseBranch, tag)
	log.Run(task)
	tag = releaseVersion.ReleaseTagString()
	if err := git.Tag(tag, releaseBranch); err != nil {
		return errs.NewError(task, err, nil)
	}
	defer func(task string) {
		// On error, delete the release tag.
		if err != nil {
			log.Rollback(task)
			if ex := git.DeleteTag(tag); ex != nil {
				errs.LogError("Delete the release tag", ex, nil)
			}
		}
	}(task)

	// Reset the staging branch to point to the newly created tag.
	task = fmt.Sprintf("Reset branch '%v' to point to tag '%v'", stagingBranch, tag)
	log.Run(task)
	act, err := git.CreateOrResetBranch(stagingBranch, tag)
	if err != nil {
		return errs.NewError(task, err, nil)
	}
	defer func(task string, act action.Action) {
		if err == nil {
			return
		}
		// Rollback on error.
		log.Rollback(task)
		if ex := act.Rollback(); ex != nil {
			errs.Log(ex)
		}
	}(task, act)

	// Delete the local release branch.
	task = fmt.Sprintf("Delete branch '%v'", releaseBranch)
	log.Run(task)
	if err := git.Branch("-d", releaseBranch); err != nil {
		return errs.NewError(task, err, nil)
	}
	defer func(task string) {
		if err == nil {
			return
		}
		// On error, re-create the local release branch.
		log.Rollback(task)
		if ex := git.Branch(releaseBranch, remoteName+"/"+releaseBranch); ex != nil {
			errs.Log(ex)
		}
	}(task)

	// Stage the release in the issue tracker.
	act, err = release.Stage()
	if err != nil {
		return err
	}
	defer func(act action.Action) {
		if err == nil {
			return
		}
		// On error, unstage the release.
		if ex := act.Rollback(); ex != nil {
			errs.Log(ex)
		}
	}(act)

	// Push to create the tag, reset client and delete release in the remote repository.
	task = "Push changes to the remote repository"
	log.Run(task)
	return git.Push(remoteName,
		"-f", "--tags", // Force push, push the release tag.
		":"+releaseBranch,               // Delete the release branch.
		stagingBranch+":"+stagingBranch) // Push the staging branch.
}

func checkCommits(release common.RunningRelease, releaseBranch string) error {
	var task = "Make sure no changes are being left behind"
	log.Run(task)

	stories, err := release.Stories()
	if err != nil {
		return errs.NewError(task, err, nil)
	}

	groups, err := changes.StoryChanges(stories)
	if err != nil {
		return errs.NewError(task, err, nil)
	}

	toCherryPick, err := releases.StoryChangesToCherryPick(groups)
	if err != nil {
		return errs.NewError(task, err, nil)
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
	if err := changes.DumpStoryChanges(toCherryPick, os.Stdout, false); err != nil {
		panic(err)
	}
	fmt.Println()

	confirmed, err := prompt.Confirm("Are you sure you really want to stage the release?")
	if err != nil {
		return errs.NewError(task, err, nil)
	}
	if !confirmed {
		prompt.PanicCancel()
	}
	fmt.Println()

	return nil
}
