package github

import (
	// Stdlib
	"errors"
	"fmt"

	// Internal
	"github.com/salsaflow/salsaflow/action"
	"github.com/salsaflow/salsaflow/errs"
	ghutil "github.com/salsaflow/salsaflow/github"
	ghissues "github.com/salsaflow/salsaflow/github/issues"
	"github.com/salsaflow/salsaflow/log"
	"github.com/salsaflow/salsaflow/modules/common"
	"github.com/salsaflow/salsaflow/version"

	// Vendor
	"github.com/google/go-github/github"
)

type release struct {
	tool *codeReviewTool
	v    *version.Version

	client *github.Client
	owner  string
	repo   string

	closingMilestone *github.Milestone
}

func newRelease(tool *codeReviewTool, v *version.Version) *release {
	return &release{
		tool: tool,
		v:    v,
	}
}

func (r *release) Initialise() (action.Action, error) {
	// Prepare for API calls.
	client, owner, repo, err := r.prepareForApiCalls()
	if err != nil {
		return nil, err
	}

	// Check whether the review milestone exists or not.
	// People can create milestones manually, so this makes the thing more robust.
	title := milestoneTitle(r.v)
	task := fmt.Sprintf(
		"Check whether GitHub review milestone exists for release %v", r.v.BaseString())
	log.Run(task)
	_, act, err := ghissues.GetOrCreateMilestoneForTitle(client, owner, repo, title)
	if err != nil {
		return nil, errs.NewError(task, err)
	}
	return act, nil
}

func (r *release) EnsureClosable() error {
	// Prepare for API calls.
	_, owner, repo, err := r.prepareForApiCalls()
	if err != nil {
		return err
	}

	// Get the relevant review milestone.
	releaseString := r.v.BaseString()
	task := fmt.Sprintf("Get GitHub review milestone for release %v", releaseString)
	log.Run(task)
	milestone, err := milestoneForVersion(r.tool.config, owner, repo, r.v)
	if err != nil {
		return errs.NewError(task, err)
	}
	if milestone == nil {
		return errs.NewErrorWithHint(task, errors.New("milestone not found"),
			fmt.Sprintf("\nMake sure the review milestone for release %v exists\n\n", r.v))
	}

	// Close the milestone unless there are some issues open.
	task = fmt.Sprintf(
		"Make sure the review milestone for release %v can be closed", releaseString)
	if num := *milestone.OpenIssues; num != 0 {
		hint := fmt.Sprintf(
			"\nreview milestone for release %v cannot be closed: %v issue(s) open\n\n",
			releaseString, num)
		return errs.NewErrorWithHint(task, common.ErrNotClosable, hint)
	}
	r.closingMilestone = milestone
	return nil
}

func (r *release) Close() (action.Action, error) {
	// Make sure EnsureClosable has been called.
	if r.closingMilestone == nil {
		if err := r.EnsureClosable(); err != nil {
			return nil, err
		}
	}

	// Prepare for API calls.
	client, owner, repo, err := r.prepareForApiCalls()
	if err != nil {
		return nil, err
	}

	// Close the milestone.
	releaseString := r.v.BaseString()
	milestoneTask := fmt.Sprintf("Close GitHub review milestone for release %v", releaseString)
	log.Run(milestoneTask)
	milestone, _, err := client.Issues.EditMilestone(
		owner, repo, *r.closingMilestone.Number, &github.Milestone{
			State: github.String("closed"),
		})
	if err != nil {
		return nil, errs.NewError(milestoneTask, err)
	}
	r.closingMilestone = milestone

	// Return a rollback function.
	return action.ActionFunc(func() error {
		log.Rollback(milestoneTask)
		task := fmt.Sprintf("Reopen GitHub review milestone for release %v", releaseString)
		milestone, _, err := client.Issues.EditMilestone(
			owner, repo, *r.closingMilestone.Number, &github.Milestone{
				State: github.String("open"),
			})
		if err != nil {
			return errs.NewError(task, err)
		}
		r.closingMilestone = milestone
		return nil
	}), nil
}

func (r *release) prepareForApiCalls() (client *github.Client, owner, repo string, err error) {
	if r.client == nil {
		r.client = ghutil.NewClient(r.tool.config.Token)
	}

	if r.owner == "" || r.repo == "" {
		var err error
		r.owner, r.repo, err = ghutil.ParseUpstreamURL()
		if err != nil {
			return nil, "", "", err
		}
	}

	return r.client, r.owner, r.repo, nil
}
