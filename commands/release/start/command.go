package startCmd

import (
	// Stdlib
	"bytes"
	"fmt"
	"os"

	// Internal
	"github.com/salsita/salsaflow/app"
	"github.com/salsita/salsaflow/config"
	"github.com/salsita/salsaflow/errs"
	"github.com/salsita/salsaflow/git"
	"github.com/salsita/salsaflow/log"
	"github.com/salsita/salsaflow/modules"
	"github.com/salsita/salsaflow/version"

	// Other
	"gopkg.in/tchap/gocli.v1"
)

var Command = &gocli.Command{
	UsageLine: `
  start [-next_trunk_version=VERSION]`,
	Short: "start a new release",
	Long: `
  Start a new release by creating the release branch from the trunk branch
  and making the relevant modifications in the issue tracker. More specifically,
  the steps are:

    1) Get the next trunk version string, either from the relevant flag
       or read it from package.json on the trunk branch and auto-increment.
    2) Ask the user to confirm the next trunk version string and the new release.
    3) Create the release branch on top of the trunk branch.
    4) Commit the new version string into the trunk branch so that it is
       prepared for the future release (the release after the one being started).
	5) Start the release in the issue tracker.
    6) Push everything.

  So, the -next_trunk_version flag is actually not affecting the release that is
  about to be started, but the release after. The release that is about to be
  started reads its version from the current trunk's package.json. This version
  string is not modified during the execution of this command.
	`,
	Action: run,
}

var flagFuture version.Version

func init() {
	Command.Flags.Var(&flagFuture, "next_trunk_version", "the next trunk version string")
}

func run(cmd *gocli.Command, args []string) {
	if len(args) != 0 {
		cmd.Usage()
		os.Exit(2)
	}

	app.MustInit()

	if err := runMain(); err != nil {
		log.Fatalln("\nError: " + err.Error())
	}
}

func handleError(task string, err error, stderr *bytes.Buffer) error {
	errs.NewError(task, stderr, err).Log(log.V(log.Info))
	return err
}

func runMain() (err error) {
	// Fetch the remote repository.
	msg := "Fetch the remote repository"
	log.Run(msg)
	stderr, err := git.UpdateRemotes(config.OriginName)
	if err != nil {
		return handleError(msg, err, stderr)
	}

	// Make sure that the trunk branch is up to date.
	msg = "Make sure that the trunk branch is up to date"
	log.Run(msg)
	stderr, err = git.EnsureBranchSynchronized(config.TrunkBranch, config.OriginName)
	if err != nil {
		return handleError(msg, err, stderr)
	}

	// Make sure that the release branch does not exist.
	msg = "Make sure that the release branch does not exist"
	log.Run(msg)
	stderr, err = git.EnsureBranchNotExists(config.ReleaseBranch, config.OriginName)
	if err != nil {
		return handleError(msg, err, stderr)
	}

	// Read the current trunk version string.
	msg = "Read the current trunk version string"
	trunkVersion, stderr, err := version.ReadFromBranch(config.TrunkBranch)
	if err != nil {
		return handleError(msg, err, stderr)
	}

	// Fetch the stories from the issue tracker.
	msg = "Fetch the stories from the issue tracker"
	log.Run(msg)
	release, err := modules.GetIssueTracker().NextRelease(trunkVersion)
	if err != nil {
		return handleError(msg, err, nil)
	}

	// Prompt the user to confirm the release.
	msg = ""
	ok, err := release.PromptUserToConfirmStart()
	if err != nil {
		return errs.LogAndReturn(err)
	}
	if !ok {
		fmt.Println("\nYour wish is my command, exiting now...")
		return nil
	}
	fmt.Println()

	// Remember the current branch.
	msg = "Remember the current branch"
	currentBranch, stderr, err := git.CurrentBranch()
	if err != nil {
		return handleError(msg, err, stderr)
	}
	defer func() {
		// Checkout the original branch on exit.
		msg := "Checkout the original branch"
		log.Run(msg)
		if stderr, err := git.Checkout(currentBranch); err != nil {
			handleError(msg, err, stderr)
		}
	}()

	// Get the next trunk version (the future release version).
	var nextTrunkVersion *version.Version
	if !flagFuture.Zero() {
		nextTrunkVersion = &flagFuture
	} else {
		nextTrunkVersion = trunkVersion.IncrementMinor()
	}

	// Create the release branch on top of the trunk branch.
	msg = "Create the release branch on top of the trunk branch"
	log.Run(msg)
	stderr, err = git.Branch(config.ReleaseBranch, config.TrunkBranch)
	if err != nil {
		return handleError(msg, err, stderr)
	}
	defer func(taskMsg string) {
		if err != nil {
			// On error, delete the newly created release branch.
			log.Rollback(taskMsg)
			if stderr, err := git.Branch("-d", config.ReleaseBranch); err != nil {
				handleError("Delete the release branch", err, stderr)
			}
		}
	}(msg)

	// Update the trunk version string.
	msg = "Update the trunk version string"
	log.Run(msg)
	origTrunk, stderr, err := git.Hexsha("refs/heads/" + config.TrunkBranch)
	if err != nil {
		return handleError(msg, err, stderr)
	}
	stderr, err = nextTrunkVersion.CommitToBranch(config.TrunkBranch)
	if err != nil {
		return handleError(msg, err, stderr)
	}
	defer func(taskMsg string) {
		if err != nil {
			// On error, reset the trunk branch to point to the original position.
			log.Rollback(taskMsg)
			if stderr, err := git.ResetKeep(config.TrunkBranch, origTrunk); err != nil {
				handleError("Reset the trunk branch to the original position", err, stderr)
			}
		}
	}(msg)

	// Start the release in the issue tracker.
	msg = ""
	action, err := release.Start()
	if err != nil {
		return errs.LogAndReturn(err)
	}
	defer func() {
		if err != nil {
			// On error, cancel the release in the issue tracker.
			if err := action.Rollback(); err != nil {
				handleError("Cancel the release in the issue tracker", err, nil)
			}
		}
	}()

	// Push the modified branches.
	msg = "Push the modified branches"
	log.Run(msg)
	stderr, err = git.Push(
		config.OriginName,
		config.ReleaseBranch+":"+config.ReleaseBranch,
		config.TrunkBranch+":"+config.TrunkBranch)
	if err != nil {
		return handleError(msg, err, stderr)
	}

	return nil
}
