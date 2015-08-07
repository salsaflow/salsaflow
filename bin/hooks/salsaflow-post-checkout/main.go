package main

import (
	// Stdlib
	"fmt"
	"os"

	// Internal
	"github.com/salsaflow/salsaflow/commands/review/post/constants"
	"github.com/salsaflow/salsaflow/errs"
	"github.com/salsaflow/salsaflow/git"
	"github.com/salsaflow/salsaflow/hooks"
	"github.com/salsaflow/salsaflow/prompt"
	"github.com/salsaflow/salsaflow/repo"
)

func main() {
	// Register the magical -salsaflow.version flag.
	hooks.IdentifyYourself()

	// Run the hook logic itself.
	if err := hook(); err != nil {
		errs.Fatal(err)
	}
}

func hook() error {
	// There are always 3 arguments passed to this hook.
	prevRef, newRef, flag := os.Args[1], os.Args[2], os.Args[3]

	// Return in case prevRef is the zero hash since that means
	// that this hook is being run right after 'git clone'.
	if prevRef == git.ZeroHash {
		return nil
	}

	// Return in case flag is '0'. That signals retrieving a file from the index.
	if flag == "0" {
		return nil
	}

	// Return unless the new HEAD is a core branch.
	isCore, err := isCoreBranchHash(newRef)
	if err != nil {
		return err
	}
	if !isCore {
		return nil
	}

	// Return also in case we are doing something with a temporary branch.
	isNewRefTemp, err := isTempBranchHash(newRef)
	if err != nil {
		return err
	}
	isPrevRefTemp, err := isTempBranchHash(prevRef)
	if err != nil {
		return err
	}
	if isNewRefTemp || isPrevRefTemp {
		return nil
	}

	// Get the relevant commits.
	// These are the commits specified by newRef..prevRef, e.g. trunk..story/foobar.
	commits, err := git.ShowCommitRange(fmt.Sprintf("%v..%v", newRef, prevRef))
	if err != nil {
		return err
	}

	// Drop commits that happened before SalsaFlow bootstrap.
	repoConfig, err := repo.LoadConfig()
	if err != nil {
		return err
	}
	enabledTimestamp := repoConfig.SalsaFlowEnabledTimestamp()
	commits = git.FilterCommits(commits, func(commit *git.Commit) bool {
		return commit.AuthorDate.After(enabledTimestamp)
	})

	// Collect the commits with missing Story-Id tag.
	missing := make([]*git.Commit, 0, len(commits))
	for _, commit := range commits {
		// Skip merge commits.
		if commit.Merge != "" {
			continue
		}

		// Add the commit in case Story-Id tag is not set.
		if commit.StoryIdTag == "" {
			missing = append(missing, commit)
		}
	}
	if len(missing) == 0 {
		return nil
	}

	// Fix commit sources.
	if err := git.FixCommitSources(missing); err != nil {
		return err
	}

	// Print the warning.
	return printWarning(missing)
}

func isCoreBranchHash(hash string) (bool, error) {
	hashes, err := git.CoreBranchHashes()
	if err != nil {
		return false, err
	}
	for _, h := range hashes {
		if h == hash {
			return true, nil
		}
	}
	return false, nil
}

func isTempBranchHash(hash string) (bool, error) {
	// Check whether the temp branch actually exists.
	// Obviously we want to return false when there is no such branch.
	exists, err := git.LocalBranchExists(constants.TempBranchName)
	if err != nil {
		return false, err
	}
	if !exists {
		return false, nil
	}

	// In case the temp branch exists, compare the hashes.
	tempHash, err := git.BranchHexsha(constants.TempBranchName)
	if err != nil {
		return false, err
	}
	return tempHash == hash, nil
}

func printWarning(commits []*git.Commit) error {
	console, err := prompt.OpenConsole(os.O_WRONLY)
	if err != nil {
		return err
	}
	defer console.Close()

	fmt.Fprintln(console)
	hooks.PrintUnassignedWarning(console, commits)
	fmt.Fprintln(console)

	return nil
}
