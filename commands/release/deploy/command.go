package deployCmd

import (
	// Stdlib
	"fmt"
	"os"

	// Internal
	"github.com/salsaflow/salsaflow/action"
	"github.com/salsaflow/salsaflow/app"
	"github.com/salsaflow/salsaflow/errs"
	"github.com/salsaflow/salsaflow/git"
	"github.com/salsaflow/salsaflow/log"
	"github.com/salsaflow/salsaflow/modules"
	"github.com/salsaflow/salsaflow/prompt"
	"github.com/salsaflow/salsaflow/version"

	// Other
	"gopkg.in/tchap/gocli.v2"
)

var Command = &gocli.Command{
	UsageLine: "deploy",
	Short:     "deploy the current staging environment into production",
	Long: `
  Deploy the current staging environment into production.

  This basically means that:

    1) The issue tracker is checked to make sure the release
       is accepted and that it can be actually released.
    2) The stable branch is reset to point to the staging branch.
    3) Version is bumped for the stable branch.
    4) The stable branch is tagged with a release tag.
    5) The staging branch is reset to the current release branch
       in case there is already another release started.
    6) Everything is pushed to the remote repository.
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
	// Load repo config.
	gitConfig, err := git.LoadConfig()
	if err != nil {
		return err
	}

	var (
		remoteName    = gitConfig.RemoteName()
		releaseBranch = gitConfig.ReleaseBranchName()
		stagingBranch = gitConfig.StagingBranchName()
		stableBranch  = gitConfig.StableBranchName()
	)

	// Make sure the stable branch exists.
	task := fmt.Sprintf("Make sure branch '%v' exists", stableBranch)
	if err := git.CreateTrackingBranchUnlessExists(stableBranch, remoteName); err != nil {
		return errs.NewError(task, err, nil)
	}

	// Make sure we are not on the stable branch.
	task = fmt.Sprintf("Make sure we are not on branch '%v'", stableBranch)
	currentBranch, err := git.CurrentBranch()
	if err != nil {
		return errs.NewError(task, err, nil)
	}

	if currentBranch == stableBranch {
		err := fmt.Errorf("cannot deploy while on branch '%v'", stableBranch)
		return errs.NewError(task, err, nil)
	}

	// Make sure the current staging branch is releasable.
	task = "Make sure the staging branch is releasable"
	log.Run(task)
	tracker, err := modules.GetIssueTracker()
	if err != nil {
		return errs.NewError(task, err, nil)
	}

	stagingVersion, err := version.GetByBranch(stagingBranch)
	if err != nil {
		return err
		return errs.NewError(task, err, nil)
	}

	release, err := tracker.RunningRelease(stagingVersion)
	if err != nil {
		return errs.NewError(task, err, nil)
	}

	if err := release.EnsureReleasable(); err != nil {
		return errs.NewError(task, err, nil)
	}

	// Reset the stable branch to point to stage.
	task = fmt.Sprintf("Reset branch '%v' to point to branch '%v'", stableBranch, stagingBranch)
	log.Run(task)
	act, err := git.CreateOrResetBranch(stableBranch, stagingBranch)
	if err != nil {
		return errs.NewError(task, err, nil)
	}
	defer action.RollbackTaskOnError(&err, task, act)

	// Bump version for the stable branch.
	stableVersion, err := stagingVersion.ToStableVersion()
	if err != nil {
		return err
	}

	task = fmt.Sprintf("Bump version (branch '%v' -> %v)", stableBranch, stableVersion)
	log.Run(task)
	act, err = version.SetForBranch(stableVersion, stableBranch)
	if err != nil {
		return errs.NewError(task, err, nil)
	}
	defer action.RollbackTaskOnError(&err, task, act)

	// Tag the stable branch.
	tag := stableVersion.ReleaseTagString()
	task = fmt.Sprintf("Tag branch '%v' with tag '%v'", stableBranch, tag)
	log.Run(task)
	if err := git.Tag(tag, stableBranch); err != nil {
		return errs.NewError(task, err, nil)
	}
	defer action.RollbackTaskOnError(&err, task, action.ActionFunc(func() error {
		task := fmt.Sprintf("Delete tag '%v'", tag)
		if err := git.Tag("-d", tag); err != nil {
			return errs.NewError(task, err, nil)
		}
		return nil
	}))

	// Reset stage to point to the release branch in case that one exists
	// so that the next release is immediately available for the client.
	toPush := []string{"--tags"}

	exists, err := git.RemoteBranchExists(releaseBranch, remoteName)
	if err != nil {
		return errs.NewError(task, err, nil)
	}
	if exists {
		task = fmt.Sprintf(
			"Reset branch '%v' to point to branch '%v'", stagingBranch, releaseBranch)
		log.Run(task)

		remoteReleaseBranch := fmt.Sprintf("%v/%v", remoteName, releaseBranch)
		act, err = git.CreateOrResetBranch(stagingBranch, remoteReleaseBranch)
		if err != nil {
			return errs.NewError(task, err, nil)
		}
		defer action.RollbackTaskOnError(&err, task, act)

		toPush = append(toPush, fmt.Sprintf("%v:%v", stagingBranch, stagingBranch))
	}

	// Push the changes to the remote repository.
	task = "Push changes to the remote repository"
	log.Run(task)

	toPush = append(toPush, fmt.Sprintf("%v:%v", stableBranch, stableBranch))
	if err := git.PushForce(remoteName, toPush...); err != nil {
		return errs.NewError(task, err, nil)
	}
	return nil
}
