package cherrypickCmd

import (
	// Stdlib
	"fmt"
	"os"

	// Internal
	"github.com/salsaflow/salsaflow/app"
	"github.com/salsaflow/salsaflow/app/appflags"
	"github.com/salsaflow/salsaflow/errs"
	"github.com/salsaflow/salsaflow/git"
	"github.com/salsaflow/salsaflow/git/gitutil"
	"github.com/salsaflow/salsaflow/log"
	"github.com/salsaflow/salsaflow/prompt"

	// Other
	"gopkg.in/tchap/gocli.v2"
)

var Command = &gocli.Command{
	UsageLine: "cherry-pick [-fetch] [-target=TARGET] COMMIT...",
	Short:     "cherry-pick commits into target branch",
	Long: `
  Use git cherry-pick to copy the specified commits into the target branch,
  which is the release branch by default. So this command is useful when
  you need to get some changes from trunk to the release branch. But it can
  be used for any cherry-picking when a custom target branch is specified.

  This command makes sure that the target branch is up to date,
  but it does not fetch the repository by default. Use -fetch to
  update the repository before doing the check.
	`,
	Action: run,
}

var (
	flagFetch  bool
	flagTarget string
)

func init() {
	// Register flags.
	Command.Flags.BoolVar(&flagFetch, "fetch", flagFetch,
		"update the project remote before proceeding futher")
	Command.Flags.StringVar(&flagTarget, "target", flagTarget,
		"the branch to cherry-pick commits into")

	// Register global flags.
	appflags.RegisterGlobalFlags(&Command.Flags)
}

func run(cmd *gocli.Command, args []string) {
	if len(args) == 0 {
		cmd.Usage()
		os.Exit(2)
	}

	app.InitOrDie()

	defer prompt.RecoverCancel()

	if err := runMain(args); err != nil {
		errs.Fatal(err)
	}
}

func runMain(args []string) (err error) {
	// Get the commit hashes.
	// We need to do this before we checkout the target branch,
	// because relative refs like HEAD change by doing so.
	hashes, err := git.RevisionsToCommitList(args...)
	if err != nil {
		return err
	}

	// Load git-related config.
	gitConfig, err := git.LoadConfig()
	if err != nil {
		return err
	}
	var (
		remoteName    = gitConfig.RemoteName
		releaseBranch = gitConfig.ReleaseBranchName
	)

	// Get the target branch name.
	targetBranch := flagTarget
	if targetBranch == "" {
		targetBranch = releaseBranch
	}

	// Fetch the remote repository if requested.
	if flagFetch {
		task := "Fetch the remote repository"
		log.Run(task)
		if err := git.UpdateRemotes(remoteName); err != nil {
			return errs.NewError(task, err)
		}
	}

	// Make sure the target branch is up to date.
	if err := ensureTargetBranchExists(targetBranch, remoteName); err != nil {
		return err
	}

	// Get the current branch name.
	currentBranch, err := gitutil.CurrentBranch()
	if err != nil {
		return err
	}

	// Checkout the target branch in case we are not at it already.
	if currentBranch != targetBranch {
		task := fmt.Sprintf("Checkout branch '%v'", targetBranch)
		log.Run(task)
		if err := git.Checkout(targetBranch); err != nil {
			return errs.NewError(task, err)
		}
		// Checkout the current branch on return, unless there is an error.
		// In that case we want to stay on the target branch.
		defer func() {
			if err == nil {
				task := fmt.Sprintf("Checkout branch '%v'", currentBranch)
				log.Run(task)
				if ex := git.Checkout(currentBranch); ex != nil {
					err = errs.NewError(task, ex)
				}
			} else {
				log.Warn(fmt.Sprintf("An error detected, staying on branch '%v'", targetBranch))
			}
		}()
	}

	// Run git cherry-pick.
	task := fmt.Sprintf("Cherry-pick the chosen commits into '%v'", targetBranch)
	log.Run(task)
	if err := git.CherryPick(hashes...); err != nil {
		return errs.NewError(task, err)
	}

	log.Warn(fmt.Sprintf("Make sure to push branch '%v' to publish the changes", targetBranch))
	return nil
}

func ensureTargetBranchExists(branch, remote string) error {
	task := fmt.Sprintf("Check whether branch '%v' exists", branch)

	// In case the branch exists locally, we are done.
	localExists, err := git.LocalBranchExists(branch)
	if err != nil {
		return errs.NewError(task, err)
	}
	if localExists {
		return nil
	}

	// Otherwise try to create a tracking branch.
	remoteExists, err := git.RemoteBranchExists(branch, remote)
	if err != nil {
		return errs.NewError(task, err)
	}
	if remoteExists {
		return errs.Wrap(task, git.CreateTrackingBranch(branch, remote))
	}

	// We have failed!
	return &git.ErrRefNotFound{branch}
}
