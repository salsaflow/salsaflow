package github

import (
	// Stdlib
	"bytes"
	"fmt"
	"io"
	"text/tabwriter"

	// Internal
	"github.com/salsaflow/salsaflow/action"
	"github.com/salsaflow/salsaflow/errs"
	"github.com/salsaflow/salsaflow/log"
	"github.com/salsaflow/salsaflow/modules/common"
	"github.com/salsaflow/salsaflow/version"

	// Other
	"github.com/google/go-github/github"
)

type runningRelease struct {
	tracker *issueTracker
	version *version.Version
	issues  []*github.Issue
}

func newRunningRelease(
	tracker *issueTracker,
	releaseVersion *version.Version,
) *runningRelease {

	return &runningRelease{
		tracker: tracker,
		version: releaseVersion,
	}
}

// Version is a part of common.RunningRelease interface.
func (release *runningRelease) Version() *version.Version {
	return release.version
}

// Stories is a part of common.RunningRelease interface.
func (release *runningRelease) Stories() ([]common.Story, error) {
	issues, err := release.loadIssues()
	if err != nil {
		return nil, err
	}
	return toCommonStories(issues, release.tracker), nil
}

// EnsureStageable is a part of common.RunningRelease interface.
func (release *runningRelease) EnsureStageable() error {
	task := "Make sure the issues can be staged"
	log.Run(task)

	// Load the assigned issues.
	issues, err := release.loadIssues()
	if err != nil {
		return errs.NewError(task, err)
	}

	// Check the states.
	var details bytes.Buffer
	tw := tabwriter.NewWriter(&details, 0, 8, 4, '\t', 0)
	io.WriteString(tw, "\n")
	io.WriteString(tw, "Issue URL\tError\n")
	io.WriteString(tw, "=========\t=====\n")

	skipLabels := release.tracker.config.SkipCheckLabels
	shouldBeSkipped := func(issue *github.Issue) bool {
		for _, skipLabel := range skipLabels {
			if labeled(issue, skipLabel) {
				return true
			}
		}
		return false
	}

	for _, issue := range issues {
		// Skip the story in case it is labeled with a skip label.
		if shouldBeSkipped(issue) {
			continue
		}

		// Check the abstract state.
		var pwned bool
		state := abstractState(issue, release.tracker.config)
		switch state {
		case common.StoryStateNew:
			pwned = true
		case common.StoryStateApproved:
			pwned = true
		case common.StoryStateBeingImplemented:
			pwned = true
		case common.StoryStateImplemented:
			pwned = true
		case common.StoryStateReviewed:
			pwned = true
		case common.StoryStateBeingTested:
			pwned = true
		case common.StoryStateTested:
			// OK
		case common.StoryStateStaged:
			// OK
		case common.StoryStateAccepted:
			// OK
		case common.StoryStateRejected:
			pwned = true
		case common.StoryStateClosed:
			// OK
		case common.StoryStateInvalid:
			pwned = true
		default:
			panic("unknown abstract issues state")
		}

		if pwned {
			fmt.Fprintf(tw, "%v\tinvalid state: %v\n", *issue.HTMLURL, state)
			err = common.ErrNotStageable
		}
	}
	if err != nil {
		io.WriteString(tw, "\n")
		tw.Flush()
		return errs.NewErrorWithHint(task, err, details.String())
	}
	return nil
}

// Stage is a part of common.RunningRelease interface.
func (release *runningRelease) Stage() (action.Action, error) {
	stageTask := fmt.Sprintf("Mark relevant GitHub issues as %v", common.StoryStateStaged)
	log.Run(stageTask)

	// Load the assigned issues.
	issues, err := release.loadIssues()
	if err != nil {
		return nil, errs.NewError(stageTask, err)
	}

	// Pick only the stories that need staging.
	issues = filterIssues(issues, func(issue *github.Issue) bool {
		return abstractState(issue, release.tracker.config) == common.StoryStateTested
	})

	// Update the issues.
	toLabelStrings := func(labels []github.Label) []string {
		ls := make([]string, 0, len(labels))
		for _, label := range labels {
			ls = append(ls, *label.Name)
		}
		return ls
	}

	var (
		remainingLabels = make(map[int][]string, len(issues))
		prunedLabels    = make(map[int][]string, len(issues))
	)
	for _, issue := range issues {
		remaining, pruned := pruneStateLabels(release.tracker.config, issue.Labels)
		remainingLabels[*issue.Number] = toLabelStrings(remaining)
		prunedLabels[*issue.Number] = toLabelStrings(pruned)
	}

	// updateFunc sets the workflow label to express the staged abstract state.
	updateFunc := func(
		client *github.Client,
		owner string,
		repo string,
		issue *github.Issue,
	) (*github.Issue, error) {

		issueNum := *issue.Number
		labels := remainingLabels[*issue.Number]
		labels = append(labels, release.tracker.config.StagedLabel)
		updatedLabels, _, err := client.Issues.ReplaceLabelsForIssue(owner, repo, issueNum, labels)
		if err != nil {
			return nil, err
		}

		i := *issue
		i.Labels = updatedLabels
		return &i, nil
	}

	// rollbackFunc resets the labels to the original value.
	rollbackFunc := func(
		client *github.Client,
		owner string,
		repo string,
		issue *github.Issue,
	) (*github.Issue, error) {

		issueNum := *issue.Number
		labels := append(remainingLabels[issueNum], prunedLabels[issueNum]...)
		updatedLabels, _, err := client.Issues.ReplaceLabelsForIssue(owner, repo, issueNum, labels)
		if err != nil {
			return nil, err
		}

		i := *issue
		i.Labels = updatedLabels
		return &i, nil
	}

	// Update the issues concurrently.
	updatedStories, act, err := release.tracker.updateIssues(issues, updateFunc, rollbackFunc)
	if err != nil {
		return nil, errs.NewError(stageTask, err)
	}
	release.issues = updatedStories

	// Return the rollback function.
	return action.ActionFunc(func() error {
		release.issues = nil
		return act.Rollback()
	}), nil
}

// EnsureClosable is a part of common.RunningRelease interface.
func (release *runningRelease) EnsureClosable() error {
	var (
		config  = release.tracker.config
		vString = release.version.BaseString()
	)

	task := fmt.Sprintf(
		"Make sure that the issues associated with release %v can be released", vString)
	log.Run(task)

	// Make sure the issues are loaded.
	issues, err := release.loadIssues()
	if err != nil {
		return errs.NewError(task, err)
	}

	// Make sure all relevant issues are accepted.
	// This includes the issues with SkipCheckLabels.
	notAccepted := filterIssues(issues, func(issue *github.Issue) bool {
		return abstractState(issue, config) != common.StoryStateAccepted
	})

	// In case there are no issues in a wrong state, we are done.
	if len(notAccepted) == 0 {
		return nil
	}

	// Generate the error hint.
	var hint bytes.Buffer
	tw := tabwriter.NewWriter(&hint, 0, 8, 2, '\t', 0)
	fmt.Fprintf(tw, "\nThe following issues are blocking the release:\n\n")
	fmt.Fprintf(tw, "Issue URL\tState\n")
	fmt.Fprintf(tw, "=========\t=====\n")
	for _, issue := range notAccepted {
		fmt.Fprintf(tw, "%v\t%v\n", *issue.HTMLURL, abstractState(issue, config))
	}
	fmt.Fprintf(tw, "\n")
	tw.Flush()

	return errs.NewErrorWithHint(task, common.ErrNotClosable, hint.String())
}

// Close is a part of common.RunningRelease interface.
func (release *runningRelease) Close() (action.Action, error) {
	_, act, err := release.tracker.closeMilestone(release.version)
	return act, err
}

func (release *runningRelease) loadIssues() ([]*github.Issue, error) {
	// Fetch the issues unless cached.
	if release.issues == nil {
		task := fmt.Sprintf(
			"Fetch GitHub issues associated with release %v", release.version.BaseString())
		log.Run(task)
		issues, err := release.tracker.issuesByRelease(release.version)
		if err != nil {
			return nil, errs.NewError(task, err)
		}
		release.issues = issues
	}

	// Return the cached issues.
	return release.issues, nil
}
