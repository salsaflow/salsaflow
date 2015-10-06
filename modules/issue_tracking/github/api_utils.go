package github

import (
	// Stdlib
	"bytes"
	"encoding/json"
	"errors"
	"fmt"

	// Internal
	"github.com/salsaflow/salsaflow/action"
	"github.com/salsaflow/salsaflow/errs"

	// Vendor
	"github.com/google/go-github/github"
)

type issueUpdateFunc func(client *github.Client, owner, repo string, issue *github.Issue) (*github.Issue, error)

type issueUpdateResult struct {
	issue *github.Issue
	err   error
}

// updateIssues can be used to update multiple issues at once concurrently.
// It basically calls the given update function on all given issues and
// collects the results. In case there is any error, updateIssues tries
// to revert partial changes. The error returned contains the complete list
// of API call errors as the error hint.
func updateIssues(
	client *github.Client,
	owner string,
	repo string,
	issues []*github.Issue,
	updateFunc issueUpdateFunc,
	rollbackFunc issueUpdateFunc,
) ([]*github.Issue, action.Action, error) {

	// Prepare a function that can be used to apply the given updateFunc.
	// It is later used to both update issues and revert changes.
	update := func(
		issues []*github.Issue,
		updateFunc issueUpdateFunc,
	) (newIssues []*github.Issue, errHint string, err error) {

		// Send the requests concurrently.
		retCh := make(chan *issueUpdateResult, len(issues))
		for _, issue := range issues {
			go func(issue *github.Issue) {
				var (
					updatedIssue *github.Issue
					err          error
				)
				withRequestAllocated(func() {
					updatedIssue, err = updateFunc(client, owner, repo, issue)
				})
				if err == nil {
					// On success, return the updated story.
					retCh <- &issueUpdateResult{updatedIssue, nil}
				} else {
					// On error, keep the original story, add the error.
					retCh <- &issueUpdateResult{nil, err}
				}
			}(issue)
		}

		// Wait for the requests to complete.
		var (
			updatedIssues = make([]*github.Issue, 0, len(issues))
			errFailed     = errors.New("failed to update GitHub issues")
			stderr        bytes.Buffer
		)
		for range issues {
			if ret := <-retCh; ret.err != nil {
				fmt.Fprintln(&stderr, ret.err)
				err = errFailed
			} else {
				updatedIssues = append(updatedIssues, ret.issue)
			}
		}

		return updatedIssues, stderr.String(), err
	}

	// Apply the update function.
	updatedIssues, errHint, err := update(issues, updateFunc)
	if err != nil {
		// In case there is an error, generate the error hint.
		var errHintAcc bytes.Buffer
		errHintAcc.WriteString("\nUpdate Errors\n-------------\n")
		errHintAcc.WriteString(errHint)
		errHintAcc.WriteString("\n")

		// Revert the changes.
		_, errHint, ex := update(updatedIssues, rollbackFunc)
		if ex != nil {
			// In case there is an error during rollback, extend the error hint.
			errHintAcc.WriteString("Rollback Errors\n---------------\n")
			errHintAcc.WriteString(errHint)
			errHintAcc.WriteString("\n")
		}

		return nil, nil, errs.NewErrorWithHint("Update GitHub issues", err, errHintAcc.String())
	}

	// On success, return the updated issues and a rollback function.
	act := action.ActionFunc(func() error {
		_, errHint, err := update(updatedIssues, rollbackFunc)
		if err != nil {
			var errHintAcc bytes.Buffer
			errHintAcc.WriteString("\nRollback Errors\n---------------\n")
			errHintAcc.WriteString(errHint)
			errHintAcc.WriteString("\n")
			return errs.NewErrorWithHint("Revert GitHub issue updates", err, errHintAcc.String())
		}
		return nil
	})
	return updatedIssues, act, nil
}

// setMilestone returns an update function that can be passed into
// updateIssues to set the milestone to the given value.
func setMilestone(milestone *github.Milestone) issueUpdateFunc {
	return func(
		client *github.Client,
		owner string,
		repo string,
		issue *github.Issue,
	) (*github.Issue, error) {

		issue, _, err := client.Issues.Edit(owner, repo, *issue.Number, &github.IssueRequest{
			Milestone: milestone.Number,
		})
		return issue, err
	}
}

// unsetMilestone returns an update function that can be passed into
// updateIssues to clear the milestone.
func unsetMilestone() issueUpdateFunc {
	return func(
		client *github.Client,
		owner string,
		repo string,
		issue *github.Issue,
	) (*github.Issue, error) {

		// It is not possible to unset the milestone using go-github
		// in a nice way, we have to do it manually.
		u := fmt.Sprintf("repos/%v/%v/issues/%v", owner, repo, *issue.Number)
		req, err := client.NewRequest("PATCH", u, json.RawMessage([]byte("{milestone:null}")))
		if err != nil {
			return nil, err
		}

		var i github.Issue
		if _, err := client.Do(req, &i); err != nil {
			return nil, err
		}
		return &i, nil
	}
}
