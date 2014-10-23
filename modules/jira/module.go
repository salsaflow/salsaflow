package jira

import (
	// Stdlib
	"fmt"
	"strings"

	// Internal
	"github.com/salsita/salsaflow/log"
	"github.com/salsita/salsaflow/modules/common"
	"github.com/salsita/salsaflow/modules/jira/client"
	"github.com/salsita/salsaflow/version"
)

type issueTracker struct{}

func Factory() (common.IssueTracker, error) {
	if err := loadConfig(); err != nil {
		return nil, err
	}
	return &issueTracker{}, nil
}

func (tracker *issueTracker) CurrentUser() (common.User, error) {
	data, _, err := newClient().Myself.Get()
	if err != nil {
		return nil, err
	}
	return &user{data}, nil
}

func (tracker *issueTracker) SelectActiveStoryIds(ids []string) (activeIds []string, err error) {
	return selectActiveIssueIds(ids)
}

func (tracker *issueTracker) NextRelease(ver *version.Version) (common.NextRelease, error) {
	return newNextRelease(ver)
}

func (tracker *issueTracker) RunningRelease(ver *version.Version) (common.RunningRelease, error) {
	return newRunningRelease(ver)
}

func selectActiveIssueIds(ids []string) (activeIds []string, err error) {
	info := log.V(log.Info)

	// Fetch the relevant issues
	msg := "Fetch the relevant issues"
	info.Run(msg)

	issues, err := listStoriesById(ids)
	if err != nil {
		return nil, err
	}

	issueMap := make(map[string]*client.Issue)
	for _, issue := range issues {
		issueMap[issue.Id] = issue
	}

	// Filter the issues according to the issue state.
	msg = "Filter the issues according to the issue state"
	var active []string
	for _, id := range ids {
		_, ok := issueMap[id]
		if !ok {
			info.Fail(msg)
			err = fmt.Errorf("issue with id %v not found", id)
			return nil, err
		}

		// XXX: Implement this!
		panic("Not implemented")
	}

	return active, nil
}

func (tracker *issueTracker) StartableStories() (stories []common.Story, err error) {
	query := fmt.Sprintf("project=%s and (status=%s)",
		config.ProjectKey(), strings.Join(startableStateIds, " OR status="))

	issues, _, err := newClient().Issues.Search(&client.SearchOptions{
		JQL:        query,
		MaxResults: 200,
	})
	if err != nil {
		return nil, err
	}

	return toCommonStories(issues), nil
}

func (tracker *issueTracker) StoriesInProgress() (stories []common.Story, err error) {
	query := fmt.Sprintf("project=%s and (status=%s)",
		config.ProjectKey(), strings.Join(inDevelopmentStateIds, " OR status="))

	issues, _, err := newClient().Issues.Search(&client.SearchOptions{
		JQL:        query,
		MaxResults: 200,
	})
	if err != nil {
		return nil, err
	}

	return toCommonStories(issues), nil
}

func toCommonStories(issues []*client.Issue) []common.Story {
	stories := make([]common.Story, len(issues))
	for i := range issues {
		stories[i] = &story{issues[i]}
	}
	return stories
}
