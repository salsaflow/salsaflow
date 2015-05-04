package pivotaltracker

import (
	// Stdlib
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
	var (
		client    = pivotal.NewClient(tracker.config.UserToken())
		projectId = tracker.config.ProjectId()
	)
	ptStories, err := searchStories(client, projectId, "state:%v", pivotal.StoryStateUnstarted)
	if err != nil {
		return nil, err
	}

	// Drop the features that are not estimated.
	estimatedStories := make([]*pivotal.Story, 0, len(ptStories))
	for _, story := range ptStories {
		if !(story.Type == pivotal.StoryTypeFeature && story.Estimate == nil) {
			estimatedStories = append(estimatedStories, story)
		}
	}

	ptStories = storiesMatchingByLabel(estimatedStories, tracker.config.IncludeStoryLabelFilter())

	return toCommonStories(ptStories, tracker), nil
}

func (tracker *issueTracker) StoriesInDevelopment() (stories []common.Story, err error) {
	var (
		client    = pivotal.NewClient(tracker.config.UserToken())
		projectId = tracker.config.ProjectId()
	)
	ptStories, err := searchStories(client, projectId,
		"state:%v AND -label:\"%v\" AND -label:\"%v\"",
		pivotal.StoryStateStarted, tracker.config.ReviewedLabel(), tracker.config.NoReviewLabel())
	if err != nil {
		return nil, err
	}
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
