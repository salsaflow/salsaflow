package postCmd

import (
	// Stdlib
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"strings"

	// Internal
	"github.com/salsaflow/salsaflow/errs"
	"github.com/salsaflow/salsaflow/git"
	"github.com/salsaflow/salsaflow/modules"
)

func postRevisions(revisions ...string) (err error) {
	// Get the commit list for the given revision ranges.
	task := "Get the commits to be posted for code review"
	commits, err := git.ShowCommits(revisions...)
	if err != nil {
		return errs.NewError(task, err)
	}

	// Make sure there are no merge commits.
	if err := ensureNoMergeCommits(commits); err != nil {
		return err
	}

	// Make sure the Story-Id tag is not missing.
	if err := ensureStoryIdTag(commits); err != nil {
		return err
	}

	// Make sure the commits exist in the upstream repository.
	if err := ensureCommitsPushed(commits); err != nil {
		return err
	}

	// Prompt the user to confirm.
	if err := promptUserToConfirmCommits(commits); err != nil {
		return err
	}

	// Post the review requests, in this case it will be only one.
	if err := postCommitsForReview(commits); err != nil {
		return err
	}

	// In case there is no error, tell the user they can do next.
	return printFollowup()
}

func ensureStoryIdTag(commits []*git.Commit) error {
	task := "Make sure all commits contain the Story-Id tag"

	storyIdTag := flagStoryIdTag
	if storyIdTag != "" && storyIdTag != git.StoryIdUnassignedTagValue {
		if err := checkStoryIdTag(storyIdTag); err != nil {
			return errs.NewError(task, err)
		}
	}

	var (
		hint    = bytes.NewBufferString("\n")
		missing bool
	)
	for _, commit := range commits {
		if commit.StoryIdTag == "" {
			if storyIdTag != "" {
				commit.StoryIdTag = storyIdTag
				continue
			}

			fmt.Fprintf(hint, "Commit %v is missing the Story-Id tag\n", commit.SHA)
			missing = true
		}
	}
	fmt.Fprintf(hint, "\n")

	if missing {
		return errs.NewErrorWithHint(
			task, errors.New("Story-Id tag missing"), hint.String())
	}
	return nil
}

func checkStoryIdTag(tag string) error {
	tracker, err := modules.GetIssueTracker()
	if err != nil {
		return err
	}

	_, err = tracker.StoryTagToReadableStoryId(tag)
	return err
}

func ensureCommitsPushed(commits []*git.Commit) error {
	task := "Make sure that all commits exist in the upstream repository"

	// Load git-related config.
	gitConfig, err := git.LoadConfig()
	if err != nil {
		return errs.NewError(task, err)
	}
	remoteName := gitConfig.RemoteName
	remotePrefix := remoteName + "/"

	// Check each commit one by one.
	//
	// We run `git branch -r --contains HASH` for each commit,
	// then we check the output. In case there is a branch prefixed
	// with the right upstream name, the commit is treated as pushed.
	var (
		hint    = bytes.NewBufferString("\n")
		missing bool
	)
CommitLoop:
	for _, commit := range commits {
		// Get `git branch -r --contains HASH` output.
		stdout, err := git.Run("branch", "-r", "--contains", commit.SHA)
		if err != nil {
			return errs.NewError(task, err)
		}

		// Parse `git branch` output line by line.
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(strings.TrimSpace(line), remotePrefix) {
				// The commit is contained in a remote branch, continue.
				continue CommitLoop
			}
		}
		if err := scanner.Err(); err != nil {
			return errs.NewError(task, err)
		}

		// The commit is not contained in any remote branch, bummer.
		fmt.Fprintf(hint,
			"Commit %v has not been pushed into remote '%v' yet.\n", commit.SHA, remoteName)
		missing = true
	}
	fmt.Fprintf(hint, "\n")
	fmt.Fprintf(hint, "All selected commits need to be pushed into the upstream pository.\n")
	fmt.Fprintf(hint, "Please make sure that is the case before trying again.\n")
	fmt.Fprintf(hint, "\n")

	// Return an error in case there is any commit that is not pushed.
	if missing {
		return errs.NewErrorWithHint(
			task, fmt.Errorf("some commits not found in upstream '%v'", remoteName), hint.String())
	}
	return nil
}
