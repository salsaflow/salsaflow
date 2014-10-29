package jira

import (
	// Internal
	"github.com/salsita/salsaflow/errs"
	"github.com/salsita/salsaflow/modules/common"
	"github.com/salsita/salsaflow/modules/jira/client"
)

type story struct {
	*client.Issue
}

func (story *story) Id() string {
	return story.Issue.Id
}

func (story *story) ReadableId() string {
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
	var data struct {
		Fields struct {
			Assignee struct {
				Name string `json:"name"`
			} `json:"assignee"`
		} `json:"fields"`
	}
	name := users[0].Id()
	data.Fields.Assignee.Name = name
	_, err := newClient().Issues.Update(story.Id(), data)
	if err != nil {
		return errs.NewError("Updating story", nil, err)
	}
	return nil
}

func (story *story) Start() *errs.Error {
	_, err := newClient().Issues.PerformTransition(story.Issue.Id, transitionIdStartImplementing)
	if err != nil {
		return errs.NewError("Starting JIRA story", nil, err)
	}
	return nil
}
