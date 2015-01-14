package pivotaltracker

import (
	// Stdlib
	"errors"
	"fmt"
	"os"
	"strings"

	// Internal
	"github.com/salsaflow/salsaflow/action"
	"github.com/salsaflow/salsaflow/errs"
	"github.com/salsaflow/salsaflow/git"
	"github.com/salsaflow/salsaflow/log"
	"github.com/salsaflow/salsaflow/prompt"
	"github.com/salsaflow/salsaflow/releases"
	"github.com/salsaflow/salsaflow/version"

	// Other
	"github.com/salsita/go-pivotaltracker/v5/pivotal"
)

type nextRelease struct {
	trunkVersion     *version.Version
	nextTrunkVersion *version.Version

	config Config

	additionalStories []*pivotal.Story
}

func newNextRelease(
	trunkVersion *version.Version,
	nextTrunkVersion *version.Version,
	config Config,
) (*nextRelease, error) {

	return &nextRelease{
		trunkVersion:     trunkVersion,
		nextTrunkVersion: nextTrunkVersion,
		config:           config,
	}, nil
}

func (release *nextRelease) PromptUserToConfirmStart() (bool, error) {
	var (
		client       = pivotal.NewClient(release.config.UserToken())
		releaseLabel = getReleaseLabel(release.trunkVersion)
	)

	// Collect the commits that modified trunk since the last release.
	task := "Collect the stories that modified trunk"
	log.Run(task)
	commits, err := releases.ListNewTrunkCommits()
	if err != nil {
		return false, errs.NewError(task, err, nil)
	}

	// Get the story IDs associated with these commits.
	tags := git.StoryIdTags(commits)
	ids := make([]string, 0, len(tags))
	for _, tag := range tags {
		// Story-Id for PT is "<project-id>/stories/<story-id>".
		i := strings.LastIndex(tag, "/")
		if i == -1 {
			return false, errs.NewError(task, fmt.Errorf("invalid Story-Id tag: %v", tag), nil)
		}
		ids = append(ids, tag[i+1:])
	}

	// Fetch the collected stories from Pivotal Tracker, if necessary.
	var additional []*pivotal.Story
	if len(ids) != 0 {
		task = "Fetch the collected stories from Pivotal Tracker"
		log.Run(task)

		var err error
		additional, err = listStoriesById(client, release.config.ProjectId(), ids)
		if len(additional) == 0 && err != nil {
			return false, errs.NewError(task, err, nil)
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
	pmLabel := release.config.PointMeLabel()

	// Fetch the already assigned but unpointed stories.
	pmStories, err := searchStories(client, release.config.ProjectId(),
		"label:\"%v\" AND label:\"%v\"", releaseLabel, pmLabel)
	if err != nil {
		return false, errs.NewError(task, err, nil)
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
		err := prompt.ListStories(toCommonStories(pmStories, release.config), os.Stdout)
		if err != nil {
			return false, err
		}
		fmt.Println()
		return false, errs.NewError(task, errors.New("unpointed stories detected"), nil)
	}

	// Print the stories to be added to the release.
	if len(additional) != 0 {
		fmt.Println("\nThe following stories are going to be added to the release:\n")
		err := prompt.ListStories(toCommonStories(additional, release.config), os.Stdout)
		if err != nil {
			return false, err
		}
	}

	// Ask the user to confirm.
	ok, err := prompt.Confirm(
		fmt.Sprintf("\nAre you sure you want to start release %v?", release.trunkVersion))
	if err == nil {
		release.additionalStories = additional
	}
	return ok, err
}

func (release *nextRelease) Start() (action.Action, error) {
	client := pivotal.NewClient(release.config.UserToken())

	// Add release labels to the relevant stories.
	task := "Label the stories with the release label"
	log.Run(task)
	releaseLabel := getReleaseLabel(release.trunkVersion)
	stories, err := addLabel(client, release.config.ProjectId(),
		release.additionalStories, releaseLabel)
	if err != nil {
		return nil, errs.NewError(task, err, nil)
	}
	release.additionalStories = nil

	// Return the rollback action, which removes the release labels that were appended.
	return action.ActionFunc(func() error {
		log.Rollback(task)
		_, err := removeLabel(client, release.config.ProjectId(),
			stories, releaseLabel)
		if err != nil {
			return errs.NewError("Remove the release label from the stories", err, nil)
		}
		return nil
	}), nil
}
