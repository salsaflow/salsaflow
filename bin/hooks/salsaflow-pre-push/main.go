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
	"github.com/salsaflow/salsaflow/asciiart"
	"github.com/salsaflow/salsaflow/errs"
	"github.com/salsaflow/salsaflow/git"
	"github.com/salsaflow/salsaflow/hooks"
	"github.com/salsaflow/salsaflow/log"
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

type revisionRange struct {
	From string
	To   string
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
	var revRanges []*revisionRange
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		var (
			line  = scanner.Text()
			parts = strings.Split(line, " ")
		)
		if len(parts) != 4 {
			return errs.NewError(task, errors.New("invalid input line: "+line), nil)
		}

		localRef, localSha, remoteRef, remoteSha := parts[0], parts[1], parts[2], parts[3]

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

		log.Log(fmt.Sprintf("Checking commits updating reference '%s'", remoteRef))

		// Append the revision range for this input line.
		var revRange *revisionRange
		if remoteSha == zeroHash {
			// In case we are pushing a new branch, check commits up to trunk.
			// There is probably no better guess that we can do in general.
			revRange = &revisionRange{gitConfig.TrunkBranchName(), localRef}
		} else {
			// Otherwise check the commits that are new compared to the remote ref.
			revRange = &revisionRange{remoteSha, localRef}
		}
		revRanges = append(revRanges, revRange)
	}
	if err := scanner.Err(); err != nil {
		return errs.NewError(task, err, nil)
	}

	// Validate the commit messages.
	var (
		invalid bool
		output  bytes.Buffer
		tw      = tabwriter.NewWriter(&output, 0, 8, 4, '\t', 0)
	)

	io.WriteString(tw, "\n")
	io.WriteString(tw, "Commit SHA\tCommit Title\tCommit Source\tError\n")
	io.WriteString(tw, "==========\t============\t=============\t=====\n")

	for _, revRange := range revRanges {
		// Get the commit objects for the relevant range.
		task := "Get the commit objects to be pushed"
		commits, err := git.ShowCommitRange(fmt.Sprintf("%v..%v", revRange.From, revRange.To))
		if err != nil {
			return errs.NewError(task, err, nil)
		}

		// Check every commit in the range.
		task = "Validate commit messages"
		var (
			salsaflowCommitsDetected bool
			ancestorsChecked         bool
		)
		for _, commit := range commits {
			// Do not check merge commits.
			if commit.Merge != "" {
				continue
			}

			// Check whether we should start checking commit messages.
			if !salsaflowCommitsDetected {
				switch {
				// Once we encounter a tag inside of the revision range,
				// we automatically start checking for tags.
				case commit.ChangeId != "" || commit.StoryId != "":
					salsaflowCommitsDetected = true

				// In case the tags are empty, check all ancestors for the relevant tags as well.
				// In case a tag is encountered in an ancestral commit, we start checking for tags.
				case !ancestorsChecked:
					var err error
					salsaflowCommitsDetected, err = checkAncestors(revRange.From)
					if err != nil {
						return errs.NewError(task, err, nil)
					}
					ancestorsChecked = true
				}
			}

			if !salsaflowCommitsDetected {
				continue
			}

			// Check the Change-Id tag.
			if commit.ChangeId == "" /* && salsaflowCommitsDetected */ {
				fmt.Fprintf(tw, "%v\t%v\t%v\t%v\n", commit.SHA, commit.MessageTitle,
					revRange.To, "commit message: Change-Id tag missing")
				invalid = true
			}

			// Check the Story-Id tag.
			if commit.StoryId == "" /* && salsaflowCommitsDetected */ {
				fmt.Fprintf(tw, "%v\t%v\t%v\t%v\n", commit.SHA, commit.MessageTitle,
					revRange.To, "commit message: Story-Id tag missing")
				invalid = true
			}
		}
	}

	if invalid {
		io.WriteString(tw, "\n")
		tw.Flush()
		return errs.NewError(task, errors.New("invalid commit messages found"), &output)
	}
	return nil
}

func checkAncestors(ref string) (salsaflowCommitsDetected bool, err error) {
	commits, err := git.ShowCommitRange(ref)
	if err != nil {
		return false, err
	}

	for _, commit := range commits {
		if commit.ChangeId != "" || commit.StoryId != "" {
			salsaflowCommitsDetected = true
		}
	}

	return
}
