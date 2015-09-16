package main

import (
	// Stdlib
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"

	// Internal
	"github.com/salsaflow/salsaflow/asciiart"
	"github.com/salsaflow/salsaflow/errs"
	"github.com/salsaflow/salsaflow/git"
	"github.com/salsaflow/salsaflow/hooks"
	"github.com/salsaflow/salsaflow/log"
	"github.com/salsaflow/salsaflow/prompt"
	"github.com/salsaflow/salsaflow/repo"
)

func main() {
	// Set up the identification command line flag.
	hooks.IdentifyYourself()

	// Tell the user what is happening.
	fmt.Println("---> Running SalsaFlow pre-push hook")

	// The hook is always invoked as `pre-push <remote-name> <push-url>`.
	if len(os.Args) != 3 {
		fmt.Fprintf(os.Stderr, "Usage: %v <remote-name> <push-url>\n", os.Args[0])
		errs.Fatal(fmt.Errorf("invalid arguments: %#v\n", os.Args[1:]))
	}

	// Run the main function.
	if err := run(os.Args[1], os.Args[2]); err != nil {
		if err != prompt.ErrCanceled {
			fmt.Println()
			errs.Log(err)
		}
		asciiart.PrintGrimReaper("PUSH ABORTED")
		os.Exit(1)
	}

	// Insert an empty line before git push output.
	fmt.Println()
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

	// Load the other necessary SalsaFlow config.
	repoConfig, err := repo.LoadConfig()
	if err != nil {
		return err
	}
	enabledTimestamp := repoConfig.SalsaFlowEnabledTimestamp()

	// Only check the project remote.
	if remoteName != gitConfig.RemoteName {
		log.Log(
			fmt.Sprintf(
				"Not pushing to the main project remote (%v), check skipped",
				gitConfig.RemoteName))
		return nil
	}

	// The commits that are being pushed are listed on stdin.
	// The format is <local ref> <local sha1> <remote ref> <remote sha1>,
	// so we parse the input and collect all the local hexshas.
	var coreRefs = []string{
		"refs/heads/" + gitConfig.TrunkBranchName,
		"refs/heads/" + gitConfig.ReleaseBranchName,
		"refs/heads/" + gitConfig.StagingBranchName,
		"refs/heads/" + gitConfig.StableBranchName,
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
			return errs.NewError(parseTask, errors.New("invalid input line: "+line))
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
				return errs.NewErrorWithHint(task, err, hint)
			}
		}

		// Append the revision range for this input line.
		var revRange *revisionRange
		if remoteSha == git.ZeroHash {
			// In case we are pushing a new branch, check commits up to trunk.
			// There is probably no better guess that we can do in general.
			revRange = &revisionRange{gitConfig.TrunkBranchName, localRef}
		} else {
			// Otherwise check the commits that are new compared to the remote ref.
			revRange = &revisionRange{remoteSha, localRef}
		}
		revRanges = append(revRanges, revRange)
	}
	if err := scanner.Err(); err != nil {
		return errs.NewError(parseTask, err)
	}

	// Check the missing Story-Id tags.
	var missing []*git.Commit

	for _, revRange := range revRanges {
		// Get the commit objects for the relevant range.
		task := "Get the commit objects to be pushed"
		commits, err := git.ShowCommitRange(fmt.Sprintf("%v..%v", revRange.From, revRange.To))
		if err != nil {
			return errs.NewError(task, err)
		}

		// Check every commit in the range.
		for _, commit := range commits {
			// Do not check merge commits.
			if commit.Merge != "" {
				continue
			}

			// Do not check commits that happened before SalsaFlow.
			if commit.AuthorDate.Before(enabledTimestamp) {
				continue
			}

			// Check the Story-Id tag.
			if commit.StoryIdTag == "" {
				missing = append(missing, commit)
			}
		}
	}

	// Prompt for confirmation in case that is needed.
	if len(missing) != 0 {
		// Fill in the commit sources.
		task := "Fix commit sources"
		if err := git.FixCommitSources(missing); err != nil {
			return errs.NewError(task, err)
		}

		// Prompt the user for confirmation.
		task = "Prompt the user for confirmation"
		confirmed, err := promptUserForConfirmation(missing)
		if err != nil {
			return errs.NewError(task, err)
		}
		if !confirmed {
			return prompt.ErrCanceled
		}
	}

	return nil
}

func promptUserForConfirmation(commits []*git.Commit) (bool, error) {
	// Open the console.
	console, err := prompt.OpenConsole(os.O_WRONLY)
	if err != nil {
		return false, err
	}
	defer console.Close()

	// Print the list of commits missing the Story-Id tag.
	fmt.Fprintln(console)
	hooks.PrintUnassignedWarning(console, commits)
	fmt.Fprintln(console)

	// Prompt the user for confirmation.
	defer fmt.Fprintln(console)
	return prompt.Confirm("Are you sure you want to push these commits?", false)
}
