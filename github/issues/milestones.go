package issues

import (
	// Stdlib
	"fmt"

	// Internal
	"github.com/salsaflow/salsaflow/action"
	"github.com/salsaflow/salsaflow/errs"
	"github.com/salsaflow/salsaflow/log"

	// Vendor
	"github.com/google/go-github/github"
)

func CreateMilestone(
	client *github.Client,
	owner string,
	repo string,
	title string,
) (*github.Milestone, action.Action, error) {

	// Create the milestone.
	milestoneTask := fmt.Sprintf("Create GitHub milestone '%v'", title)
	log.Run(milestoneTask)
	milestone, _, err := client.Issues.CreateMilestone(owner, repo, &github.Milestone{
		Title: github.String(title),
	})
	if err != nil {
		return nil, nil, errs.NewError(milestoneTask, err)
	}

	// Return a rollback function.
	return milestone, action.ActionFunc(func() error {
		log.Rollback(milestoneTask)
		task := fmt.Sprintf("Delete GitHub milestone '%v'", title)
		_, err := client.Issues.DeleteMilestone(owner, repo, *milestone.Number)
		if err != nil {
			return errs.NewError(task, err)
		}
		return nil
	}), nil
}

func FindMilestoneByTitle(
	client *github.Client,
	owner string,
	repo string,
	title string,
) (*github.Milestone, error) {

	// Fetch milestones for the given repository.
	task := fmt.Sprintf("Search for GitHub milestone '%v'", title)
	log.Run(task)
	milestones, _, err := client.Issues.ListMilestones(owner, repo, nil)
	if err != nil {
		return nil, errs.NewError(task, err)
	}

	// Find the right one.
	for _, milestone := range milestones {
		if *milestone.Title == title {
			return &milestone, nil
		}
	}

	// Milestone not found.
	return nil, nil
}

func GetOrCreateMilestoneForTitle(
	client *github.Client,
	owner string,
	repo string,
	title string,
) (*github.Milestone, action.Action, error) {

	// Try to get the milestone.
	milestone, err := FindMilestoneByTitle(client, owner, repo, title)
	if err != nil {
		return nil, nil, err
	}
	if milestone != nil {
		// Milestone found, return it.
		log.Log(fmt.Sprintf("GitHub milestone '%v' already exists", title))
		return milestone, nil, nil
	}

	// Create the milestone when not found.
	return CreateMilestone(client, owner, repo, title)
}

func CloseMilestone(
	client *github.Client,
	owner string,
	repo string,
	milestone *github.Milestone,
) (*github.Milestone, action.Action, error) {

	// Copy the milestone to have it stored locally for the rollback closure.
	mstone := *milestone

	// A helper closure.
	setState := func(milestone *github.Milestone, state string) (*github.Milestone, error) {
		task := fmt.Sprintf("Mark GitHub milestone '%v' as %v", *milestone.Title, state)
		log.Run(task)
		m, _, err := client.Issues.EditMilestone(owner, repo, *milestone.Number, &github.Milestone{
			State: &state,
		})
		if err != nil {
			return nil, errs.NewError(task, err)
		}
		return m, nil
	}

	// Close the chosen milestone.
	m, err := setState(&mstone, "closed")
	if err != nil {
		return nil, nil, err
	}

	// Return the rollback function.
	act := action.ActionFunc(func() error {
		_, err := setState(&mstone, "open")
		return err
	})
	return m, act, nil
}
