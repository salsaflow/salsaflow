package stageCmd

import (
	// Stdlib
	"bytes"
	"os"

	// Internal
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

	// Fetch the remote repository.
	taskMsg = "Fetch the remote repository"
	log.Run(taskMsg)
	stderr, err = git.UpdateRemotes(config.OriginName)
	if err != nil {
		return
	}

	// Ensure that the release branch is in sync.
	taskMsg = "Ensure that the release branch is synchronized"
	log.Run(taskMsg)
	stderr, err = git.EnsureBranchSynchronized(config.ReleaseBranch, config.OriginName)
	if err != nil {
		return
	}

	// Read the current release version string.
	taskMsg = "Read the current release version"
	ver, stderr, err := version.ReadFromBranch(config.ReleaseBranch)
	if err != nil {
		return
	}

	// Fetch the relevant Pivotal Tracker stories.
	taskMsg = "Fetch Pivotal Tracker stories"
	log.Run(taskMsg)
	stories, err := pivotaltracker.ListReleaseStories(ver.String())
	if err != nil {
		return
	}

	// Make sure that all the stories are reviewed and QA'd.
	taskMsg = "Make sure that all the stories are deliverable"
	log.Run(taskMsg)
	stderr, err = pivotaltracker.ReleaseDeliverable(stories)
	if err != nil {
		return
	}

	// Remember the current branch.
	taskMsg = "Remember the current branch"
	log.Run(taskMsg)
	currentBranch, stderr, err = git.CurrentBranch()
	if err != nil {
		return
	}

	// Tag the release branch with its version string.
	taskMsg = "Tag the release branch with its version string"
	log.Run(taskMsg)
	tag := ver.ReleaseTagString()
	stderr, err = git.Tag(tag, config.ReleaseBranch)
	if err != nil {
		return
	}
	defer func() {
		// On error, delete the release tag.
		if err != nil {
			msg := "Tag the release branch with its version string"
			log.Rollback(msg)
			out, ex := git.DeleteTag(tag)
			if ex != nil {
				log.FailWithContext(msg, out)
			}
		}
	}()

	// Reset the client branch to point to the newly created tag.
	taskMsg = "Reset the client branch to point to the release tag"
	log.Run(taskMsg)
	origClient, stderr, err := git.Hexsha("refs/heads/" + config.ClientBranch)
	if err != nil {
		return
	}

	stderr, err = git.CreateOrResetBranch(config.ClientBranch, tag)
	if err != nil {
		return
	}
	defer func() {
		// On error, reset the client branch back to the original position.
		if err != nil {
			msg := "Reset the client branch to point to the release tag"
			log.Rollback(msg)
			out, ex := git.ResetKeep(config.ClientBranch, origClient)
			if ex != nil {
				log.FailWithContext(msg, out)
			}
		}
	}()

	// Delete the local release branch.
	taskMsg = "Delete the local release branch"
	stderr, err = git.Branch("-d", config.ReleaseBranch)
	if err != nil {
		return
	}
	defer func() {
		// On error, re-create the local release branch.
		if err != nil {
			msg := "Delete the local release branch"
			log.Rollback(msg)
			out, ex := git.Branch(
				config.ReleaseBranch, config.OriginName+"/"+config.ReleaseBranch)
			if ex != nil {
				log.FailWithContext(msg, out)
			}
		}
	}()

	// Deliver the stories in Pivotal Tracker.
	taskMsg = "Deliver the stories"
	stderr, err = pivotaltracker.SetStoriesState(stories, pivotal.StoryStateDelivered)
	if err != nil {
		return
	}
	defer func() {
		// On error, set the story state back to Finished.
		if err != nil {
			msg := "Deliver the stories"
			out, ex := pivotaltracker.SetStoriesState(
				stories, pivotal.StoryStateFinished)
			if ex != nil {
				log.FailWithContext(msg, out)
			}
		}
	}()

	// Push to create the tag, reset client and delete release in the remote repository.
	taskMsg = "Push to create the tag, reset client and delete release"
	log.Run(taskMsg)
	toPush := []string{
		"-f", // for the client branch
		"--tags",
		":" + config.ReleaseBranch,
		config.ClientBranch + ":" + config.ClientBranch,
	}
	stderr, err = git.Push(config.OriginName, toPush)
	return
}
