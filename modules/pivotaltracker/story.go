package pivotaltracker

import (
	// Stdlib
	"fmt"
	"strconv"

	// Internal
	"github.com/salsaflow/salsaflow/errs"
	"github.com/salsaflow/salsaflow/modules/common"

	// Other
	"github.com/salsita/go-pivotaltracker/v5/pivotal"
)

type story struct {
	*pivotal.Story
	tracker *issueTracker
}

func (story *story) Id() string {
	return strconv.Itoa(story.Story.Id)
}

// Pivotal Tracker doesn't have any readable id, just return the story ID.
func (story *story) ReadableId() string {
	return strconv.Itoa(story.Story.Id)
}

func (story *story) URL() string {
	return fmt.Sprintf("https://www.pivotaltracker.com/projects/%v/stories/%v",
		story.Story.ProjectId, story.Story.Id)
}

func (story *story) Tag() string {
	return fmt.Sprintf("%v/stories/%v", story.Story.ProjectId, story.Story.Id)
}

func (story *story) Title() string {
	return story.Name
}

func (story *story) Assignees() []common.User {
	var users []common.User
	for _, id := range *story.OwnerIds {
		users = append(users, userId(id))
	}
	return users
}

func (story *story) AddAssignee(user common.User) error {
	task := fmt.Sprintf("Add user as the owner to story %v", user.Id(), story.Id)
	for _, id := range *story.OwnerIds {
		if strconv.Itoa(id) == user.Id() {
			return nil
		}
	}

	id, err := strconv.Atoi(user.Id())
	if err != nil {
		return errs.NewError(task, err, nil)
	}

	return story.SetAssignees(append(story.Assignees(), userId(id)))
}

func (story *story) SetAssignees(users []common.User) error {
	task := fmt.Sprintf("Set owners for story %v", story.Story.Id)

	ownerIds := make([]int, len(users))
	for i, user := range users {
		id, err := strconv.Atoi(user.Id())
		if err != nil {
			return errs.NewError(task, err, nil)
		}
		ownerIds[i] = id
	}

	var (
		config    = story.tracker.config
		client    = pivotal.NewClient(config.UserToken())
		projectId = config.ProjectId()
	)
	updateRequest := &pivotal.Story{OwnerIds: &ownerIds}
	updatedStory, _, err := client.Stories.Update(projectId, story.Story.Id, updateRequest)
	if err != nil {
		return errs.NewError(task, err, nil)
	}
	story.Story = updatedStory
	return nil
}

func (story *story) Start() error {
	task := fmt.Sprintf("Start Pivotal Tracker story %v", story.Story.Id)

	var (
		config    = story.tracker.config
		client    = pivotal.NewClient(config.UserToken())
		projectId = config.ProjectId()
	)
	updateRequest := &pivotal.Story{State: pivotal.StoryStateStarted}
	updatedStory, _, err := client.Stories.Update(projectId, story.Story.Id, updateRequest)
	if err != nil {
		return errs.NewError(task, err, nil)
	}
	story.Story = updatedStory
	return nil
}

func (s *story) LessThan(otherStory common.Story) bool {
	return s.CreatedAt.Before(*otherStory.(*story).CreatedAt)
}

func (s *story) IssueTracker() common.IssueTracker {
	return s.tracker
}
