package jira

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
	verString := releaseVersion.BaseString()
	task := fmt.Sprintf("Fetch JIRA issues for release %v", verString)
	log.Run(task)
	issues, err := tracker.issuesByRelease(releaseVersion)
	if err != nil {
		return nil, errs.NewError(task, err)
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
	versionString := release.releaseVersion.BaseString()

	var task = fmt.Sprintf(
		"Make sure that release %v can be staged", versionString)
	log.Run(task)

	var details bytes.Buffer
	tw := tabwriter.NewWriter(&details, 0, 8, 4, '\t', 0)
	io.WriteString(tw, "\n")
	io.WriteString(tw, "Issue Key\tError\n")
	io.WriteString(tw, "=========\t=====\n")

	var err error
	for _, issue := range release.issues {
		if ex := ensureStageableIssue(issue); ex != nil {
			fmt.Fprintf(tw, "%v\t%v\n", issue.Key, ex)
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

func (release *runningRelease) Stage() (action.Action, error) {

	var (
		api           = newClient(release.tracker.config)
		versionString = release.releaseVersion.BaseString()
		stageTask     = fmt.Sprintf("Stage JIRA issues associated with release %v", versionString)
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
		return nil, errs.NewError(stageTask, err)
	}

	return action.ActionFunc(func() error {
		log.Rollback(stageTask)
		unstageTask := fmt.Sprintf("Unstage JIRA issues associated with release %v", versionString)
		if err := performBulkTransition(api, issuesToStage, transitionIdUnstage, ""); err != nil {
			return errs.NewError(unstageTask, err)
		}
		return nil
	}), nil
}

func (release *runningRelease) EnsureReleasable() error {
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

	// In case there is no open story, we are done.
	if len(notAccepted) == 0 {
		return nil
	}

	// Generate the error hint.
	var hint bytes.Buffer
	tw := tabwriter.NewWriter(&hint, 0, 8, 2, '\t', 0)
	fmt.Fprintf(tw, "\nThe following issues cannot be released:\n\n")
	fmt.Fprintf(tw, "Issue Key\tStatus\n")
	fmt.Fprintf(tw, "=========\t======\n")
	for _, issue := range notAccepted {
		fmt.Fprintf(tw, "%v\t%v\n", issue.Key, issue.Fields.Status.Name)
	}
	fmt.Fprintf(tw, "\n")
	tw.Flush()

	versionString := release.releaseVersion.BaseString()
	return errs.NewErrorWithHint(
		fmt.Sprintf("Make sure release %v can be released", versionString),
		common.ErrNotReleasable,
		hint.String())
}

func (release *runningRelease) Release() error {
	// Release all issues that are accepted.
	issues := make([]*jira.Issue, 0, len(release.issues))
	for _, issue := range release.issues {
		if issue.Fields.Status.Id == stateIdAccepted {
			issues = append(issues, issue)
		}
	}
	if len(issues) == 0 {
		log.Warn("No accepted stories found in JIRA")
		return nil
	}

	return performBulkTransition(
		newClient(release.tracker.config), issues, transitionIdRelease, "")
}

func ensureStageableIssue(issue *jira.Issue) error {
	// Make sure the issue is in one of the stageable stages.
	for _, id := range stageableStateIds {
		if issue.Fields.Status.Id == id {
			return nil
		}
	}
	return fmt.Errorf("issue %v: invalid state: %v", issue.Key, issue.Fields.Status.Name)
}
