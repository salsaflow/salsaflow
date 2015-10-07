package github

import (
	// Stdlib
	"fmt"
	"strconv"
	"strings"

	// Internal
	"github.com/salsaflow/salsaflow/action"
	"github.com/salsaflow/salsaflow/errs"
	"github.com/salsaflow/salsaflow/modules/common"

	// Vendor
	"github.com/google/go-github/github"
)

type story struct {
	issue   *github.Issue
	tracker *issueTracker
}

func (story *story) Id() string {
	return strconv.Itoa(*story.issue.Number)
}

func (story *story) ReadableId() string {
	return "#" + story.Id()
}

func (story *story) Type() string {
	return "issue"
}

func (story *story) State() common.StoryState {
	return abstractState(story.issue, story.tracker.config)
}

func (story *story) URL() string {
	return fmt.Sprintf("https://github.com/%v/%v/issues/%v",
		story.tracker.config.GitHubOwner, story.tracker.config.GitHubRepository, *story.issue.Number)
}

func (story *story) Tag() string {
	return fmt.Sprintf("%v/%v#%v",
		story.tracker.config.GitHubOwner, story.tracker.config.GitHubRepository, *story.issue.Number)
}

func (story *story) Title() string {
	return *story.issue.Title
}

func (story *story) Assignees() []common.User {
	if story.issue.Assignee != nil {
		return []common.User{&user{story.issue.Assignee}}
	}
	return nil
}

func (story *story) AddAssignee(user common.User) error {
	return story.SetAssignees([]common.User{user})
}

func (story *story) SetAssignees(users []common.User) error {
	task := fmt.Sprintf("Set the assignee for issue %v", story.ReadableId())
	assignee := users[0].(*user)

	updateFunc := func(
		client *github.Client,
		owner string,
		repo string,
		issue *github.Issue,
	) (*github.Issue, error) {

		updated, _, err := client.Issues.Edit(owner, repo, *issue.Number, &github.IssueRequest{
			Assignee: assignee.me.Login,
		})
		return updated, err
	}

	updated, _, err := story.tracker.updateIssues([]*github.Issue{story.issue}, updateFunc, nil)
	if err != nil {
		return errs.NewError(task, err)
	}

	story.issue = updated[0]
	return nil
}

func (story *story) Start() error {
	task := fmt.Sprintf("Start GitHub issue %v", story.ReadableId())
	_, err := story.setStateLabel(story.tracker.config.BeingImplementedLabel)
	if err != nil {
		return errs.NewError(task, err)
	}
	return nil
}

func (story *story) MarkAsImplemented() (action.Action, error) {
	task := fmt.Sprintf("Mark GitHub issue %v as implemented", story.ReadableId())
	act, err := story.setStateLabel(story.tracker.config.ImplementedLabel)
	if err != nil {
		return nil, errs.NewError(task, err)
	}
	return act, nil
}

func (s *story) LessThan(otherStory common.Story) bool {
	return *s.issue.Number < *otherStory.(*story).issue.Number
}

func (s *story) IssueTracker() common.IssueTracker {
	return s.tracker
}

func (s *story) isLabeled(label string) bool {
	return labeled(s.issue, label)
}

func (s *story) setStateLabel(label string) (action.Action, error) {
	// A helper function for setting issue labels.
	setLabels := func(labels []string) error {
		var (
			client   = s.tracker.newClient()
			owner    = s.tracker.config.GitHubOwner
			repo     = s.tracker.config.GitHubRepository
			issueNum = *s.issue.Number

			updatedLabels []github.Label
			err           error
		)
		withRequestAllocated(func() {
			updatedLabels, _, err = client.Issues.ReplaceLabelsForIssue(
				owner, repo, issueNum, labels)
		})
		if err != nil {
			return err
		}
		s.issue.Labels = updatedLabels
		return nil
	}

	// A helper function for appending label names.
	appendLabelNames := func(names []string, labels []github.Label) []string {
		for _, label := range labels {
			names = append(names, *label.Name)
		}
		return names
	}

	// Set the state labels.
	task := fmt.Sprintf("Set state label to '%v' for issue %v", label, s.ReadableId())

	// Get the right label list.
	otherLabels, prunedLabels := pruneStateLabels(s.tracker.config, s.issue.Labels)
	newLabels := make([]string, 0, len(otherLabels)+1)
	newLabels = appendLabelNames(newLabels, otherLabels)
	newLabels = append(newLabels, label)

	// Update the issue.
	if err := setLabels(newLabels); err != nil {
		return nil, errs.NewError(task, err)
	}

	// Return a rollback function.
	return action.ActionFunc(func() error {
		// Append the pruned labels.
		newLabels := make([]string, 0, len(otherLabels)+len(prunedLabels))
		newLabels = appendLabelNames(newLabels, prunedLabels)

		// Generate the task string.
		task := fmt.Sprintf("Set the state labels to [%v] for issue %v",
			strings.Join(newLabels, ", "), s.ReadableId())

		// Append the other labels as well, thus getting the original label list.
		newLabels = appendLabelNames(newLabels, otherLabels)

		// Update the issue.
		if err := setLabels(newLabels); err != nil {
			return errs.NewError(task, err)
		}
		return nil
	}), nil
}
