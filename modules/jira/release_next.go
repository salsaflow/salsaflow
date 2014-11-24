package jira

import (
	// Stdlib
	"bytes"
	"fmt"
	"os"

	// Internal
	"github.com/salsita/salsaflow/action"
	"github.com/salsita/salsaflow/errs"
	"github.com/salsita/salsaflow/git"
	"github.com/salsita/salsaflow/log"
	"github.com/salsita/salsaflow/modules/jira/client"
	"github.com/salsita/salsaflow/prompt"
	"github.com/salsita/salsaflow/releases"
	"github.com/salsita/salsaflow/version"
)

type nextRelease struct {
	tracker              *issueTracker
	trunkVersion         *version.Version
	trunkVersionResource *client.Version
	nextTrunkVersion     *version.Version
	additionalIssues     []*client.Issue
}

func newNextRelease(
	tracker *issueTracker,
	trunkVersion *version.Version,
	nextTrunkVersion *version.Version,
) (*nextRelease, error) {

	task := fmt.Sprintf("Make sure JIRA version exists for release %v", trunkVersion)
	log.Run(task)
	projectKey := tracker.config.ProjectKey()
	versions, _, err := newClient(tracker.config).Projects.ListVersions(projectKey)
	if err != nil {
		return nil, errs.NewError(task, err, nil)
	}

	tag := trunkVersion.ReleaseTagString()
	for _, v := range versions {
		if v.Name == tag {
			// The associated JIRA version exists, we can return a new tracker instance.
			return &nextRelease{
				tracker:              tracker,
				trunkVersion:         trunkVersion,
				nextTrunkVersion:     nextTrunkVersion,
				trunkVersionResource: v,
			}, nil
		}
	}

	// The associated JIRA version was not found, return an error.
	hint := bytes.NewBufferString(`
Make sure the relevant JIRA version exists.

It is necessary to create the version manually just once
when the project is starting. SalsaFlow will handle
all subsequent JIRA versions for you.

`)
	return nil, errs.NewError(task, fmt.Errorf("JIRA version not found for release %v", tag), hint)
}

func (release *nextRelease) PromptUserToConfirmStart() (bool, error) {
	// Collect the issues to be added to the current release.
	task := "Collect the issues that modified trunk since the last release"
	log.Run(task)
	commits, err := releases.ListNewTrunkCommits()
	if err != nil {
		return false, errs.NewError(task, err, nil)
	}

	// Fetch the additional issues from JIRA.
	task = "Fetch the collected issues from JIRA"
	log.Run(task)
	ids := git.StoryIds(commits)
	issues, err := listStoriesById(newClient(release.tracker.config), ids)
	if len(issues) == 0 && err != nil {
		return false, errs.NewError(task, err, nil)
	}
	if len(issues) != len(ids) {
		log.Warn("Some issues were dropped since they were not found in JIRA")
	}

	// Drop the issues that were already assigned to the right version.
	filteredIssues := make([]*client.Issue, 0, len(issues))
IssueLoop:
	for _, issue := range issues {
		// Add only the parent tasks, i.e. skip sub-tasks.
		if issue.Fields.Parent != nil {
			continue
		}
		// Add only the issues that have not been assigned to the release yet.
		for _, v := range issue.Fields.FixVersions {
			if v.Id == release.trunkVersionResource.Id {
				continue IssueLoop
			}
		}
		filteredIssues = append(filteredIssues, issue)
	}
	issues = filteredIssues

	// Present the issues to the user.
	if len(issues) != 0 {
		fmt.Println("\nThe following issues are going to be added to the release:\n")
		err := prompt.ListStories(toCommonStories(issues, release.tracker.config), os.Stdout)
		if err != nil {
			return false, err
		}
		fmt.Println()
	}

	// Ask the user to confirm.
	ok, err := prompt.Confirm(
		fmt.Sprintf("Are you sure you want to start release %v?", release.trunkVersion))
	if err == nil {
		release.additionalIssues = issues
	}
	return ok, err
}

func (release *nextRelease) Start() (action.Action, error) {
	// We already know that the JIRA version for the release being started exists.
	// That is checked in newNextRelease. What is left is to create a JIRA version
	// for the future release, i.e. the version associated with the version string
	// as committed on the trunk branch.

	// Create the JIRA version for the future release.
	var (
		api = newClient(release.tracker.config)
		tag = release.nextTrunkVersion.ReleaseTagString()
	)
	createTask := fmt.Sprintf("Create JIRA version for the future release (%v)", tag)
	log.Run(createTask)

	versionResource, _, err := api.Versions.Create(&client.Version{
		Name:    tag,
		Project: release.tracker.config.ProjectKey(),
	})
	if err != nil {
		return nil, errs.NewError(createTask, err, nil)
	}

	rollbackFunc := func() error {
		// On rollback, delete the relevant JIRA version.
		log.Rollback(createTask)
		if _, err := api.Versions.Delete(versionResource.Id); err != nil {
			return errs.NewError("Delete JIRA version "+tag, err, nil)
		}
		return nil
	}

	// Set the Fix for Version field for the chosen issues.
	task := fmt.Sprintf(
		"Assign additional issues to the current JIRA version (%v)",
		release.trunkVersion.ReleaseTagString())
	log.Run(task)
	err = assignIssuesToVersion(api, release.additionalIssues, release.trunkVersionResource.Id)
	if err != nil {
		if ex := rollbackFunc(); ex != nil {
			errs.Log(ex)
		}
		return nil, errs.NewError(task, err, nil)
	}

	return action.ActionFunc(rollbackFunc), nil
}
