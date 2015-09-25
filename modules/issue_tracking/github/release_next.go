package github

import (
	// Stdlib
	"fmt"
	"os"
	"strconv"

	// Internal
	"github.com/salsaflow/salsaflow/action"
	"github.com/salsaflow/salsaflow/errs"
	"github.com/salsaflow/salsaflow/log"
	"github.com/salsaflow/salsaflow/prompt"
	"github.com/salsaflow/salsaflow/prompt/storyprompt"
	"github.com/salsaflow/salsaflow/releases"
	"github.com/salsaflow/salsaflow/version"

	// Other
	"github.com/google/go-github/github"
)

type nextRelease struct {
	tracker *issueTracker

	trunkVersion     *version.Version
	nextTrunkVersion *version.Version

	additionalIssues []*github.Issue
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

// PromptUserToConfirm is a part of common.NextRelease interface.
func (release *nextRelease) PromptUserToConfirmStart() (bool, error) {
	// Fetch the stories already assigned to the release.
	var (
		ver       = release.trunkVersion
		verString = ver.BaseString()
	)
	task := fmt.Sprintf("Fetch GitHub issues already assigned to release %v", verString)
	log.Run(task)
	assignedIssues, err := release.tracker.issuesByRelease(ver)
	if err != nil {
		return false, errs.NewError(task, err)
	}

	// Collect the issues that modified trunk since the last release.
	task = "Collect the issues that modified trunk since the last release"
	log.Run(task)
	issueNumsString, err := releases.ListStoryIdsToBeAssigned(release.tracker)
	if err != nil {
		return false, errs.NewError(task, err)
	}

	// Turn []string into []int.
	issueNums := make([]int, len(issueNumsString))
	for i, numString := range issueNumsString {
		// numString is #ISSUE_NUMBER.
		num, err := strconv.Atoi(numString[1:])
		if err != nil {
			panic(err)
		}
		issueNums[i] = num
	}

	// Drop the stories that are already assigned.
	numSet := make(map[int]struct{}, len(assignedIssues))
	for _, issue := range assignedIssues {
		numSet[*issue.Number] = struct{}{}
	}
	nums := make([]int, 0, len(issueNums))
	for _, num := range issueNums {
		if _, ok := numSet[num]; !ok {
			nums = append(nums, num)
		}
	}
	issueNums = nums

	// Fetch the collected issues from GitHub, if necessary.
	var additionalIssues []*github.Issue
	if len(issueNums) != 0 {
		task = "Fetch the collected issues from GitHub"
		log.Run(task)

		var err error
		additionalIssues, err = release.tracker.issuesByNumber(issueNums)
		if err != nil {
			return false, errs.NewError(task, err)
		}

		// Drop stories already assigned to another release.
		notAssigned := make([]*github.Issue, 0, len(additionalIssues))
		for _, issue := range additionalIssues {
			switch {
			case issue.Milestone == nil:
				notAssigned = append(notAssigned, issue)
			default:
				log.Warn(fmt.Sprintf(
					"Skipping issue #%v: modified trunk, but already assigned to milestone '%v'",
					*issue.Number, *issue.Milestone.Title))
			}
		}
		additionalIssues = notAssigned
	}

	// Print the summary into the console.
	summary := []struct {
		header string
		issues []*github.Issue
	}{
		{
			"The following issues were manually assigned to the release:",
			assignedIssues,
		},
		{
			"The following issues were added automatically (modified trunk):",
			additionalIssues,
		},
	}
	for _, item := range summary {
		if len(item.issues) != 0 {
			fmt.Println()
			fmt.Println(item.header)
			fmt.Println()
			err := storyprompt.ListStories(
				toCommonStories(item.issues, release.tracker), os.Stdout)
			if err != nil {
				return false, err
			}
		}
	}

	// Ask the user to confirm.
	ok, err := prompt.Confirm(
		fmt.Sprintf("\nAre you sure you want to start release %v?", verString), false)
	if err == nil {
		release.additionalIssues = additionalIssues
	}
	return ok, err
}

// Start is a part of common.NextRelease interface.
func (release *nextRelease) Start() (act action.Action, err error) {
	// In case there are no additional issues, we are done.
	if len(release.additionalIssues) == 0 {
		return action.Noop, nil
	}

	// Set milestone for the additional issues.
	task := fmt.Sprintf(
		"Set milestone to '%v' for the issues added automatically",
		release.trunkVersion.BaseString())
	log.Run(task)

	// Get the milestone corresponding to the release branch version string.
	chain := action.NewActionChain()
	defer chain.RollbackOnError(&err)

	milestone, act, err := release.tracker.getOrCreateMilestone(release.trunkVersion)
	if err != nil {
		return nil, errs.NewError(task, err)
	}
	chain.Push(act)

	// Update the issues.
	issues, act, err := release.tracker.updateIssues(
		release.additionalIssues, setMilestone(milestone), unsetMilestone())
	if err != nil {
		return nil, errs.NewError(task, err)
	}
	chain.Push(act)
	release.additionalIssues = issues

	return chain, nil
}
