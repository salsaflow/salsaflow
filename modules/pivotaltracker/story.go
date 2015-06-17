package pivotaltracker

import (
	// Stdlib
	"fmt"
	"strconv"

	// Internal
	"github.com/salsaflow/salsaflow/action"
	"github.com/salsaflow/salsaflow/errs"
	"github.com/salsaflow/salsaflow/log"
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
		return errs.NewError(task, err)
	}

	return story.SetAssignees(append(story.Assignees(), userId(id)))
}

func (story *story) SetAssignees(users []common.User) error {
	task := fmt.Sprintf("Set owners for story %v", story.Story.Id)

	ownerIds := make([]int, len(users))
	for i, user := range users {
		id, err := strconv.Atoi(user.Id())
		if err != nil {
			return errs.NewError(task, err)
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
		return errs.NewError(task, err)
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
		return errs.NewError(task, err)
	}
	story.Story = updatedStory
	return nil
}

func (story *story) MarkAsImplemented() (action.Action, error) {
	var (
		config    = story.tracker.config
		client    = pivotal.NewClient(config.UserToken())
		projectId = config.ProjectId()
		label     = config.ReviewedLabel()
	)

	var alreadyThere bool
	ls := make([]*pivotal.Label, 0, len(*story.Labels))
	for _, l := range *story.Labels {
		if l.Name == label {
			alreadyThere = true
		}
		ls = append(ls, &pivotal.Label{Name: l.Name})
	}
	if alreadyThere {
		return nil, nil
	}
	ls = append(ls, &pivotal.Label{Name: label})

	updateTask := fmt.Sprintf("Update Pivotal Tracker story (id = %v)", story.Story.Id)
	updateRequest := &pivotal.Story{Labels: &ls}
	updatedStory, _, err := client.Stories.Update(projectId, story.Story.Id, updateRequest)
	if err != nil {
		return nil, errs.NewError(updateTask, err)
	}
	originalStory := story.Story
	story.Story = updatedStory

	return action.ActionFunc(func() error {
		log.Rollback(updateTask)
		updateRequest := &pivotal.Story{Labels: originalStory.Labels}
		updatedStory, _, err := client.Stories.Update(projectId, story.Story.Id, updateRequest)
		if err != nil {
			return err
		}
		story.Story = updatedStory
		return nil
	}), nil
}

func (s *story) LessThan(otherStory common.Story) bool {
	return s.CreatedAt.Before(*otherStory.(*story).CreatedAt)
}

func (s *story) IssueTracker() common.IssueTracker {
	return s.tracker
}
