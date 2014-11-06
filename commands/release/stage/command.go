package stageCmd

import (
	// Stdlib
	"bytes"
	"errors"
	"os"

	// Internal
	"github.com/salsita/salsaflow/app"
	"github.com/salsita/salsaflow/config"
	"github.com/salsita/salsaflow/git"
	"github.com/salsita/salsaflow/log"
	"github.com/salsita/salsaflow/modules"
	"github.com/salsita/salsaflow/version"

	// Other
	"gopkg.in/tchap/gocli.v1"
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
		task          string
		stderr        *bytes.Buffer
		currentBranch string
	)
	defer func() {
		// Print error details.
		if err != nil {
			log.FailWithDetails(task, stderr)
		}

		// Checkout the original branch.
		if currentBranch == "" {
			return
		}
		task = "Checkout the original branch"
		log.Run(task)
		out, ex := git.Checkout(currentBranch)
		if ex != nil {
			log.FailWithDetails(task, out)
			return
		}
	}()

	// Remember the current branch.
	task = "Remember the current branch"
	log.Run(task)
	currentBranch, stderr, err = git.CurrentBranch()
	if err != nil {
		return
	}

	// Cannot be on the release branch, it will be deleted.
	task = "Make sure that the release branch is not checked out"
	if currentBranch == config.ReleaseBranch {
		err = errors.New("cannot stage the release while on the release branch")
		return
	}

	// Fetch the remote repository.
	task = "Fetch the remote repository"
	log.Run(task)
	stderr, err = git.UpdateRemotes(config.OriginName)
	if err != nil {
		return
	}

	// Make sure that the local release branch exists.
	task = "Make sure that the local release branch exists"
	stderr, err = git.CreateTrackingBranchUnlessExists(config.ReleaseBranch, config.OriginName)
	if err != nil {
		return
	}

	// Make sure that the release branch is up to date.
	task = "Make sure that the release branch is up to date"
	log.Run(task)
	stderr, err = git.EnsureBranchSynchronized(config.ReleaseBranch, config.OriginName)
	if err != nil {
		return
	}

	// Read the current release version.
	task = "Read the current release version"
	releaseVersion, stderr, err := version.ReadFromBranch(config.ReleaseBranch)
	if err != nil {
		return
	}

	// Instantiate an issue tracker release and ensure it is deliverable.
	task = "Fetch stories from the issue tracker"
	log.Run(task)
	release, err := modules.GetIssueTracker().RunningRelease(releaseVersion)
	if err != nil {
		return
	}
	err = release.EnsureDeliverable()
	if err != nil {
		return
	}

	// Tag the release branch with the associated version string.
	task = "Tag the release branch with the associated version string"
	log.Run(task)
	tag := releaseVersion.ReleaseTagString()
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
				log.FailWithDetails(taskMsg, out)
			}
		}
	}(task)

	// Reset the client branch to point to the newly created tag.
	task = "Reset the client branch to point to the release tag"
	log.Run(task)
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
				log.FailWithDetails(taskMsg, out)
			}
		}
	}(task)

	// Delete the local release branch.
	task = "Delete the local release branch"
	log.Run(task)
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
				log.FailWithDetails(taskMsg, out)
			}
		}
	}(task)

	// Deliver the release in the issue tracker.
	task = ""
	action, err := release.Deliver()
	if err != nil {
		return
	}
	defer func() {
		if err != nil {
			action.Rollback()
		}
	}()

	// Push to create the tag, reset client and delete release in the remote repository.
	task = "Push to create the tag, reset client and delete release"
	log.Run(task)
	stderr, err = git.Push(
		config.OriginName,
		"-f", "--tags",
		":"+config.ReleaseBranch,
		config.ClientBranch+":"+config.ClientBranch)
	return
}
