package startCmd

import (
	// Stdlib
	"bytes"
	"errors"
	"fmt"
	"os"

	// Internal
	"github.com/tchap/git-trunk/config"
	"github.com/tchap/git-trunk/git"
	"github.com/tchap/git-trunk/log"
	"github.com/tchap/git-trunk/prompt"
	pt "github.com/tchap/git-trunk/utils/pivotaltracker"
	"github.com/tchap/git-trunk/version"

	// Other
	"github.com/tchap/gocli"
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

	if err := runMain(); err != nil {
		log.Fatalln("\nError: " + err.Error())
	}
}

func runMain() (err error) {
	var (
		taskMsg       string
		stderr        *bytes.Buffer
		currentBranch string
	)
	defer func() {
		// Print error details.
		if err != nil {
			log.FailWithContext(taskMsg, stderr)
		}

		// Checkout the original branch.
		if currentBranch == "" {
			return
		}
		taskMsg = "Checkout the original branch"
		log.Run(taskMsg)
		out, ex := git.Checkout(currentBranch)
		if ex != nil {
			log.FailWithContext(taskMsg, out)
			return
		}
	}()

	// Fetch the Pivotal Tracker candidate stories.
	taskMsg = "Fetch Pivotal Tracker stories"
	stories, err := pt.ListReleaseCandidateStories()
	if err != nil {
		return err
	}

	// Prompt the user to confirm the release.
	confirmed, err := prompt.ConfirmStories(
		"The following stories will be included in the next release:",
		stories)
	if err != nil {
		return
	}
	if !confirmed {
		err = errors.New("Operation canceled")
		return
	}
	fmt.Println()

	// Remember the current branch.
	taskMsg = "Remember the current branch"
	log.Run(taskMsg)
	currentBranch, stderr, err = git.CurrentBranch()
	if err != nil {
		return
	}

	// Fetch the remote repository.
	taskMsg = "Fetch the remote repository"
	log.Run(taskMsg)
	stderr, err = git.UpdateRemotes(config.OriginName)
	if err != nil {
		return
	}

	// Make sure that release does not exist.
	taskMsg = "Ensure that the release branch does not exist"
	log.Run(taskMsg)
	exists, stderr, err := git.BranchExists(config.ReleaseBranch, config.OriginName)
	if err != nil {
		return err
	}
	if exists {
		err = fmt.Errorf("branch %s already exists either locally or remotely",
			config.ReleaseBranch)
		return
	}

	// Make sure that trunk is up to date.
	taskMsg = "Ensure that the trunk branch is up to date"
	log.Run(taskMsg)
	stderr, err = git.EnsureBranchSynchronized(config.TrunkBranch, config.OriginName)
	if err != nil {
		return
	}

	// Read the trunk version string.
	taskMsg = "Read the current trunk version string"
	log.Run(taskMsg)
	ver, stderr, err := version.ReadFromBranch(config.TrunkBranch)
	if err != nil {
		return
	}

	// Get the future version string.
	var futureVersion *version.Version
	if !flagFuture.Zero() {
		futureVersion = &flagFuture
	} else {
		futureVersion = ver.IncrementPatch()
	}

	// Create release on top of trunk.
	taskMsg = "Create the release branch on top of the trunk branch"
	log.Run(taskMsg)
	stderr, err = git.Branch(config.ReleaseBranch, config.TrunkBranch)
	if err != nil {
		return
	}
	defer func() {
		if err != nil {
			msg := "Create the release branch on top of the trunk branch"
			log.Rollback(msg)
			out, ex := git.Branch("-d", config.ReleaseBranch)
			if ex != nil {
				log.FailWithContext(msg, out)
			}
		}
	}()

	// Commit the future version string to the trunk branch.
	taskMsg = "Commit the future version string into the trunk branch"
	log.Run(taskMsg)
	origTrunk, stderr, err := git.Hexsha("refs/heads/" + config.TrunkBranch)
	if err != nil {
		return
	}
	stderr, err = futureVersion.CommitToBranch(config.TrunkBranch)
	if err != nil {
		return
	}
	defer func() {
		if err != nil {
			msg := "Commit the future version string into the trunk branch"
			log.Rollback(msg)
			out, ex := git.ResetKeep(config.TrunkBranch, origTrunk)
			if ex != nil {
				log.FailWithContext(msg, out)
			}
		}
	}()

	// Add release labels to the relevant stories.
	taskMsg = "Label the stories with the release label"
	log.Run(taskMsg)
	stories, stderr, err = pt.AddLabel(stories, pt.ReleaseLabel(ver.String()))
	if err != nil {
		return
	}
	defer func() {
		// On error, remove the release labels again.
		if err != nil {
			msg := "Label the stories with the release label"
			log.Rollback(msg)
			_, out, ex := pt.RemoveLabel(stories, pt.ReleaseLabel(ver.String()))
			if ex != nil {
				log.FailWithContext(msg, out)
			}
		}
	}()

	// Push trunk and release.
	taskMsg = "Push the modified branches"
	log.Run(taskMsg)
	stderr, err = git.Push(
		config.OriginName,
		config.ReleaseBranch+":"+config.ReleaseBranch,
		config.TrunkBranch+":"+config.TrunkBranch)
	if err != nil {
		return
	}

	return nil
}
