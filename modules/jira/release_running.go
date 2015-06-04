package jira

import (
	// Stdlib
	"bytes"
	"errors"
	"fmt"
	"io"
	"text/tabwriter"

	// Internal
	"github.com/salsaflow/salsaflow/action"
	"github.com/salsaflow/salsaflow/errs"
	"github.com/salsaflow/salsaflow/log"
	"github.com/salsaflow/salsaflow/modules/common"
	"github.com/salsaflow/salsaflow/version"

	// Vendor
	"github.com/salsita/go-jira/v2/jira"
)

type runningRelease struct {
	tracker        *issueTracker
	releaseVersion *version.Version
	issues         []*jira.Issue
}

func newRunningRelease(
	tracker *issueTracker,
	releaseVersion *version.Version,
) (*runningRelease, error) {

	// Fetch relevant issues from JIRA.
	var (
		key          = tracker.config.ProjectKey()
		releaseLabel = releaseVersion.ReleaseTagString()
		task         = fmt.Sprintf("Fetch issues labeled with '%v'", releaseLabel)
	)
	log.Run(task)
	issues, err := issuesByLabel(newClient(tracker.config), key, releaseLabel)
	if err != nil {
		return nil, errs.NewError(task, err, nil)
	}

	// Return a new release instance.
	return &runningRelease{tracker, releaseVersion, issues}, nil
}

func (release *runningRelease) Version() *version.Version {
	return release.releaseVersion
}

func (release *runningRelease) Stories() ([]common.Story, error) {
	return toCommonStories(release.issues, release.tracker), nil
}

func (release *runningRelease) EnsureStageable() error {
	var task = fmt.Sprintf(
		"Make sure release '%v' can be staged", release.releaseVersion.ReleaseTagString())
	log.Run(task)

	var details bytes.Buffer
	tw := tabwriter.NewWriter(&details, 0, 8, 4, '\t', 0)
	io.WriteString(tw, "\n")
	io.WriteString(tw, "Issue Key\tError\n")
	io.WriteString(tw, "=========\t=====\n")

	var (
		err             error
		errNotStageable = errors.New("release cannot be staged")
	)
	for _, issue := range release.issues {
		if ex := ensureStageableIssue(issue); ex != nil {
			fmt.Fprintf(tw, "%v\t%v\n", issue.Key, ex)
			err = errNotStageable
		}
	}

	if err != nil {
		io.WriteString(tw, "\n")
		tw.Flush()
		return errs.NewError(task, err, &details)
	}
	return nil
}

func (release *runningRelease) Stage() (action.Action, error) {
	var (
		api       = newClient(release.tracker.config)
		tag       = release.releaseVersion.ReleaseTagString()
		stageTask = fmt.Sprintf("Stage JIRA release '%v'", tag)
	)
	log.Run(stageTask)

	// Make sure we only try to stage the issues that are in Tested.
	var issuesToStage []*jira.Issue
	for _, issue := range release.issues {
		if issue.Fields.Status.Id == stateIdTested {
			issuesToStage = append(issuesToStage, issue)
		}
	}

	// Perform the transition.
	err := performBulkTransition(api, issuesToStage, transitionIdStage, transitionIdUnstage)
	if err != nil {
		return nil, errs.NewError(stageTask, err, nil)
	}

	return action.ActionFunc(func() error {
		log.Rollback(stageTask)
		unstageTask := fmt.Sprintf("Unstage JIRA release %v", tag)
		if err := performBulkTransition(api, issuesToStage, transitionIdUnstage, ""); err != nil {
			return errs.NewError(unstageTask, err, nil)
		}
		return nil
	}), nil
}

func (release *runningRelease) Releasable() (bool, error) {
	// Drop accepted issues.
	var notAccepted []*jira.Issue
IssueLoop:
	for _, issue := range release.issues {
		for _, id := range acceptedStateIds {
			if id == issue.Fields.Status.Id {
				continue IssueLoop
			}
		}
		notAccepted = append(notAccepted, issue)
	}

	// Return the result.
	return len(notAccepted) == 0, nil
}

func (release *runningRelease) Release() error {
	if release.issues == nil {
		panic("bug(release.issues == nil)")
	}
	return performBulkTransition(
		newClient(release.tracker.config), release.issues, transitionIdRelease, "")
}

func ensureStageableIssue(issue *jira.Issue) error {
	// Check subtasks recursively.
	for _, subtask := range issue.Fields.Subtasks {
		if err := ensureStageableIssue(subtask); err != nil {
			return err
		}
	}

	// Check the issue itself.
	for _, id := range stageableStateIds {
		if issue.Fields.Status.Id == id {
			return nil
		}
	}
	return fmt.Errorf("issue %v: invalid state: %v", issue.Key, issue.Fields.Status.Name)
}
