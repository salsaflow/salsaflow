package jira

import (
	// Internal
	"github.com/salsita/SalsaFlow/git-trunk/errors"
	"github.com/salsita/SalsaFlow/git-trunk/modules/common"
	"github.com/salsita/SalsaFlow/git-trunk/modules/jira/client"
)

type story struct {
	s *client.Issue
}

func (story *story) GetId() string {
	return story.s.Id
}

func (story *story) GetReadableId() string {
	return story.s.Key
}

func (story *story) GetAssignees() []common.User {
	panic("Not implemented")
}

func (story *story) GetTitle() string {
	return story.s.Fields.Summary
}

func (story *story) Start() *errors.Error {
	_, err := newClient().Issues.PerformTransition(story.s.Id, transitionStart.Id)
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
	name := users[0].GetId()
	data.Fields.Assignee.Name = name
	_, err := newClient().Issues.Update(story.GetId(), data)
	if err != nil {
		return errors.NewError("Updating story", nil, err)
	}
	return nil
}
