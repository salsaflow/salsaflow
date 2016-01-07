package postCmd

import (
	// Stdlib
	"bytes"
	"errors"
	"fmt"

	// Internal
	"github.com/salsaflow/salsaflow/errs"
	"github.com/salsaflow/salsaflow/git"
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

	var (
		hint    = bytes.NewBufferString("\n")
		missing bool
	)
	for _, commit := range commits {
		if commit.StoryIdTag == "" {
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
