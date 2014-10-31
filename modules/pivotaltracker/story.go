package pivotaltracker

import (
	// Stdlib
	"fmt"
	"strconv"

	// Internal
	"github.com/salsita/salsaflow/errs"
	"github.com/salsita/salsaflow/modules/common"

	// Other
	"gopkg.in/salsita/go-pivotaltracker.v0/v5/pivotal"
)

type story struct {
	*pivotal.Story
}

func (story *story) Id() string {
	return strconv.Itoa(story.Story.Id)
}

// Pivotal Tracker doesn't have readable id so this just returns normal id.
func (story *story) ReadableId() string {
	return strconv.Itoa(story.Story.Id)
}

func (story *story) Title() string {
	return story.Name
}

func (story *story) Assignees() []common.User {
	var users []common.User
	for _, id := range story.OwnerIds {
		users = append(users, userId(id))
	}
	return users
}

func (story *story) AddAssignee(user common.User) *errs.Error {
	msg := fmt.Sprintf("Add user as the owner to story %v", user.Id(), story.Id)
	for _, id := range story.OwnerIds {
		if strconv.Itoa(id) == user.Id() {
			return nil
		}
	}

	id, err := strconv.Atoi(user.Id())
	if err != nil {
		return errs.NewError(msg, err, nil)
	}

	return story.SetAssignees(append(story.Assignees(), userId(id)))
}

func (story *story) SetAssignees(users []common.User) *errs.Error {
	msg := fmt.Sprintf("Set owners for story %v", story.Story.Id)
	ownerIds := make([]int, len(users))
	for i, user := range users {
		id, err := strconv.Atoi(user.Id())
		if err != nil {
			return errs.NewError(msg, err, nil)
		}
		ownerIds[i] = id
	}
	updateRequest := &pivotal.Story{OwnerIds: ownerIds}
	_, stderr, err := updateStories([]*pivotal.Story{story.Story}, func(story *pivotal.Story) *pivotal.Story {
		return updateRequest
	})
	if err != nil {
		return errs.NewError(msg, err, stderr)
	}
	return nil
}

func (story *story) Start() *errs.Error {
	stories := []*pivotal.Story{story.Story}
	_, stderr, err := setStoriesState(stories, pivotal.StoryStateStarted)
	if err != nil {
		return errs.NewError("Start Pivotal Tracker story", stderr, err)
	}
	return nil
}
