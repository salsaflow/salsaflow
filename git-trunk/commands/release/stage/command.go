package stageCmd

import (
	// Stdlib
	"bytes"
	"errors"
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

	// Remember the current branch.
	msg = "Remember the current branch"
	log.Run(msg)
	currentBranch, stderr, err = git.CurrentBranch()
	if err != nil {
		return
	}

	// Cannot be on the release branch, it will be deleted.
	msg = "Make sure that the release branch is not checked out"
	if currentBranch == config.ReleaseBranch {
		err = errors.New("cannot stage the release while on the release branch")
		return
	}

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

	// Read the current release version.
	msg = "Read the current release version"
	releaseVersion, stderr, err := version.ReadFromBranch(config.ReleaseBranch)
	if err != nil {
		return
	}

	// Instantiate an issue tracker release and ensure it is deliverable.
	msg = "Fetch stories from the issue tracker"
	log.Run(msg)
	release, err := modules.GetIssueTracker().RunningRelease(releaseVersion)
	if err != nil {
		return
	}
	err = release.EnsureDeliverable()
	if err != nil {
		return
	}

	// Tag the release branch with the associated version string.
	msg = "Tag the release branch with the associated version string"
	log.Run(msg)
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
				log.FailWithDetails(taskMsg, out)
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
				log.FailWithDetails(taskMsg, out)
			}
		}
	}(msg)

	// Deliver the release in the issue tracker.
	msg = ""
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
	msg = "Push to create the tag, reset client and delete release"
	log.Run(msg)
	stderr, err = git.Push(
		config.OriginName,
		"-f", "--tags",
		":"+config.ReleaseBranch,
		config.ClientBranch+":"+config.ClientBranch)
	return
}
