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
	"github.com/salsaflow/salsaflow/prompt"
)

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
		errs.Log(err)
		asciiart.PrintGrimReaper("PUSH ABORTED")
		os.Exit(1)
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

	// Load the hook-related SalsaFlow config.
	enabledTimestamp, err := SalsaFlowEnabledTimestamp()
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

	parseTask := "Parse the hook input"
	var revRanges []*revisionRange
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		var (
			line  = scanner.Text()
			parts = strings.Split(line, " ")
		)
		if len(parts) != 4 {
			return errs.NewError(parseTask, errors.New("invalid input line: "+line), nil)
		}

		localRef, localSha, remoteRef, remoteSha := parts[0], parts[1], parts[2], parts[3]

		// Skip the refs that are being deleted.
		if localSha == git.ZeroHash {
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

		// Make sure the reference is up to date.
		// In this case the reference is not up to date when
		// the remote hash cannot be found in the local clone.
		if remoteSha != git.ZeroHash {
			task := fmt.Sprintf("Make sure remote ref '%s' is up to date", remoteRef)
			if _, err := git.Run("cat-file", "-t", remoteSha); err != nil {
				hint := fmt.Sprintf(`
Commit %v does not exist locally.
This is probably because '%v' is not up to date.
Please update the reference from the remote repository,
perhaps by executing 'git pull'.

`, remoteSha, remoteRef)
				return errs.NewError(task, err, bytes.NewBufferString(hint))
			}
		}

		log.Log(fmt.Sprintf("Checking commits updating reference '%s'", remoteRef))

		// Append the revision range for this input line.
		var revRange *revisionRange
		if remoteSha == git.ZeroHash {
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
		return errs.NewError(parseTask, err, nil)
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
		var (
			salsaflowCommitsDetected bool
			ancestorsChecked         bool
		)
		for _, commit := range commits {
			// Do not check merge commits.
			if commit.Merge != "" {
				continue
			}

			if !enabledTimestamp.IsZero() {
				// In case the SalsaFlow enabled timestamp is available,
				// use it to decide whether to check the commit or not.
				if commit.AuthorDate.Before(enabledTimestamp) {
					continue
				}
			} else {
				// In case the timestamp is missing, we traverse the git graph
				// to see whether there were some commit message tags inserted in the past
				// and we only return an error if that is the case.
				if !salsaflowCommitsDetected {
					switch {
					// Once we encounter a tag inside of the revision range,
					// we automatically start checking for tags.
					case commit.ChangeIdTag != "" || commit.StoryIdTag != "":
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
			}

			commitMessageTitle := prompt.ShortenCommitTitle(commit.MessageTitle)

			printErrorLine := func(reason string) {
				fmt.Fprintf(tw, "%v\t%v\t%v\t%v\n",
					commit.SHA, commitMessageTitle, revRange.To, reason)
				invalid = true
			}

			// Check the Change-Id tag.
			if commit.ChangeIdTag == "" /* && salsaflowCommitsDetected */ {
				printErrorLine("commit message: Change-Id tag missing")
			}

			// Check the Story-Id tag.
			if commit.StoryIdTag == "" /* && salsaflowCommitsDetected */ {
				printErrorLine("commit message: Story-Id tag missing")
			}
		}
	}

	if invalid {
		io.WriteString(tw, "\n")
		tw.Flush()
		return errs.NewError(
			"Validate commit messages", errors.New("invalid commit messages found"), &output)
	}
	return nil
}

func checkAncestors(ref string) (salsaflowCommitsDetected bool, err error) {
	commits, err := git.ShowCommitRange(ref)
	if err != nil {
		return false, err
	}

	for _, commit := range commits {
		if commit.ChangeIdTag != "" || commit.StoryIdTag != "" {
			salsaflowCommitsDetected = true
		}
	}

	return
}
