package jira

import (
	// Stdlib
	"fmt"
	"net/url"
	"strings"

	// Internal
	"github.com/salsaflow/salsaflow/modules/common"
	"github.com/salsaflow/salsaflow/version"

	// Vendor
	"github.com/salsita/go-jira/v2/jira"
	"github.com/toqueteos/webbrowser"
)

const ServiceName = "JIRA"

type issueTracker struct {
	config       Config
	versionCache map[string]*jira.Version
}

func Factory() (common.IssueTracker, error) {
	config, err := LoadConfig()
	if err != nil {
		return nil, err
	}
	return &issueTracker{config, nil}, nil
}

func (tracker *issueTracker) ServiceName() string {
	return ServiceName
}

func (tracker *issueTracker) CurrentUser() (common.User, error) {
	data, _, err := newClient(tracker.config).Myself.Get()
	if err != nil {
		return nil, err
	}
	return &user{data}, nil
}

func (tracker *issueTracker) StartableStories() (stories []common.Story, err error) {
	query := fmt.Sprintf("(%v) AND (%v)",
		formatInRange("type", codingIssueTypeIds...),
		formatInRange("status", startableStateIds...))

	return tracker.searchIssues(query)
}

func (tracker *issueTracker) StoriesInDevelopment() (stories []common.Story, err error) {
	query := fmt.Sprintf("(%v) AND (%v)",
		formatInRange("type", codingIssueTypeIds...),
		formatInRange("status", inDevelopmentStateIds...))

	return tracker.searchIssues(query)
}

func (tracker *issueTracker) ReviewedStories() (stories []common.Story, err error) {
	query := fmt.Sprintf("(%v) AND (%v)",
		formatInRange("type", codingIssueTypeIds...),
		formatInRange("status", stateIdReviewed))

	return tracker.searchIssues(query)
}

func (tracker *issueTracker) ListStoriesByTag(tags []string) (stories []common.Story, err error) {
	// Fetch issues by ID. Apparently, the tags are the same as issue keys for JIRA.
	issues, err := listStoriesByIdOrdered(newClient(tracker.config), tags)
	if err != nil {
		return nil, err
	}

	// Convert to []common.Story and return.
	return toCommonStories(issues, tracker), nil
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

	return newRunningRelease(tracker, releaseVersion)
}

func (tracker *issueTracker) OpenStory(storyId string) error {
	relativeURL, _ := url.Parse("browse/" + storyId)
	return webbrowser.Open(tracker.config.ServerURL().ResolveReference(relativeURL).String())
}

func (tracker *issueTracker) StoryTagToReadableStoryId(tag string) (storyId string, err error) {
	prefix := fmt.Sprintf("%v-", tracker.config.ProjectKey())
	if !strings.HasPrefix(tag, prefix) {
		return "", fmt.Errorf("not a valid issue key: %v", tag)
	}
	return tag, nil
}

func (tracker *issueTracker) getVersionResource(ver *version.Version) (*jira.Version, error) {
	var (
		projectKey  = tracker.config.ProjectKey()
		versionName = ver.ReleaseTagString()
		api         = newClient(tracker.config)
	)

	// In case the resource cache is empty, fill it.
	if tracker.versionCache == nil {
		vs, _, err := api.Projects.ListVersions(projectKey)
		if err != nil {
			return nil, err
		}

		m := make(map[string]*jira.Version, len(vs))
		for _, v := range vs {
			m[v.Name] = v
		}
		tracker.versionCache = m
	}

	// Return the resource we are looking for.
	if res, ok := tracker.versionCache[versionName]; ok {
		return res, nil
	}
	return nil, nil
}

func (tracker *issueTracker) searchIssues(query string) ([]common.Story, error) {
	jql := fmt.Sprintf("project = \"%v\" AND (%v)", tracker.config.ProjectKey(), query)

	issues, _, err := newClient(tracker.config).Issues.Search(&jira.SearchOptions{
		JQL:        jql,
		MaxResults: 200,
	})
	if err != nil {
		return nil, err
	}

	return toCommonStories(issues, tracker), nil
}

func toCommonStories(issues []*jira.Issue, tracker *issueTracker) []common.Story {
	stories := make([]common.Story, len(issues))
	for i := range issues {
		s, err := newStory(issues[i], tracker)
		if err != nil {
			panic(err)
		}
		stories[i] = s
	}
	return stories
}
