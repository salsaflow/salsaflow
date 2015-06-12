package pivotaltracker

import (
	// Stdlib
	"bytes"
	"fmt"
	"strings"

	// Internal
	"github.com/salsaflow/salsaflow/modules/common"
	"github.com/salsaflow/salsaflow/version"

	// Other
	"github.com/salsita/go-pivotaltracker/v5/pivotal"
	"github.com/toqueteos/webbrowser"
)

const ServiceName = "Pivotal Tracker"

type issueTracker struct {
	config Config
}

func Factory() (common.IssueTracker, error) {
	config, err := LoadConfig()
	if err != nil {
		return nil, err
	}
	return &issueTracker{config}, nil
}

func (tracker *issueTracker) ServiceName() string {
	return ServiceName
}

func (tracker *issueTracker) CurrentUser() (common.User, error) {
	me, err := fetchMe()
	if err != nil {
		return nil, err
	}
	return &user{me}, nil
}

func (tracker *issueTracker) StartableStories() (stories []common.Story, err error) {
	// Fetch the stories with the right story type.
	var (
		client    = pivotal.NewClient(tracker.config.UserToken())
		projectId = tracker.config.ProjectId()
	)
	ptStories, err := searchStories(client, projectId,
		"state:%v,%v", pivotal.StoryStateUnstarted, pivotal.StoryStatePlanned)
	if err != nil {
		return nil, err
	}

	// Make sure that only estimated stories are included.
	ss := make([]*pivotal.Story, 0, len(ptStories))
	for _, story := range ptStories {
		switch {
		case story.Type == pivotal.StoryTypeFeature && story.Estimate != nil:
			fallthrough
		case story.Type == pivotal.StoryTypeBug:
			ss = append(ss, story)
		}
	}

	// Filter by include label.
	ptStories = storiesMatchingByLabel(ss, tracker.config.IncludeStoryLabelFilter())

	// Return what is left.
	return toCommonStories(ptStories, tracker), nil
}

func (tracker *issueTracker) StoriesInDevelopment(
	includeReviewed bool,
) (stories []common.Story, err error) {

	// Build the query.
	query := bytes.NewBufferString(
		fmt.Sprintf("state:%v AND -label:\"%v\"",
			pivotal.StoryStateStarted, tracker.config.NoReviewLabel()))

	if !includeReviewed {
		fmt.Fprintf(query, " AND -label:\"%v\"", tracker.config.ReviewedLabel())
	}

	// Fetch the stories.
	var (
		client    = pivotal.NewClient(tracker.config.UserToken())
		projectId = tracker.config.ProjectId()
	)
	ptStories, err := searchStories(client, projectId, query.String())
	if err != nil {
		return nil, err
	}

	// Wrap the stories in the right interface.
	return toCommonStories(ptStories, tracker), nil
}

func (tracker *issueTracker) ListStoriesByTag(tags []string) ([]common.Story, error) {
	// Convert tags to ids.
	ids := make([]string, 0, len(tags))
	for _, tag := range tags {
		id, err := tracker.StoryTagToReadableStoryId(tag)
		if err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}

	// Fetch the relevant stories.
	var (
		client    = pivotal.NewClient(tracker.config.UserToken())
		projectId = tracker.config.ProjectId()
	)
	stories, err := listStoriesByIdOrdered(client, projectId, ids)
	if err != nil {
		return nil, err
	}

	// Convert to []common.Story and return.
	return toCommonStories(stories, tracker), nil
}

func (tracker *issueTracker) NextRelease(
	trunkVersion *version.Version,
	nextTrunkVersion *version.Version,
) (common.NextRelease, error) {

	return newNextRelease(tracker, trunkVersion, nextTrunkVersion)
}

func (tracker *issueTracker) RunningRelease(
	releaseVersion *version.Version,
) (common.RunningRelease, error) {

	return newRunningRelease(releaseVersion, tracker)
}

func (tracker *issueTracker) OpenStory(storyId string) error {
	return webbrowser.Open(fmt.Sprintf("https://pivotaltracker.com/story/show/%v", storyId))
}

func (tracker *issueTracker) StoryTagToReadableStoryId(tag string) (storyId string, err error) {
	parts := strings.Split(tag, "/")
	if len(parts) != 3 {
		return "", fmt.Errorf("invalid Pivotal Tracker Story-Id tag: %v", tag)
	}
	return parts[2], nil
}
