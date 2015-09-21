// Copyright (c) 2014 Salsita Software
// Copyright (C) 2015 Scott Devoid
// Use of this source code is governed by the MIT License.
// The license can be found in the LICENSE file.

package pivotal

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

const (
	StoryTypeFeature = "feature"
	StoryTypeBug     = "bug"
	StoryTypeChore   = "chore"
	StoryTypeRelease = "release"
)

const (
	StoryStateUnscheduled = "unscheduled"
	StoryStatePlanned     = "planned"
	StoryStateUnstarted   = "unstarted"
	StoryStateStarted     = "started"
	StoryStateFinished    = "finished"
	StoryStateDelivered   = "delivered"
	StoryStateAccepted    = "accepted"
	StoryStateRejected    = "rejected"
)

type Story struct {
	Id            int        `json:"id,omitempty"`
	ProjectId     int        `json:"project_id,omitempty"`
	Name          string     `json:"name,omitempty"`
	Description   string     `json:"description,omitempty"`
	Type          string     `json:"story_type,omitempty"`
	State         string     `json:"current_state,omitempty"`
	Estimate      *float64   `json:"estimate,omitempty"`
	AcceptedAt    *time.Time `json:"accepted_at,omitempty"`
	Deadline      *time.Time `json:"deadline,omitempty"`
	RequestedById int        `json:"requested_by_id,omitempty"`
	OwnerIds      []int      `json:"owner_ids,omitempty"`
	LabelIds      []int      `json:"label_ids,omitempty"`
	Labels        []*Label   `json:"labels,omitempty"`
	TaskIds       []int      `json:"task_ids,omitempty"`
	Tasks         []int      `json:"tasks,omitempty"`
	FollowerIds   []int      `json:"follower_ids,omitempty"`
	CommentIds    []int      `json:"comment_ids,omitempty"`
	CreatedAt     *time.Time `json:"created_at,omitempty"`
	UpdatedAt     *time.Time `json:"updated_at,omitempty"`
	IntegrationId int        `json:"integration_id,omitempty"`
	ExternalId    string     `json:"external_id,omitempty"`
	URL           string     `json:"url,omitempty"`
}

type StoryRequest struct {
	Name        string    `json:"name,omitempty"`
	Description string    `json:"description,omitempty"`
	Type        string    `json:"story_type,omitempty"`
	State       string    `json:"current_state,omitempty"`
	Estimate    *float64  `json:"estimate,omitempty"`
	OwnerIds    *[]int    `json:"owner_ids,omitempty"`
	LabelIds    *[]int    `json:"label_ids,omitempty"`
	Labels      *[]*Label `json:"labels,omitempty"`
	TaskIds     *[]int    `json:"task_ids,omitempty"`
	Tasks       *[]int    `json:"tasks,omitempty"`
	FollowerIds *[]int    `json:"follower_ids,omitempty"`
	CommentIds  *[]int    `json:"comment_ids,omitempty"`
}

type Label struct {
	Id        int        `json:"id,omitempty"`
	ProjectId int        `json:"project_id,omitempty"`
	Name      string     `json:"name,omitempty"`
	CreatedAt *time.Time `json:"created_at,omitempty"`
	UpdatedAt *time.Time `json:"updated_at,omitempty"`
	Kind      string     `json:"kind,omitempty"`
}

type Task struct {
	Id          int        `json:"id,omitempty"`
	StoryId     int        `json:"story_id,omitempty"`
	Description string     `json:"description,omitempty"`
	Position    int        `json:"position,omitempty"`
	Complete    bool       `json:"complete,omitempty"`
	CreatedAt   *time.Time `json:"created_at,omitempty"`
	UpdatedAt   *time.Time `json:"updated_at,omitempty"`
}

type Person struct {
	Id       int    `json:"id,omitempty"`
	Name     string `json:"name,omitempty"`
	Email    string `json:"email,omitempty"`
	Initials string `json:"initials,omitempty"`
	Username string `json:"username,omitempty"`
	Kind     string `json:"kind,omitempty"`
}

type Comment struct {
	Id                  int        `json:"id,omitempty"`
	StoryId             int        `json:"story_id,omitempty"`
	EpicId              int        `json:"epic_id,omitempty"`
	PersonId            int        `json:"person_id,omitempty"`
	Text                string     `json:"text,omitempty"`
	FileAttachmentIds   []int      `json:"file_attachment_ids,omitempty"`
	GoogleAttachmentIds []int      `json:"google_attachment_ids,omitempty"`
	CommitType          string     `json:"commit_type,omitempty"`
	CommitIdentifier    string     `json:"commit_identifier,omitempty"`
	CreatedAt           *time.Time `json:"created_at,omitempty"`
	UpdatedAt           *time.Time `json:"updated_at,omitempty"`
}

type StoryService struct {
	client *Client
}

func newStoryService(client *Client) *StoryService {
	return &StoryService{client}
}

// List returns all stories matching the filter in case the filter is specified.
// It uses Iterate() to collect all stories and returns them as a slice.
func (service *StoryService) List(projectId int, filter string) ([]*Story, error) {
	cursor, err := service.Iterate(projectId, filter)
	if err != nil {
		return nil, err
	}

	stories := make([]*Story, 0, 10)
	for {
		story, err := cursor.Next()
		if err != nil {
			if err == io.EOF {
				return stories, nil
			}
			return nil, err
		}
		stories = append(stories, story)
	}
}

type StoryCursor struct {
	*cursor
	buff []*Story
}

// Next returns the next story.
//
// In case there are no more stories, io.EOF is returned as an error.
func (c *StoryCursor) Next() (s *Story, err error) {
	if len(c.buff) == 0 {
		_, err = c.next(&c.buff)
		if err != nil {
			return nil, err
		}
	}

	if len(c.buff) == 0 {
		err = io.EOF
	} else {
		s, c.buff = c.buff[0], c.buff[1:]
	}
	return s, err
}

// Iterate returns a cursor that can be used to iterate over the stories specified
// by the filter. More stories are fetched on demand as needed.
func (service *StoryService) Iterate(projectId int, filter string) (c *StoryCursor, err error) {
	reqfn := func() (req *http.Request) {
		u := fmt.Sprintf("projects/%v/stories", projectId)
		if filter != "" {
			u += "?filter=" + url.QueryEscape(filter)
		}
		req, _ = service.client.NewRequest("GET", u, nil)
		return req
	}
	cc, err := newCursor(service.client, reqfn, 10)
	return &StoryCursor{cc, make([]*Story, 0)}, err
}

func (service *StoryService) Get(projectId, storyId int) (*Story, *http.Response, error) {
	u := fmt.Sprintf("projects/%v/stories/%v", projectId, storyId)
	req, err := service.client.NewRequest("GET", u, nil)
	if err != nil {
		return nil, nil, err
	}

	var story Story
	resp, err := service.client.Do(req, &story)
	if err != nil {
		return nil, resp, err
	}

	return &story, resp, err
}

func (service *StoryService) Update(projectId, storyId int, story *StoryRequest) (*Story, *http.Response, error) {
	u := fmt.Sprintf("projects/%v/stories/%v", projectId, storyId)
	req, err := service.client.NewRequest("PUT", u, story)
	if err != nil {
		return nil, nil, err
	}

	var bodyStory Story
	resp, err := service.client.Do(req, &bodyStory)
	if err != nil {
		return nil, resp, err
	}

	return &bodyStory, resp, err

}

func (service *StoryService) ListTasks(projectId, storyId int) ([]*Task, *http.Response, error) {
	u := fmt.Sprintf("projects/%v/stories/%v/tasks", projectId, storyId)
	req, err := service.client.NewRequest("GET", u, nil)
	if err != nil {
		return nil, nil, err
	}

	var tasks []*Task
	resp, err := service.client.Do(req, &tasks)
	if err != nil {
		return nil, resp, err
	}

	return tasks, resp, err
}

func (service *StoryService) AddTask(projectId, storyId int, task *Task) (*http.Response, error) {
	if task.Description == "" {
		return nil, &ErrFieldNotSet{"description"}
	}

	u := fmt.Sprintf("projects/%v/stories/%v/tasks", projectId, storyId)
	req, err := service.client.NewRequest("POST", u, task)
	if err != nil {
		return nil, err
	}

	return service.client.Do(req, nil)
}

func (service *StoryService) ListOwners(projectId, storyId int) ([]*Person, *http.Response, error) {
	u := fmt.Sprintf("projects/%d/stories/%d/owners", projectId, storyId)
	req, err := service.client.NewRequest("GET", u, nil)
	if err != nil {
		return nil, nil, err
	}

	var owners []*Person
	resp, err := service.client.Do(req, &owners)
	if err != nil {
		return nil, resp, err
	}

	return owners, resp, err
}

func (service *StoryService) AddComment(
	projectId int,
	storyId int,
	comment *Comment,
) (*Comment, *http.Response, error) {

	u := fmt.Sprintf("projects/%v/stories/%v/comments", projectId, storyId)
	req, err := service.client.NewRequest("POST", u, comment)
	if err != nil {
		return nil, nil, err
	}

	var newComment Comment
	resp, err := service.client.Do(req, &newComment)
	if err != nil {
		return nil, resp, err
	}

	return &newComment, resp, err
}