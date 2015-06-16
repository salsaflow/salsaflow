package jira

import (
	// Stdlib
	"fmt"
	"os"

	// Internal
	"github.com/salsaflow/salsaflow/action"
	"github.com/salsaflow/salsaflow/errs"
	"github.com/salsaflow/salsaflow/log"
	"github.com/salsaflow/salsaflow/prompt"
	"github.com/salsaflow/salsaflow/releases"
	"github.com/salsaflow/salsaflow/version"

	// Vendor
	"github.com/salsita/go-jira/v2/jira"
)

type nextRelease struct {
	tracker          *issueTracker
	trunkVersion     *version.Version
	nextTrunkVersion *version.Version
	additionalIssues []*jira.Issue
}

func newNextRelease(
	tracker *issueTracker,
	trunkVersion *version.Version,
	nextTrunkVersion *version.Version,
) (*nextRelease, error) {

	return &nextRelease{
		tracker:          tracker,
		trunkVersion:     trunkVersion,
		nextTrunkVersion: nextTrunkVersion,
	}, nil
}

func (release *nextRelease) PromptUserToConfirmStart() (bool, error) {
	// Collect the issues to be added to the current release.
	task := "Collect the issues that modified trunk since the last release"
	log.Run(task)
	ids, err := releases.ListStoryIdsToBeAssigned(release.tracker)
	if err != nil {
		return false, errs.NewError(task, err)
	}

	// Fetch the additional issues from JIRA.
	task = "Fetch the collected issues from JIRA"
	log.Run(task)
	issues, err := listStoriesById(newClient(release.tracker.config), ids)
	if len(issues) == 0 && err != nil {
		return false, errs.NewError(task, err)
	}
	if len(issues) != len(ids) {
		log.Warn("Some issues were dropped since they were not found in JIRA")
	}

	// Drop the issues that were already assigned to the right version.
	releaseLabel := release.trunkVersion.ReleaseTagString()
	filteredIssues := make([]*jira.Issue, 0, len(issues))
IssueLoop:
	for _, issue := range issues {
		// Add only the parent tasks, i.e. skip sub-tasks.
		if issue.Fields.Parent != nil {
			continue
		}
		// Add only the issues that have not been assigned to the release yet.
		for _, label := range issue.Fields.Labels {
			if label == releaseLabel {
				continue IssueLoop
			}
		}
		filteredIssues = append(filteredIssues, issue)
	}
	issues = filteredIssues

	// Present the issues to the user.
	if len(issues) != 0 {
		fmt.Println("\nThe following issues are going to be added to the release:\n")
		err := prompt.ListStories(toCommonStories(issues, release.tracker), os.Stdout)
		if err != nil {
			return false, err
		}
	}

	// Ask the user to confirm.
	ok, err := prompt.Confirm(
		fmt.Sprintf(
			"\nAre you sure you want to start release %v?",
			release.trunkVersion.BaseString()))
	if err == nil {
		release.additionalIssues = issues
	}
	return ok, err
}

func (release *nextRelease) Start() (action.Action, error) {
	// In case there are no additional stories, we are done.
	if len(release.additionalIssues) == 0 {
		return action.Noop, nil
	}

	// Add the release label to the stories that were assigned automatically.
	releaseLabel := release.trunkVersion.ReleaseTagString()
	task := fmt.Sprintf("Label the newly added issues with the release label (%v)", releaseLabel)
	log.Run(task)

	api := newClient(release.tracker.config)
	if err := addLabel(api, release.additionalIssues, releaseLabel); err != nil {
		return nil, errs.NewError(task, err)
	}

	return action.ActionFunc(func() error {
		return removeLabel(api, release.additionalIssues, releaseLabel)
	}), nil
}
