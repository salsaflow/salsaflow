package pivotaltracker

import (
	// Stdlib
	"errors"
	"fmt"
	"strconv"

	// Internal
	"github.com/salsaflow/salsaflow/action"
	"github.com/salsaflow/salsaflow/errs"
	"github.com/salsaflow/salsaflow/log"
	"github.com/salsaflow/salsaflow/modules/common"

	// Other
	"gopkg.in/salsita/go-pivotaltracker.v1/v5/pivotal"
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

func (story *story) Type() string {
	return story.Story.Type
}

func (story *story) State() common.StoryState {
	var (
		config           = story.tracker.config
		reviewedLabel    = config.ReviewedLabel
		skipReviewLabel  = config.SkipReviewLabel
		testedLabel      = config.TestedLabel
		skipTestingLabel = config.SkipTestingLabel
	)

	switch story.Story.State {
	case pivotal.StoryStateUnscheduled:
		return common.StoryStateNew

	case pivotal.StoryStatePlanned:
		fallthrough
	case pivotal.StoryStateUnstarted:
		return common.StoryStateApproved

	case pivotal.StoryStateStarted:
		return common.StoryStateBeingImplemented

	case pivotal.StoryStateFinished:
		reviewed := story.isLabeled(reviewedLabel) || story.isLabeled(skipReviewLabel)
		tested := story.isLabeled(testedLabel) || story.isLabeled(skipTestingLabel)

		switch {
		case reviewed && tested:
			return common.StoryStateTested
		case reviewed:
			return common.StoryStateReviewed
		default:
			return common.StoryStateImplemented
		}

	case pivotal.StoryStateDelivered:
		return common.StoryStateStaged

	case pivotal.StoryStateAccepted:
		return common.StoryStateAccepted

	case pivotal.StoryStateRejected:
		return common.StoryStateRejected

	default:
		panic("unknown Pivotal Tracker story state: " + story.Story.State)
	}
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
	for _, id := range story.OwnerIds {
		users = append(users, userId(id))
	}
	return users
}

func (story *story) AddAssignee(user common.User) error {
	task := fmt.Sprintf("Add user as the owner to story %v", user.Id(), story.Id)
	for _, id := range story.OwnerIds {
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
		client    = pivotal.NewClient(config.UserToken)
		projectId = config.ProjectId
	)
	updateRequest := &pivotal.StoryRequest{OwnerIds: &ownerIds}
	updatedStory, _, err := client.Stories.Update(projectId, story.Story.Id, updateRequest)
	if err != nil {
		return errs.NewError(task, err)
	}
	story.Story = updatedStory
	return nil
}

func (story *story) Start() error {
	task := fmt.Sprintf("Start Pivotal Tracker story %v", story.Story.Id)

	if s := story.Story; s.Type == pivotal.StoryTypeFeature && s.Estimate == nil {
		panic(errors.New("story not estimated"))
	}

	var (
		config    = story.tracker.config
		client    = pivotal.NewClient(config.UserToken)
		projectId = config.ProjectId
	)
	updateRequest := &pivotal.StoryRequest{State: pivotal.StoryStateStarted}
	updatedStory, _, err := client.Stories.Update(projectId, story.Story.Id, updateRequest)
	if err != nil {
		return errs.NewError(task, err)
	}
	story.Story = updatedStory
	return nil
}

func (story *story) MarkAsImplemented() (action.Action, error) {
	// Make sure the story is started.
	switch story.Story.State {
	case pivotal.StoryStateStarted:
		// Continue further to set the state to finished.
	case pivotal.StoryStateFinished:
		// Nothing to do here.
		return nil, nil
	default:
		// Foobar, an unexpected story state encountered.
		return nil, fmt.Errorf("unexpected story state: %v", story.State)
	}

	// Set the story state to finished.
	var (
		config    = story.tracker.config
		client    = pivotal.NewClient(config.UserToken)
		projectId = config.ProjectId
	)

	updateTask := fmt.Sprintf("Update Pivotal Tracker story (id = %v)", story.Story.Id)
	updateRequest := &pivotal.StoryRequest{State: pivotal.StoryStateFinished}
	updatedStory, _, err := client.Stories.Update(projectId, story.Story.Id, updateRequest)
	if err != nil {
		return nil, errs.NewError(updateTask, err)
	}
	originalStory := story.Story
	story.Story = updatedStory

	return action.ActionFunc(func() error {
		log.Rollback(updateTask)
		updateRequest := &pivotal.StoryRequest{State: originalStory.State}
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

func (s *story) isLabeled(label string) bool {
	return labeled(s.Story, label)
}
