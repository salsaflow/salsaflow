package main

import (
	// Stdlib
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"

	// Internal
	"github.com/salsita/salsaflow/asciiart"
	"github.com/salsita/salsaflow/errs"
	"github.com/salsita/salsaflow/git"
	"github.com/salsita/salsaflow/hooks"
	"github.com/salsita/salsaflow/log"
)

const zeroHash = "0000000000000000000000000000000000000000"

func main() {
	// Set up the identification command line flag.
	hooks.IdentifyYourself()

	// Tell the user what is happening.
	fmt.Println("---> Running the SalsaFlow pre-push hook")

	// The hook is always invoked as `pre-push <remote-name> <push-url>`.
	if len(os.Args) != 3 {
		fmt.Fprintf(os.Stderr, "Usage: %v <remote-name> <push-url>\n", os.Args[0])
		errs.Fatal(fmt.Errorf("invalid arguments: %#v\n", os.Args[1:]))
	}

	// Run the main function.
	if err := run(os.Args[1], os.Args[2]); err != nil {
		asciiart.PrintGrimReaper("PUSH ABORTED")
		errs.Fatal(err)
	}
}

func run(remoteName, pushURL string) error {
	// Load the git-related SalsaFlow config.
	gitConfig, err := git.LoadConfig()
	if err != nil {
		return err
	}

	// Only check the project remote.
	if remoteName != gitConfig.RemoteName() {
		log.Log(
			fmt.Sprintf(
				"Not pushing to the main project remote (%v), check skipped",
				gitConfig.RemoteName()))
		return nil
	}

	// The commits that are being pushed are listed on stdin.
	// The format is <local ref> <local sha1> <remote ref> <remote sha1>,
	// so we parse the input and collect all the local hexshas.
	var coreRefs = []string{
		"refs/heads/" + gitConfig.TrunkBranchName(),
		"refs/heads/" + gitConfig.ReleaseBranchName(),
		"refs/heads/" + gitConfig.StagingBranchName(),
		"refs/heads/" + gitConfig.StableBranchName(),
	}

	task := "Parse the hook input"
	var revRanges []string
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		var (
			line  = scanner.Text()
			parts = strings.Split(line, " ")
		)
		if len(parts) != 4 {
			return errs.NewError(task, errors.New("invalid input line: "+line), nil)
		}

		localSha, remoteRef, remoteSha := parts[1], parts[2], parts[3]

		// Skip the refs that are being deleted.
		if localSha == zeroHash {
			continue
		}

		// Check only updates to the core branches,
		// i.e. trunk, release, client or master.
		var isCoreBranch bool
		for _, ref := range coreRefs {
			if remoteRef == ref {
				isCoreBranch = true
			}
		}
		if !isCoreBranch {
			continue
		}

		log.Log(fmt.Sprintf("Checking commits updating remote reference '%s'", remoteRef))

		// Append the revision range for this input line.
		var revRange string
		if remoteSha == zeroHash {
			// In case we are pushing a new branch, check commits up to trunk.
			// There is probably no better guess that we can do in general.
			revRange = fmt.Sprintf("%s..%s", gitConfig.TrunkBranchName(), localSha)
		} else {
			// Otherwise check the commits that are new compared to the remote ref.
			revRange = fmt.Sprintf("%s..%s", remoteSha, localSha)
		}
		revRanges = append(revRanges, revRange)
	}
	if err := scanner.Err(); err != nil {
		return errs.NewError(task, err, nil)
	}

	// Get the relevant commit objects.
	task = "Get the commit objects to be pushed"
	var commits []*git.Commit
	for _, revRange := range revRanges {
		cs, err := git.ShowCommitRange(revRange)
		if err != nil {
			return errs.NewError(task, err, nil)
		}
		commits = append(commits, cs...)
	}

	// Validate the commit messages.
	task = "Validate the commit messages"
	var invalid bool

	stderr := new(bytes.Buffer)
	tw := tabwriter.NewWriter(stderr, 0, 8, 4, '\t', 0)
	io.WriteString(tw, "\n")
	io.WriteString(tw, "Commit SHA\tError\n")
	io.WriteString(tw, "==========\t=====\n")

	for _, commit := range commits {
		if commit.Merge == "" {
			// Require the Change-Id tag in all non-merge commits.
			if commit.ChangeId == "" {
				fmt.Fprintf(tw, "%v\t%v\n", commit.SHA, "commit message: Change-Id tag missing")
				invalid = true
			}
			// Require the Story-Id tag in all non-merge commits.
			if commit.StoryId == "" {
				fmt.Fprintf(tw, "%v\t%v\n", commit.SHA, "commit message: Story-Id tag missing")
				invalid = true
			}
		}
	}

	if invalid {
		tw.Flush()
		stderr.WriteString("\n")
		return errs.NewError(task, errors.New("invalid commit message"), stderr)
	}
	return nil
}
