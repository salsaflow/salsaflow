package jira

import (
	// Stdlib
	"container/list"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"
	"text/tabwriter"

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

	// Convert []*jira.Issue into []*assignedIssue.
	assigned := make([]*assignedIssue, 0, len(issues))
	for _, issue := range issues {
		assigned = append(assigned, &assignedIssue{
			Issue:  issue,
			Reason: "assigned manually",
		})
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
	for _, issue := range collectedIssues {
		assigned = append(assigned, &assignedIssue{
			Issue:  issue,
			Reason: "modified trunk",
		})
	}

	// Get all issues that are reachable from the collected issues.
	log.Run("Collect the issues to be assigned automatically")
	allIssues, err := release.computeClosure(assigned)

	// Split the result by what is already assigned and what is to be assigned.
	var (
		alreadyAssigned  = make([]*assignedIssue, 0, len(allIssues))
		tasksToAssign    = make([]*assignedIssue, 0, len(allIssues))
		subtasksToAssign = make([]*assignedIssue, 0, len(allIssues))
	)
	for _, issue := range allIssues {
		if isLabeled(issue.Issue, verLabel) {
			alreadyAssigned = append(alreadyAssigned, issue)
			continue
		}

		if issue.Fields.IssueType.Subtask {
			subtasksToAssign = append(subtasksToAssign, issue)
		} else {
			tasksToAssign = append(tasksToAssign, issue)
		}
	}

	// Present the issues to the user.
	issueLists := []*issueDialogSection{
		{
			"The following issues were manually assigned to the release:",
			alreadyAssigned,
			true,
		},
		{
			"The following top-level issues are going to be assigned by SalsaFlow:",
			tasksToAssign,
			false,
		},
		{
			"The following subtasks are going to be assigned by SalsaFlow as well:",
			subtasksToAssign,
			false,
		},
	}
	for _, l := range issueLists {
		listAssignedIssues(l, os.Stdout)
	}

	// Ask the user to confirm.
	ok, err := prompt.Confirm(
		fmt.Sprintf("\nAre you sure you want to start release %v?", verString))
	if err == nil {
		// Need to make []*jira.Issue out of []*assignedIssues, annoying...
		issues := append(tasksToAssign, subtasksToAssign...)
		additional := make([]*jira.Issue, 0, len(issues))
		for _, issue := range issues {
			additional = append(additional, issue.Issue)
		}
		// Store the issues to be labeled in the release object.
		release.additionalIssues = additional
	}
	return ok, err
}

type issueDialogSection struct {
	message    string
	issues     []*assignedIssue
	skipReason bool
}

func listAssignedIssues(section *issueDialogSection, writer io.Writer) {
	// Do nothing when no issues.
	if len(section.issues) == 0 {
		return
	}

	// Sort the issues by issue key.
	sort.Sort(assignedIssues(section.issues))

	tw := tabwriter.NewWriter(writer, 0, 8, 2, '\t', 0)

	// Write the message.
	fmt.Fprintln(tw)
	fmt.Fprintln(tw, section.message)
	fmt.Fprintln(tw)

	// Write the issues.
	if section.skipReason {
		fmt.Fprint(tw, "  Issue Key\tSummary\n")
		fmt.Fprint(tw, "  =========\t=======\n")
		for _, issue := range section.issues {
			fmt.Fprintf(tw, "  %v\t%v\n", issue.Key, issue.Fields.Summary)
		}
	} else {
		fmt.Fprint(tw, "  Issue Key\tSummary\tReason\n")
		fmt.Fprint(tw, "  =========\t=======\t======\n")
		for _, issue := range section.issues {
			fmt.Fprintf(tw, "  %v\t%v\t%v\n", issue.Key, issue.Fields.Summary, issue.Reason)
		}
	}

	// Flush the tabwriter.
	tw.Flush()
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

type assignedIssue struct {
	*jira.Issue
	Reason string
}

// assignedIssues implements sort.Interface
// The issues are sorted by the issue key.
type assignedIssues []*assignedIssue

func (as assignedIssues) Len() int {
	return len(as)
}

func (as assignedIssues) Less(i, j int) bool {
	seq := func(index int) int {
		var part string
		parts := strings.SplitN(as[index].Key, "-", 2)
		if len(parts) == 2 {
			part = parts[1]
		} else {
			part = parts[0]
		}
		seq, _ := strconv.Atoi(part)
		return seq
	}

	return seq(i) < seq(j)
}

func (as assignedIssues) Swap(i, j int) {
	as[i], as[j] = as[j], as[i]
}

// computeClosure returns all issues that are reachable from the given list of issues
// by following the parent/subtask relations. The issues returned are by definition
// a superset of the issues passed into the function.
func (release *nextRelease) computeClosure(issues []*assignedIssue) ([]*assignedIssue, error) {
	closure := make([]*assignedIssue, 0, len(issues))

	// We push the collected issues onto a stack and we loop over.
	// During every iteration, we pop an issue, remember it in case it is not labeled,
	// then we push the parent and all the subtasks to the stack to check them later.
	// processedKeys is used to remember what issue keys were checked already.
	processedKeys := make(map[string]struct{}, len(issues))

	processed := func(issue *jira.Issue) bool {
		_, ok := processedKeys[issue.Key]
		return ok
	}

	markAsProcessed := func(issue *jira.Issue) {
		processedKeys[issue.Key] = struct{}{}
	}

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
		issue := e.Value.(*assignedIssue)

		// In case this issue has already been processed, continue.
		if processed(issue.Issue) {
			continue
		}

		// Add the issue into the closure.
		closure = append(closure, issue)

		// Push the parent task onto the stack.
		if parent := issue.Fields.Parent; parent != nil && !processed(parent) {
			// We need to fetch the parent issue to get the list of subtasks.
			// The parent link in the subtask issue is not a complete issue resource.
			task := fmt.Sprintf("Fetch additional JIRA issue resource: %v", parent.Key)
			log.Run(task)
			parentIssue, err := release.tracker.issueByIdOrKey(parent.Key)
			if err != nil {
				return nil, errs.NewError(task, err)
			}
			stack.PushBack(&assignedIssue{
				Issue:  parentIssue,
				Reason: fmt.Sprintf("parent of %v", issue.Key),
			})
		}

		// Push the subtasks onto the stack.
		for _, child := range issue.Fields.Subtasks {
			// No need to fetch additional resources as with the parent link right above.
			// The subtask resource is incomplete, but we are not going to use that resource
			// to get the parent link since the parent issue is being processed right now.
			if !processed(child) {
				stack.PushBack(&assignedIssue{
					Issue:  child,
					Reason: fmt.Sprintf("subtask of %v", issue.Key),
				})
			}
		}

		// Mark the issue as processed.
		markAsProcessed(issue.Issue)
	}

	return closure, nil
}
