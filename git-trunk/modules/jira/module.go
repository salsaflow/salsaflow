package jira

import (
	// Stdlib
	"fmt"

	// Internal
	"github.com/salsita/SalsaFlow/git-trunk/log"
	"github.com/salsita/SalsaFlow/git-trunk/modules/common"
	"github.com/salsita/SalsaFlow/git-trunk/modules/jira/client"
	"github.com/salsita/SalsaFlow/git-trunk/version"
)

type issueTracker struct{}

func Factory() (common.IssueTracker, error) {
	if err := loadConfig(); err != nil {
		return nil, err
	}
	return &issueTracker{}, nil
}

func (tracker *issueTracker) CurrentUser() (common.User, error) {
	resource, _, err := newClient().Myself.Get()
	if err != nil {
		return nil, err
	}
	return &myself{resource}, nil
}

func (tracker *issueTracker) ActiveStoryIds(ids []string) (activeIds []string, err error) {
	return onlyActiveIssueIds(ids)
}

func (tracker *issueTracker) NextRelease(ver *version.Version) (common.NextRelease, error) {
	return newNextRelease(ver)
}

func (tracker *issueTracker) RunningRelease(ver *version.Version) (common.RunningRelease, error) {
	return newRunningRelease(ver)
}

func onlyActiveIssueIds(ids []string) (activeIds []string, err error) {
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
