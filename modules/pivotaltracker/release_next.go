package pivotaltracker

import (
	// Stdlib
	"errors"
	"fmt"
	"os"

	// Internal
	"github.com/salsaflow/salsaflow/action"
	"github.com/salsaflow/salsaflow/errs"
	"github.com/salsaflow/salsaflow/log"
	"github.com/salsaflow/salsaflow/prompt"
	"github.com/salsaflow/salsaflow/releases"
	"github.com/salsaflow/salsaflow/version"

	// Other
	"github.com/salsita/go-pivotaltracker/v5/pivotal"
)

type nextRelease struct {
	tracker *issueTracker

	trunkVersion     *version.Version
	nextTrunkVersion *version.Version

	additionalStories []*pivotal.Story
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
	var (
		config       = release.tracker.config
		client       = pivotal.NewClient(config.UserToken())
		releaseLabel = getReleaseLabel(release.trunkVersion)
	)

	// Collect the commits that modified trunk since the last release.
	task := "Collect the stories that modified trunk"
	log.Run(task)
	ids, err := releases.ListStoryIdsToBeAssigned(release.tracker)
	if err != nil {
		return false, errs.NewError(task, err)
	}

	// Fetch the collected stories from Pivotal Tracker, if necessary.
	var additional []*pivotal.Story
	if len(ids) != 0 {
		task = "Fetch the collected stories from Pivotal Tracker"
		log.Run(task)

		var err error
		additional, err = listStoriesById(client, config.ProjectId(), ids)
		if len(additional) == 0 && err != nil {
			return false, errs.NewError(task, err)
		}
		if len(additional) != len(ids) {
			log.Warn("Some stories were dropped since they were not found in PT")
		}

		// Drop the issues that are already assigned to the right release.
		unassignedStories := make([]*pivotal.Story, 0, len(additional))
		for _, story := range additional {
			if labeled(story, releaseLabel) {
				continue
			}
			unassignedStories = append(unassignedStories, story)
		}
		additional = unassignedStories
	}

	// Check the Point Me label.
	task = "Make sure there are no unpointed stories"
	log.Run(task)
	pmLabel := config.PointMeLabel()

	// Fetch the already assigned but unpointed stories.
	pmStories, err := searchStories(client, config.ProjectId(),
		"label:\"%v\" AND label:\"%v\"", releaseLabel, pmLabel)
	if err != nil {
		return false, errs.NewError(task, err)
	}
	// Also add these that are to be added but are unpointed.
	for _, story := range additional {
		if labeled(story, pmLabel) {
			pmStories = append(pmStories, story)
		}
	}
	// In case there are some unpointed stories, stop the release.
	if len(pmStories) != 0 {
		fmt.Println("\nThe following stories are still yet to be pointed:\n")
		err := prompt.ListStories(toCommonStories(pmStories, release.tracker), os.Stdout)
		if err != nil {
			return false, err
		}
		fmt.Println()
		return false, errs.NewError(task, errors.New("unpointed stories detected"))
	}

	// Print the stories to be added to the release.
	if len(additional) != 0 {
		fmt.Println("\nThe following stories are going to be added to the release:\n")
		err := prompt.ListStories(toCommonStories(additional, release.tracker), os.Stdout)
		if err != nil {
			return false, err
		}
	}

	// Ask the user to confirm.
	ok, err := prompt.Confirm(
		fmt.Sprintf(
			"\nAre you sure you want to start release %v?",
			release.trunkVersion.BaseString()))
	if err == nil {
		release.additionalStories = additional
	}
	return ok, err
}

func (release *nextRelease) Start() (action.Action, error) {
	// In case there are no additional stories, we are done.
	if len(release.additionalStories) == 0 {
		return action.Noop, nil
	}

	// Add release labels to the relevant stories.
	var (
		config    = release.tracker.config
		client    = pivotal.NewClient(config.UserToken())
		projectId = config.ProjectId()
	)

	task := "Label the stories with the release label"
	log.Run(task)
	releaseLabel := getReleaseLabel(release.trunkVersion)
	stories, err := addLabel(client, projectId,
		release.additionalStories, releaseLabel)
	if err != nil {
		return nil, errs.NewError(task, err)
	}
	release.additionalStories = nil

	// Return the rollback action, which removes the release labels that were appended.
	return action.ActionFunc(func() error {
		log.Rollback(task)
		_, err := removeLabel(client, projectId, stories, releaseLabel)
		if err != nil {
			return errs.NewError("Remove the release label from the stories", err)
		}
		return nil
	}), nil
}
