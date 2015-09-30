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
	issue  *github.Issue
	module *issueTracker
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
	return abstractState(story.issue, story.module.config)
}

func (story *story) URL() string {
	return fmt.Sprintf("https://github.com/%v/%v/issues/%v",
		story.module.config.GitHubOwner, story.module.config.GitHubRepository, *story.issue.Number)
}

func (story *story) Tag() string {
	return fmt.Sprintf("%v/%v#%v",
		story.module.config.GitHubOwner, story.module.config.GitHubRepository, *story.issue.Number)
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

	var (
		client   = story.module.newClient()
		assignee = users[0].(*user)
		owner    = story.module.config.GitHubOwner
		repo     = story.module.config.GitHubRepository
		issueNum = *story.issue.Number

		issue *github.Issue
		err   error
	)
	withRequestAllocated(func() {
		issue, _, err = client.Issues.Edit(owner, repo, issueNum, &github.IssueRequest{
			Assignee: assignee.me.Login,
		})
	})
	if err != nil {
		return errs.NewError(task, err)
	}

	story.issue = issue
	return nil
}

func (story *story) Start() error {
	task := fmt.Sprintf("Start GitHub issue %v", story.ReadableId())
	_, err := story.setWorkflowLabel(story.module.config.BeingImplementedLabel)
	if err != nil {
		return errs.NewError(task, err)
	}
	return nil
}

func (story *story) MarkAsImplemented() (action.Action, error) {
	task := fmt.Sprintf("Mark GitHub issue %v as implemented", story.ReadableId())
	act, err := story.setWorkflowLabel(story.module.config.ImplementedLabel)
	if err != nil {
		return nil, errs.NewError(task, err)
	}
	return act, nil
}

func (s *story) LessThan(otherStory common.Story) bool {
	return *s.issue.Number < *otherStory.(*story).issue.Number
}

func (s *story) IssueTracker() common.IssueTracker {
	return s.module
}

func (s *story) isLabeled(label string) bool {
	return labeled(s.issue, label)
}

func (s *story) setWorkflowLabel(label string) (action.Action, error) {
	task := fmt.Sprintf("Set workflow label to '%v' for issue %v", label, s.ReadableId())

	// Get the right label list.
	remainingLabels, prunedLabels := pruneWorkflowLabels(s.module.config, s.issue.Labels)

	newLabels := make([]string, 0, len(remainingLabels)+1)
	for _, l := range remainingLabels {
		newLabels = append(newLabels, *l.Name)
	}
	newLabels = append(newLabels, label)

	// Update the issue.
	var (
		client   = s.module.newClient()
		owner    = s.module.config.GitHubOwner
		repo     = s.module.config.GitHubRepository
		issueNum = *s.issue.Number

		updatedLabels []github.Label
		err           error
	)
	withRequestAllocated(func() {
		updatedLabels, _, err = client.Issues.ReplaceLabelsForIssue(
			owner, repo, issueNum, newLabels)
	})
	if err != nil {
		return nil, errs.NewError(task, err)
	}
	s.issue.Labels = updatedLabels

	return action.ActionFunc(func() error {
		newLabels := make([]string, 0, len(remainingLabels)+len(prunedLabels))
		appendLabels := func(labels []github.Label) {
			for _, label := range labels {
				newLabels = append(newLabels, *label.Name)
			}
		}
		appendLabels(prunedLabels)
		task := fmt.Sprintf("Set the workflow labels to [%v] for issue %v",
			strings.Join(newLabels, ", "), s.ReadableId())
		appendLabels(remainingLabels)

		var (
			updatedLabels []github.Label
			err           error
		)
		withRequestAllocated(func() {
			updatedLabels, _, err = client.Issues.ReplaceLabelsForIssue(
				owner, repo, issueNum, newLabels)
		})
		if err != nil {
			return errs.NewError(task, err)
		}
		s.issue.Labels = updatedLabels
		return nil
	}), nil
}
