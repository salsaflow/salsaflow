package jira

import (
	// Stdlib
	"fmt"
	"strconv"
	"strings"

	// Internal
	"github.com/salsaflow/salsaflow/errs"
	"github.com/salsaflow/salsaflow/modules/common"
	"github.com/salsaflow/salsaflow/modules/jira/client"
)

type story struct {
	*client.Issue
	seq     int
	tracker *issueTracker
}

func newStory(issue *client.Issue, tracker *issueTracker) (*story, error) {
	parts := strings.SplitAfterN(issue.Key, "-", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid issue key: %v", issue.Key)
	}

	seq, err := strconv.Atoi(parts[1])
	if err != nil {
		return nil, fmt.Errorf("invalid issue key: %v", issue.Key)
	}

	return &story{issue, seq, tracker}, nil
}

func (story *story) Id() string {
	return story.Issue.Id
}

func (story *story) ReadableId() string {
	return story.Issue.Key
}

func (story *story) URL() string {
	u := newClient(story.tracker.config).BaseURL
	return fmt.Sprintf("%v://%v/browse/%v", u.Scheme, u.Host, story.Issue.Key)
}

func (story *story) Tag() string {
	return story.Issue.Key
}

func (story *story) Title() string {
	return story.Issue.Fields.Summary
}

func (story *story) Assignees() []common.User {
	if story.Issue.Fields.Assignee == nil {
		return nil
	}
	return []common.User{&user{story.Issue.Fields.Assignee}}
}

func (story *story) AddAssignee(user common.User) *errs.Error {
	return story.SetAssignees([]common.User{user})
}

func (story *story) SetAssignees(users []common.User) *errs.Error {
	api := newClient(story.tracker.config)

	var data struct {
		Fields struct {
			Assignee struct {
				Name string `json:"name"`
			} `json:"assignee"`
		} `json:"fields"`
	}
	name := users[0].Id()
	data.Fields.Assignee.Name = name
	_, err := api.Issues.Update(story.Id(), data)
	if err != nil {
		return errs.NewError(fmt.Sprintf("Set assignees for story %v", story.Issue.Key), err, nil)
	}
	return nil
}

func (story *story) Start() *errs.Error {
	api := newClient(story.tracker.config)

	_, err := api.Issues.PerformTransition(story.Issue.Id, transitionIdStartImplementing)
	if err != nil {
		return errs.NewError(fmt.Sprintf("Start story %v", story.Issue.Key), err, nil)
	}
	return nil
}

func (s *story) LessThan(commonStory common.Story) bool {
	otherStory := commonStory.(*story)
	return s.seq < otherStory.seq
}

func (s *story) IssueTracker() common.IssueTracker {
	return s.tracker
}
