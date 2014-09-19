package startCmd

import (
	// Stdlib
	"bytes"
	"fmt"
	"os"

	// Internal
	"github.com/salsita/SalsaFlow/git-trunk/app"
	"github.com/salsita/SalsaFlow/git-trunk/config"
	"github.com/salsita/SalsaFlow/git-trunk/git"
	"github.com/salsita/SalsaFlow/git-trunk/log"
	"github.com/salsita/SalsaFlow/git-trunk/modules"
	"github.com/salsita/SalsaFlow/git-trunk/version"

	// Other
	"gopkg.in/tchap/gocli.v1"
)

var Command = &gocli.Command{
	UsageLine: `
  start [-future_release=FUTURE]`,
	Short: "start the release branch",
	Long: `
  Start a new release by creating the release branch from the trunk branch.
  More specifically, the steps are:

    1) Get the future release version string, either from the relevant flag
       or read it from package.json on the trunk branch and auto-increment.
    2) Ask the user to confirm the future release version string and the new release.
    1) Create the release branch on top of the trunk branch.
    4) Commit the new version string into the trunk branch so that it is
       prepared for the future release.
    5) Push everything.

  So, the -future_release flag is actually not for the release that is
  about to be started, but for the release after. The release that is
  about to be started reads its version from package.json on that branch.
	`,
	Action: run,
}

var flagFuture version.Version

func init() {
	Command.Flags.Var(&flagFuture, "future_release", "the future version string")
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

func runMain() (err error) {
	var (
		msg           string
		stderr        *bytes.Buffer
		currentBranch string
	)
	defer func() {
		// Print error details.
		if err != nil {
			log.FailWithDetails(msg, stderr)
		}

		// Checkout the original branch.
		if currentBranch == "" {
			return
		}
		msg = "Checkout the original branch"
		log.Run(msg)
		out, ex := git.Checkout(currentBranch)
		if ex != nil {
			log.FailWithDetails(msg, out)
			return
		}
	}()

	// Read the current trunk version string.
	msg = "Read the current trunk version string"
	trunkVersion, stderr, err := version.ReadFromBranch(config.TrunkBranch)
	if err != nil {
		return
	}

	// Get the issue tracker instance and ensure that a new release can be started.
	msg = "Fetch stories from the issue tracker"
	log.Run(msg)
	release, err := modules.GetIssueTracker().NextRelease(trunkVersion)
	if err != nil {
		return
	}
	ok, err := release.PromptUserToConfirmStart()
	if err != nil {
		return
	}
	if !ok {
		fmt.Println("\nYour wish is my command, exiting now...")
		return
	}

	// Remember the current branch.
	msg = "Remember the current branch"
	log.Run(msg)
	currentBranch, stderr, err = git.CurrentBranch()
	if err != nil {
		return
	}

	// Fetch the remote repository.
	msg = "Fetch the remote repository"
	log.Run(msg)
	stderr, err = git.UpdateRemotes(config.OriginName)
	if err != nil {
		return
	}

	// Make sure that release does not exist.
	msg = "Make sure that the release branch does not exist"
	log.Run(msg)
	stderr, err = git.EnsureBranchNotExists(config.ReleaseBranch, config.OriginName)
	if err != nil {
		return
	}

	// Make sure that trunk is up to date.
	msg = "Make sure that the trunk branch is up to date"
	log.Run(msg)
	stderr, err = git.EnsureBranchSynchronized(config.TrunkBranch, config.OriginName)
	if err != nil {
		return
	}

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
		return
	}
	defer func(taskMsg string) {
		if err != nil {
			log.Rollback(taskMsg)
			out, ex := git.Branch("-d", config.ReleaseBranch)
			if ex != nil {
				log.FailWithDetails(taskMsg, out)
			}
		}
	}(msg)

	// Update the trunk version string.
	msg = "Update the trunk version string"
	log.Run(msg)
	origTrunk, stderr, err := git.Hexsha("refs/heads/" + config.TrunkBranch)
	if err != nil {
		return
	}
	stderr, err = nextTrunkVersion.CommitToBranch(config.TrunkBranch)
	if err != nil {
		return
	}
	defer func(taskMsg string) {
		if err != nil {
			log.Rollback(taskMsg)
			out, ex := git.ResetKeep(config.TrunkBranch, origTrunk)
			if ex != nil {
				log.FailWithDetails(taskMsg, out)
			}
		}
	}(msg)

	// Start the release in the issue tracker.
	msg = ""
	action, err := release.Start()
	if err != nil {
		return
	}
	defer func() {
		if err != nil {
			action.Rollback()
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
		return
	}

	return nil
}
