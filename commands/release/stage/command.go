package stageCmd

import (
	// Stdlib
	"bytes"
	"os"

	// Internal
	"github.com/tchap/git-trunk/app"
	"github.com/tchap/git-trunk/config"
	"github.com/tchap/git-trunk/git"
	"github.com/tchap/git-trunk/log"
	"github.com/tchap/git-trunk/utils/pivotaltracker"
	"github.com/tchap/git-trunk/version"

	// Other
	"github.com/tchap/gocli"
	"gopkg.in/salsita/go-pivotaltracker.v0/v5/pivotal"
)

var Command = &gocli.Command{
	UsageLine: `
  stage`,
	Short: "stage and close the current release",
	Long: `
  Stage and close the release that is currently running. This means that:

    1) The release branch is tagged with its version string.
    2) The release branch is deleted.
    3) The client branch is moved to point to the newly created tag.
    4) Everything is pushed.
	`,
	Action: run,
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
			log.FailWithContext(msg, stderr)
		}

		// Checkout the original branch.
		if currentBranch == "" {
			return
		}
		msg = "Checkout the original branch"
		log.Run(msg)
		out, ex := git.Checkout(currentBranch)
		if ex != nil {
			log.FailWithContext(msg, out)
			return
		}
	}()

	// Fetch the remote repository.
	msg = "Fetch the remote repository"
	log.Run(msg)
	stderr, err = git.UpdateRemotes(config.OriginName)
	if err != nil {
		return
	}

	// Make sure that the local release branch exists.
	msg = "Make sure that the local release branch exists"
	stderr, err = git.CreateTrackingBranchUnlessExists(config.ReleaseBranch, config.OriginName)
	if err != nil {
		return
	}

	// Make sure that the release branch is up to date.
	msg = "Make sure that the release branch is up to date"
	log.Run(msg)
	stderr, err = git.EnsureBranchSynchronized(config.ReleaseBranch, config.OriginName)
	if err != nil {
		return
	}

	// Read the current release version string.
	msg = "Read the current release version"
	ver, stderr, err := version.ReadFromBranch(config.ReleaseBranch)
	if err != nil {
		return
	}

	// Fetch the relevant Pivotal Tracker stories.
	msg = "Fetch Pivotal Tracker stories"
	log.Run(msg)
	stories, err := pivotaltracker.ListReleaseStories(ver.String())
	if err != nil {
		return
	}

	// Make sure that all the stories are reviewed and QA'd.
	msg = "Make sure that all the stories are deliverable"
	log.Run(msg)
	stderr, err = pivotaltracker.ReleaseDeliverable(stories)
	if err != nil {
		return
	}

	// Remember the current branch.
	msg = "Remember the current branch"
	log.Run(msg)
	currentBranch, stderr, err = git.CurrentBranch()
	if err != nil {
		return
	}

	// Tag the release branch with the associated version string.
	msg = "Tag the release branch with the associated version string"
	log.Run(msg)
	tag := ver.ReleaseTagString()
	stderr, err = git.Tag(tag, config.ReleaseBranch)
	if err != nil {
		return
	}
	defer func(taskMsg string) {
		// On error, delete the release tag.
		if err != nil {
			log.Rollback(taskMsg)
			out, ex := git.DeleteTag(tag)
			if ex != nil {
				log.FailWithContext(taskMsg, out)
			}
		}
	}(msg)

	// Reset the client branch to point to the newly created tag.
	msg = "Reset the client branch to point to the release tag"
	log.Run(msg)
	origClient, stderr, err := git.Hexsha("refs/heads/" + config.ClientBranch)
	if err != nil {
		return
	}

	stderr, err = git.CreateOrResetBranch(config.ClientBranch, tag)
	if err != nil {
		return
	}
	defer func(taskMsg string) {
		// On error, reset the client branch back to the original position.
		if err != nil {
			log.Rollback(taskMsg)
			out, ex := git.ResetKeep(config.ClientBranch, origClient)
			if ex != nil {
				log.FailWithContext(taskMsg, out)
			}
		}
	}(msg)

	// Delete the local release branch.
	msg = "Delete the local release branch"
	log.Run(msg)
	stderr, err = git.Branch("-d", config.ReleaseBranch)
	if err != nil {
		return
	}
	defer func(taskMsg string) {
		// On error, re-create the local release branch.
		if err != nil {
			log.Rollback(taskMsg)
			out, ex := git.Branch(
				config.ReleaseBranch, config.OriginName+"/"+config.ReleaseBranch)
			if ex != nil {
				log.FailWithContext(taskMsg, out)
			}
		}
	}(msg)

	// Deliver the stories in Pivotal Tracker.
	msg = "Deliver the stories"
	log.Run(msg)
	stories, stderr, err = pivotaltracker.SetStoriesState(stories, pivotal.StoryStateDelivered)
	if err != nil {
		return
	}
	defer func(taskMsg string) {
		// On error, set the story state back to Finished.
		if err != nil {
			_, out, ex := pivotaltracker.SetStoriesState(
				stories, pivotal.StoryStateFinished)
			if ex != nil {
				log.FailWithContext(taskMsg, out)
			}
		}
	}(msg)

	// Push to create the tag, reset client and delete release in the remote repository.
	msg = "Push to create the tag, reset client and delete release"
	log.Run(msg)
	stderr, err = git.Push(
		config.OriginName,
		"-f", "--tags",
		":"+config.ReleaseBranch,
		config.ClientBranch+":"+config.ClientBranch)
	return
}
