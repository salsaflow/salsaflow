package jira

import (
	// Internal
	"github.com/salsita/SalsaFlow/git-trunk/modules/common"
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
	data, err := fetchMyself()
	if err != nil {
		return nil, err
	}
	return &myself{data}, nil
}

func (tracker *issueTracker) ActiveStoryIds(ids []string) (activeIds []string, err error) {
	return onlyActiveStoryIds(ids)
}

func (tracker *issueTracker) NextRelease(ver *version.Version) (common.NextRelease, error) {
	return newNextRelease(ver)
}

func (tracker *issueTracker) RunningRelease(ver *version.Version) (common.RunningRelease, error) {
	return newRunningRelease(ver)
}

func onlyActiveStoryIds(ids []string) (activeIds []string, err error) {
	panic("Not implemented")
}
