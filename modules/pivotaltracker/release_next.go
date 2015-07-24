package pivotaltracker

import (
	// Stdlib
	"errors"
	"fmt"
	"os"
	"strconv"

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
	// Fetch the stories already assigned to the release.
	var (
		ver       = release.trunkVersion
		verString = ver.BaseString()
		verLabel  = ver.ReleaseTagString()
	)
	task := fmt.Sprintf("Fetch the stories already assigned to release %v", verString)
	log.Run(task)
	assignedStories, err := release.tracker.storiesByRelease(ver)
	if err != nil {
		return false, errs.NewError(task, err)
	}

	// Collect the story IDs associated with the commits that
	// modified trunk since the last release.
	task = "Collect the stories that modified trunk since the last release"
	log.Run(task)
	storyIds, err := releases.ListStoryIdsToBeAssigned(release.tracker)
	if err != nil {
		return false, errs.NewError(task, err)
	}

	// Drop the stories that are already assigned.
	idSet := make(map[string]struct{}, len(assignedStories))
	for _, story := range assignedStories {
		idSet[strconv.Itoa(story.Id)] = struct{}{}
	}
	ids := make([]string, 0, len(storyIds))
	for _, id := range storyIds {
		if _, ok := idSet[id]; !ok {
			ids = append(ids, id)
		}
	}
	storyIds = ids

	// Fetch the collected stories from Pivotal Tracker, if necessary.
	var additionalStories []*pivotal.Story
	if len(storyIds) != 0 {
		task = "Fetch the collected stories from Pivotal Tracker"
		log.Run(task)

		var err error
		additionalStories, err = release.tracker.storiesById(storyIds)
		if len(additionalStories) == 0 && err != nil {
			return false, errs.NewError(task, err)
		}
		if len(additionalStories) != len(storyIds) {
			log.Warn("Some stories were dropped since they were not found in PT")
		}
	}

	// Check the Point Me label.
	task = "Make sure there are no unpointed stories"
	log.Run(task)
	pmLabel := release.tracker.config.PointMeLabel()

	// Fetch the already assigned but unpointed stories.
	pmStories, err := release.tracker.searchStories(
		"label:\"%v\" AND label:\"%v\"", verLabel, pmLabel)
	if err != nil {
		return false, errs.NewError(task, err)
	}
	// Also add these that are to be added but are unpointed.
	for _, story := range additionalStories {
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

	// Print the summary into the console.
	summary := []struct {
		header  string
		stories []*pivotal.Story
	}{
		{
			"The following stories were manually assigned to the release:",
			assignedStories,
		},
		{
			"The following stories were added automatically (modified trunk):",
			additionalStories,
		},
	}
	for _, item := range summary {
		if len(item.stories) != 0 {
			fmt.Println()
			fmt.Println(item.header)
			fmt.Println()
			err := prompt.ListStories(toCommonStories(item.stories, release.tracker), os.Stdout)
			if err != nil {
				return false, err
			}
		}
	}

	// Ask the user to confirm.
	ok, err := prompt.Confirm(
		fmt.Sprintf(
			"\nAre you sure you want to start release %v?",
			release.trunkVersion.BaseString()), false)
	if err == nil {
		release.additionalStories = additionalStories
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
