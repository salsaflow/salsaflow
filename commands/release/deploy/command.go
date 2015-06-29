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
	"github.com/salsaflow/salsaflow/modules/common"
	"github.com/salsaflow/salsaflow/prompt"
	"github.com/salsaflow/salsaflow/releases/commands"
	"github.com/salsaflow/salsaflow/version"

	// Other
	"gopkg.in/tchap/gocli.v2"
)

var Command = &gocli.Command{
	UsageLine: "deploy [-no_fetch]",
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

var flagNoFetch bool

func init() {
	Command.Flags.BoolVar(&flagNoFetch, "no_fetch", flagNoFetch,
		"do not fetch the remote repository")
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
		stagingBranch = gitConfig.StagingBranchName()
		stableBranch  = gitConfig.StableBranchName()
	)

	// Fetch the repository.
	if !flagNoFetch {
		if err := git.UpdateRemotes(remoteName); err != nil {
			return err
		}
	}

	// Check branches.
	checkBranch := func(branchName string) error {
		// Make sure the branch exists.
		task := fmt.Sprintf("Make sure that branch '%v' exists and is up to date", branchName)
		if err := git.CheckOrCreateTrackingBranch(branchName, remoteName); err != nil {
			return errs.NewError(task, err)
		}

		// Make sure we are not on the branch.
		task = fmt.Sprintf("Make sure that branch '%v' is not checked out", branchName)
		currentBranch, err := git.CurrentBranch()
		if err != nil {
			return errs.NewError(task, err)
		}
		if currentBranch == branchName {
			err := fmt.Errorf("cannot deploy while on branch '%v'", branchName)
			return errs.NewError(task, err)
		}
		return nil
	}

	for _, branch := range []string{stableBranch, stagingBranch} {
		if err := checkBranch(branch); err != nil {
			return err
		}
	}

	// Make sure the current staging branch can be released.
	task := fmt.Sprintf("Make sure that branch '%v' can be released", stagingBranch)
	log.Run(task)
	tracker, err := modules.GetIssueTracker()
	if err != nil {
		return errs.NewError(task, err)
	}

	stagingVersion, err := version.GetByBranch(stagingBranch)
	if err != nil {
		return errs.NewError(task, err)
	}

	release, err := tracker.RunningRelease(stagingVersion)
	if err != nil {
		return err
	}

	if err := release.EnsureReleasable(); err != nil {
		return err
	}

	// Reset the stable branch to point to stage.
	task = fmt.Sprintf("Reset branch '%v' to point to branch '%v'", stableBranch, stagingBranch)
	log.Run(task)
	act, err := git.CreateOrResetBranch(stableBranch, stagingBranch)
	if err != nil {
		return errs.NewError(task, err)
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
		return errs.NewError(task, err)
	}
	defer action.RollbackTaskOnError(&err, task, act)

	// Tag the stable branch.
	tag := stableVersion.ReleaseTagString()
	task = fmt.Sprintf("Tag branch '%v' with tag '%v'", stableBranch, tag)
	log.Run(task)
	if err := git.Tag(tag, stableBranch); err != nil {
		return errs.NewError(task, err)
	}
	defer action.RollbackTaskOnError(&err, task, action.ActionFunc(func() error {
		task := fmt.Sprintf("Delete tag '%v'", tag)
		if err := git.Tag("-d", tag); err != nil {
			return errs.NewError(task, err)
		}
		return nil
	}))

	// Try to reset the staging branch to the release branch
	// in case the release branch is started already.
	// This basically means that we want to run `release stage`.
	log.Log("Trying to stage the next release for acceptance")
	act, err = commands.Stage(&commands.StageOptions{
		SkipFetch: true,
	})
	if err != nil {
		// Not terribly pretty, but it works. We just handle the known errors and continue.
		// It is ok when the release branch does not exist yet or the release cannot be staged
		// in the issue tracker.
		rootCause := errs.RootCause(err)
		if ex, ok := rootCause.(*git.ErrRefNotFound); ok {
			log.Log(fmt.Sprintf("Git reference '%v' not found, staging canceled", ex.Ref()))
		} else if rootCause == common.ErrNotStageable {
			log.Log("The next release cannot be staged yet, skipping ...")
		} else {
			return err
		}
	}
	defer action.RollbackOnError(&err, act)

	// Push the changes to the remote repository.
	task = "Push changes to the remote repository"
	log.Run(task)
	toPush := []string{
		"--tags",
		fmt.Sprintf("%v:%v", stableBranch, stableBranch),
	}
	if err := git.PushForce(remoteName, toPush...); err != nil {
		return errs.NewError(task, err)
	}
	return nil
}
