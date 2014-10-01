package jira

import (
	// Internal
	"github.com/salsita/salsaflow/errors"
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

func (story *story) Assignees() []common.User {
	panic("Not implemented")
}

func (story *story) Title() string {
	return story.Issue.Fields.Summary
}

func (story *story) Start() *errors.Error {
	_, err := newClient().Issues.PerformTransition(story.Issue.Id, transitionStart.Id)
	if err != nil {
		return errors.NewError("Starting JIRA story", nil, err)
	}
	return nil
}

func (story *story) SetOwners(users []common.User) *errors.Error {
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
		return errors.NewError("Updating story", nil, err)
	}
	return nil
}
