package jira

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
	"github.com/salsita/go-jira/v2/jira"
)

type story struct {
	*jira.Issue
	seq     int
	tracker *issueTracker
}

func newStory(issue *jira.Issue, tracker *issueTracker) (*story, error) {
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

func (story *story) Type() string {
	return story.Issue.Fields.IssueType.Name
}

func (story *story) State() common.StoryState {
	switch story.Fields.Status.Id {
	case stateIdNew:
		return common.StoryStateNew
	case stateIdApproved:
		return common.StoryStateApproved
	case stateIdBeingImplemented:
		return common.StoryStateBeingImplemented
	case stateIdImplemented:
		return common.StoryStateImplemented
	case stateIdReviewed:
		return common.StoryStateReviewed
	case stateIdBeingTested:
		return common.StoryStateBeingTested
	case stateIdTested:
		return common.StoryStateTested
	case stateIdStaged:
		return common.StoryStateStaged
	case stateIdAccepted:
		return common.StoryStateAccepted
	case stateIdDone:
		fallthrough
	case stateIdComplete:
		fallthrough
	case stateIdReleased:
		fallthrough
	case stateIdClosed:
		return common.StoryStateClosed
	default:
		panic("JIRA issue status not being handled properly")
	}
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

func (story *story) AddAssignee(user common.User) error {
	return story.SetAssignees([]common.User{user})
}

func (story *story) SetAssignees(users []common.User) error {
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
		return errs.NewError(fmt.Sprintf("Set assignees for story %v", story.Issue.Key), err)
	}
	return nil
}

func (story *story) Start() error {
	api := newClient(story.tracker.config)

	_, err := api.Issues.PerformTransition(story.Issue.Id, jira.M{
		"transition": jira.M{
			"id": transitionIdStartImplementing,
		},
	})
	if err != nil {
		return errs.NewError(fmt.Sprintf("Start story %v", story.Issue.Key), err)
	}
	return nil
}

func (story *story) MarkAsImplemented() (action.Action, error) {
	if story.Fields.Status.Id == stateIdImplemented {
		return nil, nil
	}

	fmt.Println(`
SalsaFlow cannot mark the JIRA issue as implemented
since there are some manual steps involved. It will, however,
open the web page where the issue can be marked as implemented.`)

	return nil, story.tracker.OpenStory(story.Issue.Key)
}

func (s *story) LessThan(commonStory common.Story) bool {
	otherStory := commonStory.(*story)
	return s.seq < otherStory.seq
}

func (s *story) IssueTracker() common.IssueTracker {
	return s.tracker
}
