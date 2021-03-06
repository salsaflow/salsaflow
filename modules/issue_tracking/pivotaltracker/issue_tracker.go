package pivotaltracker

import (
	// Stdlib
	"bytes"
	"fmt"
	"strconv"
	"strings"

	// Internal
	"github.com/salsaflow/salsaflow/modules/common"
	"github.com/salsaflow/salsaflow/version"

	// Other
	"github.com/toqueteos/webbrowser"
	"gopkg.in/salsita/go-pivotaltracker.v1/v5/pivotal"
)

const ServiceName = "Pivotal Tracker"

type issueTracker struct {
	config *moduleConfig
}

func newIssueTracker() (common.IssueTracker, error) {
	config, err := loadConfig()
	if err != nil {
		return nil, err
	}

	return &issueTracker{config}, nil
}

func (tracker *issueTracker) ServiceName() string {
	return ServiceName
}

func (tracker *issueTracker) CurrentUser() (common.User, error) {
	client := pivotal.NewClient(tracker.config.UserToken)
	me, _, err := client.Me.Get()
	if err != nil {
		return nil, err
	}
	return &user{me}, nil
}

func (tracker *issueTracker) StartableStories() ([]common.Story, error) {
	// Fetch the stories with the right story type.
	stories, err := tracker.searchStories(
		"state:%v,%v", pivotal.StoryStateUnstarted, pivotal.StoryStatePlanned)
	if err != nil {
		return nil, err
	}

	// Make sure that only estimated stories are included.
	ss := make([]*pivotal.Story, 0, len(stories))
	for _, story := range stories {
		switch {
		case story.Type == pivotal.StoryTypeFeature && story.Estimate != nil:
			fallthrough
		case story.Type == pivotal.StoryTypeBug:
			ss = append(ss, story)
		}
	}

	// Return what is left.
	return toCommonStories(ss, tracker), nil
}

func (tracker *issueTracker) ReviewableStories() (stories []common.Story, err error) {
	ptStories, err := tracker.searchStories(
		`(state:%v OR state:%v) AND -label:"%v" AND -label:"%v"`,
		pivotal.StoryStateStarted, pivotal.StoryStateFinished,
		tracker.config.ReviewedLabel, tracker.config.SkipReviewLabel)
	if err != nil {
		return nil, err
	}

	return toCommonStories(ptStories, tracker), nil
}

func (tracker *issueTracker) ReviewedStories() (stories []common.Story, err error) {
	ptStories, err := tracker.searchStories(
		"state:%v AND label:\"%v\"", pivotal.StoryStateFinished, tracker.config.ReviewedLabel)
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
	stories, err := tracker.storiesByIdOrdered(ids)
	if err != nil {
		return nil, err
	}

	// Convert to []common.Story and return.
	return toCommonStories(stories, tracker), nil
}

func (tracker *issueTracker) ListStoriesByRelease(v *version.Version) ([]common.Story, error) {
	stories, err := tracker.storiesByRelease(v)
	if err != nil {
		return nil, err
	}
	return toCommonStories(stories, tracker), nil
}

func (tracker *issueTracker) NextRelease(
	trunkVersion *version.Version,
	nextTrunkVersion *version.Version,
) common.NextRelease {

	return newNextRelease(tracker, trunkVersion, nextTrunkVersion)
}

func (tracker *issueTracker) RunningRelease(
	releaseVersion *version.Version,
) common.RunningRelease {

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

func (tracker *issueTracker) searchStories(
	queryFormat string,
	v ...interface{},
) ([]*pivotal.Story, error) {

	// Get the client.
	var (
		client    = pivotal.NewClient(tracker.config.UserToken)
		projectId = tracker.config.ProjectId
	)

	// Generate the query.
	query := fmt.Sprintf(queryFormat, v...)

	// Automatically limit the story type.
	query = fmt.Sprintf("(type:%v OR type:%v) AND (%v)",
		pivotal.StoryTypeFeature, pivotal.StoryTypeBug, query)

	// Send the query to PT.
	stories, err := client.Stories.List(projectId, query)

	// Filter stories by the component label when necessary.
	if label := tracker.config.ComponentLabel; label != "" {
		stories = filterStories(stories, func(story *pivotal.Story) bool {
			return labeled(story, label)
		})
	}

	return stories, err
}

func (tracker *issueTracker) updateStories(
	stories []*pivotal.Story,
	updateFunc storyUpdateFunc,
	rollbackFunc storyUpdateFunc,
) ([]*pivotal.Story, error) {
	var (
		client    = pivotal.NewClient(tracker.config.UserToken)
		projectId = tracker.config.ProjectId
	)
	return updateStories(client, projectId, stories, updateFunc, rollbackFunc)
}

func (tracker *issueTracker) storiesById(ids []string) ([]*pivotal.Story, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	// Generate the query.
	var filter bytes.Buffer
	fmt.Fprintf(&filter, "id:%v", ids[0])
	for _, id := range ids[1:] {
		fmt.Fprintf(&filter, " OR id:%v", id)
	}

	// Send the query.
	return tracker.searchStories(filter.String())
}

func (tracker *issueTracker) storiesByIdOrdered(ids []string) ([]*pivotal.Story, error) {
	// Fetch the stories.
	stories, err := tracker.storiesById(ids)
	if err != nil {
		return nil, err
	}

	// Order them.
	idMap := make(map[string]*pivotal.Story, len(ids))
	for _, story := range stories {
		idMap[strconv.Itoa(story.Id)] = story
	}

	ordered := make([]*pivotal.Story, 0, len(ids))
	for _, id := range ids {
		if story, ok := idMap[id]; ok {
			ordered = append(ordered, story)
			continue
		}
		ordered = append(ordered, nil)
	}

	return ordered, nil
}

func (tracker *issueTracker) storiesByRelease(v *version.Version) ([]*pivotal.Story, error) {
	return tracker.searchStories(`label:"%v"`, getReleaseLabel(v))
}

func (tracker *issueTracker) canStoryBeStaged(story *pivotal.Story) bool {
	var (
		config           = tracker.config
		reviewedLabel    = config.ReviewedLabel
		skipReviewLabel  = config.SkipReviewLabel
		testedLabel      = config.TestedLabel
		skipTestingLabel = config.SkipTestingLabel
	)
	var (
		reviewed = labeled(story, reviewedLabel) || labeled(story, skipReviewLabel)
		tested   = labeled(story, testedLabel) || labeled(story, skipTestingLabel)
	)
	return story.State == pivotal.StoryStateFinished && reviewed && tested
}
