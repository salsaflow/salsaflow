package closeCmd

import (
	// Stdlib
	"bytes"
	"fmt"
	"os"

	// Internal
	"github.com/tchap/git-trunk/config"
	"github.com/tchap/git-trunk/git"
	"github.com/tchap/git-trunk/log"
	"github.com/tchap/git-trunk/version"

	// Other
	"github.com/tchap/gocli"
)

var Command = &gocli.Command{
	UsageLine: `
  close`,
	Short: "close the current release",
	Long: `
  Close the release that is currently running. This means that:

    1) Branch 'release' is tagged with its version string.
    2) Branch 'release' is deleted.
    3) Branch 'client' is moved to point to the newly created tag.
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
		taskMsg = "Checkout the original branch"
		log.Run(taskMsg)
		_, ex := git.Checkout(currentBranch)
		if ex != nil {
			log.Fail(taskMsg)
			return
		}
	}()

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

	// Ensure that the remote release branch exists.
	taskMsg = "Ensure that the release branch exists in the remote"
	log.Run(taskMsg)
	exists, stderr, err := git.RemoteBranchExists(config.ReleaseBranch, config.OriginName)
	if err != nil {
		return
	}
	if !exists {
		err = fmt.Errorf("branch %v not found in the remote (%v)",
			config.ReleaseBranch, config.OriginName)
		return
	}

	// Tag the release branch with its version string.
	taskMsg = "Tag the release branch with its version string"
	ver, stderr, err := version.ReadFromBranch(config.ReleaseBranch)
	if err != nil {
		return
	}
	tag := ver.ReleaseTagString()
	stderr, err = git.Tag(tag, config.ReleaseBranch)
	if err != nil {
		return
	}
	defer func() {
		if err != nil {
			// Delete the release tag.
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
	stderr, err = git.CreateOrResetBranch(config.ClientBranch, tag)
	if err != nil {
		return
	}

	// Delete the release branch.
	taskMsg = "Delete the local release branch"
	exists, stderr, err = git.LocalBranchExists(config.ReleaseBranch)
	if err != nil {
		return
	}
	if !exists {
		log.Skip(taskMsg)
		return
	}

	// Push to create the tag, reset client and delete release in the remote repository.
	taskMsg = "Push to create the tag, reset client and delete release"
	toPush := []string{
		"--tags",
		":" + config.ReleaseBranch,
		config.ClientBranch + ":" + config.ClientBranch,
	}
	stderr, err = git.Push(config.OriginName, toPush)
	return
}
