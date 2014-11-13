package startCmd

import (
	// Stdlib
	"fmt"
	"os"

	// Internal
	"github.com/salsita/salsaflow/action"
	"github.com/salsita/salsaflow/app"
	"github.com/salsita/salsaflow/errs"
	"github.com/salsita/salsaflow/git"
	"github.com/salsita/salsaflow/log"
	"github.com/salsita/salsaflow/modules"
	"github.com/salsita/salsaflow/version"

	// Other
	"github.com/coreos/go-semver/semver"
	"gopkg.in/tchap/gocli.v1"
)

var Command = &gocli.Command{
	UsageLine: `
  start [-no_fetch] [-next_trunk_version=VERSION]`,
	Short: "start a new release",
	Long: `
  Start a new release, i.e. create the release branch,
  bump the next version number into the trunk branch
  and make the relevant changes in the issue tracker.

  The release branch is always created on top of the trunk branch,
  the issue tracker actions carried out depend on the plugin being used.

  Considering version numbers, the new release branch inherits
  the version that was on the trunk branch. Then a new version is bumped
  into the trunk branch. By default the new version is taken from
  the previous one by auto-incrementing the minor version.
  However, -next_trunk_version can be used to overwrite this behaviour.
	`,
	Action: run,
}

var (
	flagNextTrunk version.Version
	flagNoFetch   bool
)

func init() {
	Command.Flags.Var(&flagNextTrunk, "next_trunk_version", "the next trunk version string")
	Command.Flags.BoolVar(&flagNoFetch, "no_fetch", flagNoFetch, "do not fetch the remote")
}

func run(cmd *gocli.Command, args []string) {
	if len(args) != 0 {
		cmd.Usage()
		os.Exit(2)
	}

	app.InitOrDie()

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
		remote        = gitConfig.RemoteName()
		trunkBranch   = gitConfig.TrunkBranchName()
		releaseBranch = gitConfig.ReleaseBranchName()
	)

	// Fetch the remote repository.
	if !flagNoFetch {
		task := "Fetch the remote repository"
		log.Run(task)
		if err := git.UpdateRemotes(remote); err != nil {
			return errs.NewError(task, err, nil)
		}
	}

	// Make sure that the trunk branch is up to date.
	task := fmt.Sprintf("Make sure that branch '%v' is up to date", trunkBranch)
	if err := git.EnsureBranchSynchronized(trunkBranch, remote); err != nil {
		return errs.NewError(task, err, nil)
	}

	// Make sure that the release branch does not exist.
	task = fmt.Sprintf("Make sure that branch '%v' does not exist", releaseBranch)
	if err := git.EnsureBranchNotExist(releaseBranch, remote); err != nil {
		return errs.NewError(task, err, nil)
	}

	// Get the current trunk version string.
	task = "Get the current trunk version string"
	trunkVersion, err := version.GetByBranch(trunkBranch)
	if err != nil {
		return errs.NewError(task, err, nil)
	}

	// Get the next trunk version (the future release version).
	var nextTrunkVersion *version.Version
	if !flagNextTrunk.Zero() {
		// Make sure the new version is actually incrementing the current one.
		current, err := semver.NewVersion(trunkVersion.String())
		if err != nil {
			panic(err)
		}

		next, err := semver.NewVersion(flagNextTrunk.String())
		if err != nil {
			panic(err)
		}

		if !current.LessThan(*next) {
			return fmt.Errorf("future version string not an increment: %v <= %v", next, current)
		}

		nextTrunkVersion = &flagNextTrunk
	} else {
		nextTrunkVersion = trunkVersion.IncrementMinor()
	}

	// Fetch the stories from the issue tracker.
	tracker, err := modules.GetIssueTracker()
	if err != nil {
		return errs.NewError(task, err, nil)
	}
	release, err := tracker.NextRelease(trunkVersion, nextTrunkVersion)
	if err != nil {
		return errs.NewError(task, err, nil)
	}

	// Prompt the user to confirm the release.
	fmt.Printf(`
You are about to start a new release branch.
The relevant version strings are:

  current release (current trunk version): %v
  future release (next trunk version):     %v

`, trunkVersion, nextTrunkVersion)
	ok, err := release.PromptUserToConfirmStart()
	if err != nil {
		return err
	}
	if !ok {
		fmt.Println("\nYour wish is my command, exiting now!")
		return nil
	}
	fmt.Println()

	// Create the release branch on top of the trunk branch.
	task = fmt.Sprintf("Create branch '%v' on top of branch '%v'", releaseBranch, trunkBranch)
	log.Run(task)
	if err := git.Branch(releaseBranch, trunkBranch); err != nil {
		return errs.NewError(task, err, nil)
	}
	defer func(task string) {
		if err == nil {
			return
		}
		// On error, delete the newly created release branch.
		log.Rollback(task)
		if err := git.Branch("-d", releaseBranch); err != nil {
			errs.LogError("Delete the release branch", err, nil)
		}
	}(task)

	// Commit the next version string into the trunk branch.
	task = fmt.Sprintf(
		"Bump version for the future release (branch '%v' -> %v)", trunkBranch, nextTrunkVersion)
	act, err := version.SetForBranch(nextTrunkVersion, trunkBranch)
	if err != nil {
		return err
	}
	defer func(act action.Action) {
		if err != nil {
			if ex := act.Rollback(); ex != nil {
				errs.Log(ex)
			}
		}
	}(act)

	// Start the release in the issue tracker.
	act, err = release.Start()
	if err != nil {
		return err
	}
	defer func(act action.Action) {
		if err == nil {
			return
		}
		// On error, cancel the release in the issue tracker.
		if err := act.Rollback(); err != nil {
			errs.Log(err)
		}
	}(act)

	// Push the modified branches.
	task = "Push the affected git branches"
	log.Run(task)
	err = git.Push(remote, trunkBranch+":"+trunkBranch, releaseBranch+":"+releaseBranch)
	if err != nil {
		return errs.NewError(task, err, nil)
	}

	return nil
}
