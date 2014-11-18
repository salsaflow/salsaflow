package jira

import (
	// Stdlib
	"bytes"
	"errors"
	"fmt"
	"io"
	"text/tabwriter"

	// Internal
	"github.com/salsita/salsaflow/action"
	"github.com/salsita/salsaflow/errs"
	"github.com/salsita/salsaflow/log"
	"github.com/salsita/salsaflow/modules/common"
	"github.com/salsita/salsaflow/modules/jira/client"
	"github.com/salsita/salsaflow/version"
)

type runningRelease struct {
	releaseVersion *version.Version
	issues         []*client.Issue
	config         Config
}

func newRunningRelease(
	tracker *issueTracker,
	releaseVersion *version.Version,
) (*runningRelease, error) {

	// Fetch relevant issues from JIRA.
	task := "Fetch data from JIRA"
	log.Run(task)
	var (
		key = tracker.config.ProjectKey()
		tag = releaseVersion.ReleaseTagString()
	)
	query := fmt.Sprintf("project = %v AND fixVersion = \"%v\"", key, tag)
	issues, err := search(newClient(tracker.config), query)
	if err != nil {
		return nil, err
	}

	// Return a new release instance.
	return &runningRelease{releaseVersion, issues, tracker.config}, nil
}

func (release *runningRelease) Stories() ([]common.Story, error) {
	return toCommonStories(release.issues, release.config), nil
}

func (release *runningRelease) EnsureStageable() error {
	var task = fmt.Sprintf(
		"Make sure JIRA version '%v' is stageable", release.releaseVersion.ReleaseTagString())
	log.Run(task)

	var details bytes.Buffer
	tw := tabwriter.NewWriter(&details, 0, 8, 4, '\t', 0)
	io.WriteString(tw, "\n")
	io.WriteString(tw, "Issue Key\tError\n")
	io.WriteString(tw, "=========\t=====\n")

	var (
		err             error
		errNotStageable = errors.New("release not stageable")
	)
IssueLoop:
	for _, issue := range release.issues {
		for _, id := range stageableStateIds {
			if issue.Fields.Status.Id == id {
				continue IssueLoop
			}
		}
		fmt.Fprintf(tw, "%v\tNot a stageable state: %v\n", issue.Key, issue.Fields.Status.Name)
		err = errNotStageable
	}

	if err != nil {
		io.WriteString(tw, "\n")
		tw.Flush()
		return errs.NewError(task, err, &details)
	}
	return nil
}

func (release *runningRelease) Stage() (action.Action, error) {
	tag := release.releaseVersion.ReleaseTagString()
	stageTask := fmt.Sprintf("Stage JIRA version '%v'", tag)
	log.Run(stageTask)
	api := newClient(release.config)
	err := performBulkTransition(api, release.issues, transitionIdStage, transitionIdUnstage)
	if err != nil {
		return nil, errs.NewError(stageTask, err, nil)
	}

	return action.ActionFunc(func() error {
		log.Rollback(stageTask)
		unstageTask := fmt.Sprintf("Unstage JIRA version %v", tag)
		if err := performBulkTransition(api, release.issues, transitionIdUnstage, ""); err != nil {
			return errs.NewError(unstageTask, err, nil)
		}
		return nil
	}), nil
}
