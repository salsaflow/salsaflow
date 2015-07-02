package jira

import (
	// Stdlib
	"container/list"
	"fmt"
	"os"
	"sort"

	// Internal
	"github.com/salsaflow/salsaflow/action"
	"github.com/salsaflow/salsaflow/errs"
	"github.com/salsaflow/salsaflow/log"
	"github.com/salsaflow/salsaflow/modules/common"
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
	// Fetch the issues already assigned to the release.
	var (
		ver       = release.trunkVersion
		verString = ver.BaseString()
		verLabel  = ver.ReleaseTagString()
	)
	task := fmt.Sprintf("Fetch JIRA issues already assigned to release %v", verString)
	log.Run(task)
	issues, err := release.tracker.issuesByRelease(ver)
	if err != nil {
		return false, errs.NewError(task, err)
	}

	// Collect the issues that modified trunk since the last release.
	task = "Collect the issues that modified trunk since the last release"
	log.Run(task)
	issueKeys, err := releases.ListStoryIdsToBeAssigned(release.tracker)
	if err != nil {
		return false, errs.NewError(task, err)
	}

	// Drop the issues that are already assigned.
	keySet := make(map[string]struct{}, len(issues))
	for _, issue := range issues {
		keySet[issue.Key] = struct{}{}
	}
	keys := make([]string, 0, len(issueKeys))
	for _, key := range issueKeys {
		if _, ok := keySet[key]; !ok {
			keys = append(keys, key)
		}
	}
	issueKeys = keys

	// Fetch the additional issues from JIRA.
	task = "Fetch JIRA issues that modified trunk since the last release"
	log.Run(task)
	collectedIssues, err := listStoriesById(newClient(release.tracker.config), issueKeys)
	if len(collectedIssues) == 0 && err != nil {
		return false, errs.NewError(task, err)
	}
	if len(collectedIssues) != len(issueKeys) {
		log.Warn("Some issues were dropped since they were not found in JIRA")
	}

	// Append the collected issues to the assigned issues.
	issues = append(issues, collectedIssues...)

	// Get the issues to be labeled.
	log.Run("Collect the issues to be assigned automatically")
	toLabel := make([]*jira.Issue, 0, len(issues))

	// We push the collected issues onto a stack and we loop over.
	// During every iteration, we pop an issue, remember it in case it is not labeled,
	// then we push the parent and all the subtasks to the stask to check them later.
	// processedKeys is used to remember what issue keys were checked already.
	processedKeys := make(map[string]struct{}, len(issues))

	// Use list.List as a stack, fill it with the collected issues and loop.
	// Actually doesn't matter whether we use the list as a queue or a stack.
	stack := list.New()
	for _, issue := range issues {
		stack.PushBack(issue)
	}
	for {
		// Pop the top issue from the stack.
		e := stack.Back()
		// No issues left, we are done.
		if e == nil {
			break
		}
		stack.Remove(e)
		issue := e.Value.(*jira.Issue)

		// In case this issue has already been processed, continue.
		if _, processed := processedKeys[issue.Key]; processed {
			continue
		}

		// In case the issue is not labeled, remember it.
		if !isLabeled(issue, verLabel) {
			toLabel = append(toLabel, issue)
		}

		// Push the parent task onto the stack.
		if parent := issue.Fields.Parent; parent != nil {
			stack.PushBack(parent)
		}

		// Push the subtasks onto the stack.
		for _, child := range issue.Fields.Subtasks {
			stack.PushBack(child)
		}

		// Mark the issue as processed.
		processedKeys[issue.Key] = struct{}{}
	}

	// Present the issues to the user.
	if len(toLabel) != 0 {
		fmt.Println("\nThe following issues are going to be added to the release:\n")
		commonStories := common.Stories(toCommonStories(toLabel, release.tracker))
		sort.Sort(common.Stories(commonStories))
		err := prompt.ListStories(commonStories, os.Stdout)
		if err != nil {
			return false, err
		}
	}

	// Ask the user to confirm.
	ok, err := prompt.Confirm(
		fmt.Sprintf("\nAre you sure you want to start release %v?", verString))
	if err == nil {
		release.additionalIssues = toLabel
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
