package pivotaltracker

import (
	// Stdlib
	"strconv"

	// Internal
	"github.com/salsita/SalsaFlow/git-trunk/errors"
	"github.com/salsita/SalsaFlow/git-trunk/modules/common"

	// Other
	"gopkg.in/salsita/go-pivotaltracker.v0/v5/pivotal"
)

type story struct {
	s *pivotal.Story
}

func (story *story) GetId() string {
	return strconv.Itoa(story.s.Id)
}

func (story *story) GetAssignees() []common.User {
	var users []common.User
	for _, id := range story.s.OwnerIds {
		users = append(users, userId(id))
	}
	return users
}

func (story *story) GetTitle() string {
	return story.s.Name
}

func (story *story) Start() *errors.Error {
	stories := []*pivotal.Story{story.s}
	_, stderr, err := setStoriesState(stories, pivotal.StoryStateStarted)
	if err != nil {
		return errors.NewError("Start Pivotal Tracker story", stderr, err)
	}
	return nil
}

func (story *story) SetOwners(users []common.User) *errors.Error {
	msg := "Updating PivotalTracker story"
	ownerIds := make([]int, len(users))
	for i, user := range users {
		id, err := strconv.Atoi(user.GetId())
		if err != nil {
			return errors.NewError(msg, nil, err)
		}
		ownerIds[i] = id
	}
	updateRequest := &pivotal.Story{OwnerIds: ownerIds}
	_, stderr, err := updateStories([]*pivotal.Story{story.s}, func(story *pivotal.Story) *pivotal.Story {
		return updateRequest
	})
	if err != nil {
		return errors.NewError(msg, stderr, err)
	}
	return nil
}
