package jira

import (
	// Stdlib
	"fmt"
	"net/url"

	// Internal
	"github.com/salsita/salsaflow/log"
	"github.com/salsita/salsaflow/modules/common"
	"github.com/salsita/salsaflow/modules/jira/client"
	"github.com/salsita/salsaflow/version"

	// Other
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
	data, _, err := newClient(tracker).Myself.Get()
	if err != nil {
		return nil, err
	}
	return &user{data}, nil
}

func (tracker *issueTracker) NextRelease(
	trunkVersion *version.Version,
	nextTrunkVersion *version.Version,
) (common.NextRelease, error) {

	return newNextRelease(tracker, trunkVersion, nextTrunkVersion)
}

func (tracker *issueTracker) RunningRelease(ver *version.Version) (common.RunningRelease, error) {
	return newRunningRelease(ver)
}

func (tracker *issueTracker) SelectActiveStoryIds(ids []string) (activeIds []string, err error) {
	info := log.V(log.Info)

	// Fetch the relevant issues
	task := "Fetch the relevant issues"
	info.Run(task)

	issues, err := listStoriesById(newClient(tracker), ids)
	if err != nil {
		return nil, err
	}

	issueMap := make(map[string]*client.Issue)
	for _, issue := range issues {
		issueMap[issue.Id] = issue
	}

	// Filter the issues according to the issue state.
	task = "Filter the issues according to the issue state"
	var active []string
	for _, id := range ids {
		_, ok := issueMap[id]
		if !ok {
			info.Fail(task)
			err = fmt.Errorf("issue with id %v not found", id)
			return nil, err
		}

		// XXX: Implement this!
		panic("Not implemented")
	}

	return active, nil
}

func (tracker *issueTracker) OpenStory(storyId string) error {
	relativeURL, _ := url.Parse("browse/" + storyId)
	return webbrowser.Open(tracker.config.BaseURL().ResolveReference(relativeURL).String())
}

func (tracker *issueTracker) StartableStories() (stories []common.Story, err error) {
	query := fmt.Sprintf("project=%v AND (%v) AND (%v)", tracker.config.ProjectKey(),
		formatInRange("type", codingIssueTypeIds...),
		formatInRange("status", startableStateIds...))

	issues, _, err := newClient(tracker).Issues.Search(&client.SearchOptions{
		JQL:        query,
		MaxResults: 200,
	})
	if err != nil {
		return nil, err
	}

	return toCommonStories(tracker, issues), nil
}

func (tracker *issueTracker) StoriesInDevelopment() (stories []common.Story, err error) {
	query := fmt.Sprintf("project=%v AND (%v) AND (%v)", tracker.config.ProjectKey(),
		formatInRange("type", codingIssueTypeIds...),
		formatInRange("status", inDevelopmentStateIds...))

	issues, _, err := newClient(tracker).Issues.Search(&client.SearchOptions{
		JQL:        query,
		MaxResults: 200,
	})
	if err != nil {
		return nil, err
	}

	return toCommonStories(tracker, issues), nil
}

func toCommonStories(tracker *issueTracker, issues []*client.Issue) []common.Story {
	stories := make([]common.Story, len(issues))
	for i := range issues {
		stories[i] = &story{issues[i], tracker}
	}
	return stories
}
