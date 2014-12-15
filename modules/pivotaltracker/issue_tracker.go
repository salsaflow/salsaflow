package pivotaltracker

import (
	// Stdlib
	"fmt"

	// Internal
	"github.com/salsaflow/salsaflow/modules/common"
	"github.com/salsaflow/salsaflow/version"

	// Other
	"github.com/salsita/go-pivotaltracker/v5/pivotal"
	"github.com/toqueteos/webbrowser"
)

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

	ptStories = storiesMatchingByLabel(ptStories, tracker.config.IncludeStoryLabelFilter())

	return toCommonStories(ptStories, tracker.config), nil
}

func (tracker *issueTracker) StoriesInDevelopment() (stories []common.Story, err error) {
	var (
		client    = pivotal.NewClient(tracker.config.UserToken())
		projectId = tracker.config.ProjectId()
	)
	ptStories, err := searchStories(client, projectId,
		"(state:%v OR state:%v) AND -label:\"%v\" AND -label:\"%v\"",
		pivotal.StoryStateStarted, pivotal.StoryStateFinished,
		tracker.config.ReviewedLabel(), tracker.config.NoReviewLabel())
	if err != nil {
		return nil, err
	}
	return toCommonStories(ptStories, tracker.config), nil
}

func (tracker *issueTracker) NextRelease(
	trunkVersion *version.Version,
	nextTrunkVersion *version.Version,
) (common.NextRelease, error) {

	return newNextRelease(trunkVersion, nextTrunkVersion, tracker.config)
}

func (tracker *issueTracker) RunningRelease(
	releaseVersion *version.Version,
) (common.RunningRelease, error) {

	return newRunningRelease(releaseVersion, tracker.config)
}

func (tracker *issueTracker) OpenStory(storyId string) error {
	return webbrowser.Open(fmt.Sprintf("https://pivotaltracker.com/story/show/%v", storyId))
}
