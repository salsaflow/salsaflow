package startCmd

import (
	// Stdlib
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"text/tabwriter"

	// Internal
	"github.com/salsita/SalsaFlow/git-trunk/app"
	"github.com/salsita/SalsaFlow/git-trunk/config"
	"github.com/salsita/SalsaFlow/git-trunk/git"
	"github.com/salsita/SalsaFlow/git-trunk/log"
	"github.com/salsita/SalsaFlow/git-trunk/prompt"
	pt "github.com/salsita/SalsaFlow/git-trunk/utils/pivotaltracker"
	"github.com/salsita/SalsaFlow/git-trunk/version"

	// Other
	"gopkg.in/salsita/go-pivotaltracker.v0/v5/pivotal"
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

	// Fetch the Pivotal Tracker candidate stories.
	msg = "Fetch Pivotal Tracker stories"
	log.Run(msg)
	stories, err := pt.ListReleaseCandidateStories()
	if err != nil {
		return err
	}

	// Exit if there are not candidate stories.
	if len(stories) == 0 {
		msg = ""
		err = errors.New("No candidate stories found in Pivotal Tracker")
		return
	}

	// Check point-me label.
	var (
		pmLabel   = config.PivotalTracker.PointMeLabel()
		pmStories []*pivotal.Story
	)
	for _, story := range stories {
		if pt.StoryLabeled(story, pmLabel) {
			pmStories = append(pmStories, story)
		}
	}
	if len(pmStories) != 0 {
		tw := tabwriter.NewWriter(os.Stdout, 0, 8, 2, '\t', 0)
		fmt.Fprintf(tw, "\nThe following stories are labeled '%v':\n\n", pmLabel)
		io.WriteString(tw, "Story Name\tStory URL\n")
		io.WriteString(tw, "========= \t=========\n")
		for _, story := range stories {
			fmt.Fprintf(tw, "%v\t%v\n", story.Name, story.URL)
		}
		io.WriteString(tw, "\n")
		tw.Flush()

		ok, ex := prompt.Confirm("Are you sure you want to continue?")
		if ex != nil {
			err = ex
			return
		}
		if !ok {
			msg = ""
			err = errors.New("Operation canceled")
			return
		}
	}

	// Prompt the user to confirm the release.
	confirmed, err := prompt.ConfirmStories(
		"The following stories will be included in the next release:",
		stories)
	if err != nil {
		return
	}
	if !confirmed {
		// Don't print the fail message.
		msg = ""
		err = errors.New("Operation canceled")
		return
	}
	fmt.Println()

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

	// Read the trunk version string.
	msg = "Read the current trunk version string"
	log.Run(msg)
	ver, stderr, err := version.ReadFromBranch(config.TrunkBranch)
	if err != nil {
		return
	}

	// Get the future version string.
	var futureVersion *version.Version
	if !flagFuture.Zero() {
		futureVersion = &flagFuture
	} else {
		futureVersion = ver.IncrementMinor()
	}

	// Create release on top of trunk.
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

	// Commit the future version string to the trunk branch.
	msg = "Commit the future version string into the trunk branch"
	log.Run(msg)
	origTrunk, stderr, err := git.Hexsha("refs/heads/" + config.TrunkBranch)
	if err != nil {
		return
	}
	stderr, err = futureVersion.CommitToBranch(config.TrunkBranch)
	if err != nil {
		return
	}
	defer func(taskMsg string) {
		if err != nil {
			log.Rollback(taskMsg)
			out, ex := git.ResetKeep(config.TrunkBranch, origTrunk)
			if ex != nil {
				log.FailWithContext(taskMsg, out)
			}
		}
	}(msg)

	// Add release labels to the relevant stories.
	msg = "Label the stories with the release label"
	log.Run(msg)
	stories, stderr, err = pt.AddLabel(stories, pt.ReleaseLabel(ver.String()))
	if err != nil {
		return
	}
	defer func(taskMsg string) {
		// On error, remove the release labels again.
		if err != nil {
			log.Rollback(taskMsg)
			_, out, ex := pt.RemoveLabel(stories, pt.ReleaseLabel(ver.String()))
			if ex != nil {
				log.FailWithContext(taskMsg, out)
			}
		}
	}(msg)

	// Push trunk and release.
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
