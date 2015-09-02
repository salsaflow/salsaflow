package pivotaltracker

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
	"gopkg.in/salsita/go-pivotaltracker.v1/v5/pivotal"
)

type runningRelease struct {
	version *version.Version
	stories []*pivotal.Story
	tracker *issueTracker
}

func newRunningRelease(releaseVersion *version.Version, tracker *issueTracker) (*runningRelease, error) {
	return &runningRelease{
		version: releaseVersion,
		tracker: tracker,
	}, nil
}

func (release *runningRelease) Version() *version.Version {
	return release.version
}

func (release *runningRelease) Stories() ([]common.Story, error) {
	stories, err := release.loadStories()
	if err != nil {
		return nil, err
	}
	return toCommonStories(stories, release.tracker), nil
}

func (release *runningRelease) EnsureStageable() error {
	task := "Make sure the stories can be staged"
	log.Run(task)

	// Load the assigned stories.
	stories, err := release.loadStories()
	if err != nil {
		return errs.NewError(task, err)
	}

	// Check the states.
	var details bytes.Buffer
	tw := tabwriter.NewWriter(&details, 0, 8, 4, '\t', 0)
	io.WriteString(tw, "\n")
	io.WriteString(tw, "Story URL\tError\n")
	io.WriteString(tw, "=========\t=====\n")

	skipLabels := release.tracker.config.SkipCheckLabels()
	shouldBeSkipped := func(story *pivotal.Story) bool {
		for _, skipLabel := range skipLabels {
			if labeled(story, skipLabel) {
				return true
			}
		}
		return false
	}

	for _, story := range stories {
		// Skip the story in case it is labeled with a skip label.
		if shouldBeSkipped(story) {
			continue
		}

		// Rejected stories are no good.
		if story.State == pivotal.StoryStateRejected {
			fmt.Fprintf(tw, "%v\tstory state: %v\n", story.URL, story.State)
			err = common.ErrNotStageable
			continue
		}

		// Stories that are delivered and further are otherwise OK.
		if stateAtLeast(story, pivotal.StoryStateDelivered) {
			continue
		}

		// Unfinished stories are no good.
		if story.State != pivotal.StoryStateFinished {
			fmt.Fprintf(tw, "%v\tstory state: %v\n", story.URL, story.State)
			err = common.ErrNotStageable
			continue
		}

		// Finished stories need to be checked for the review and QA labels.
		if !release.tracker.canStoryBeStaged(story) {
			fmt.Fprintf(tw, "%v\t%v\n", story.URL, "story not reviewed and tested yet")
			err = common.ErrNotStageable
			continue
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
	stageTask := fmt.Sprintf(
		"Mark the stories as %v in Pivotal Tracker", pivotal.StoryStateDelivered)
	log.Run(stageTask)

	// Load the assigned stories.
	stories, err := release.loadStories()
	if err != nil {
		return nil, errs.NewError(stageTask, err)
	}

	// Pick only the stories that are in the right state.
	ss := make([]*pivotal.Story, 0, len(stories))
	for _, s := range stories {
		if release.tracker.canStoryBeStaged(s) {
			ss = append(ss, s)
		}
	}
	stories = ss

	// Mark the selected stories as delivered. Leave the labels as they are.
	updateRequest := &pivotal.StoryRequest{State: pivotal.StoryStateDelivered}
	updateFunc := func(story *pivotal.Story) *pivotal.StoryRequest {
		return updateRequest
	}
	// On rollback, set the story state to finished again.
	rollbackFunc := func(story *pivotal.Story) *pivotal.StoryRequest {
		return &pivotal.StoryRequest{State: pivotal.StoryStateFinished}
	}

	// Update the stories.
	updatedStories, err := release.tracker.updateStories(stories, updateFunc, rollbackFunc)
	if err != nil {
		return nil, errs.NewError(stageTask, err)
	}
	release.stories = updatedStories

	// Return the rollback function.
	return action.ActionFunc(func() error {
		// On error, set the states back to the original ones.
		log.Rollback(stageTask)
		task := fmt.Sprintf("Reset the story states back to %v", pivotal.StoryStateFinished)
		updatedStories, err := release.tracker.updateStories(release.stories, rollbackFunc, nil)
		if err != nil {
			return errs.NewError(task, err)
		}
		release.stories = updatedStories
		return nil
	}), nil
}

func (release *runningRelease) EnsureReleasable() error {
	versionString := release.version.BaseString()

	task := fmt.Sprintf(
		"Make sure that the stories associated with release %v can be released", versionString)
	log.Run(task)

	// Make sure the stories are loaded.
	stories, err := release.loadStories()
	if err != nil {
		return errs.NewError(task, err)
	}

	// Make sure all relevant stories are accepted.
	// This includes the stories with SkipCheckLabels.
	notAccepted := make([]*pivotal.Story, 0, len(stories))
	for _, story := range stories {
		if story.State != pivotal.StoryStateAccepted {
			notAccepted = append(notAccepted, story)
		}
	}

	// In case there is no story that is not accepted, we are done.
	if len(notAccepted) == 0 {
		return nil
	}

	// Generate the error hint.
	var hint bytes.Buffer
	tw := tabwriter.NewWriter(&hint, 0, 8, 2, '\t', 0)
	fmt.Fprintf(tw, "\nThe following stories cannot be released:\n\n")
	fmt.Fprintf(tw, "Story URL\tState\n")
	fmt.Fprintf(tw, "=========\t=====\n")
	for _, story := range notAccepted {
		fmt.Fprintf(tw, "%v\t%v\n", story.URL, story.State)
	}
	fmt.Fprintf(tw, "\n")
	tw.Flush()

	return errs.NewErrorWithHint(task, common.ErrNotReleasable, hint.String())
}

func (release *runningRelease) Release() error {
	// There is no release step in the Pivotal Tracker really.
	// All the stories are accepted, nothing to be done here.
	return nil
}

func (release *runningRelease) loadStories() ([]*pivotal.Story, error) {
	// Fetch the stories unless cached.
	if release.stories == nil {
		task := "Fetch data from Pivotal Tracker"
		log.Run(task)
		stories, err := release.tracker.storiesByRelease(release.version)
		if err != nil {
			return nil, errs.NewError(task, err)
		}
		release.stories = stories
	}

	// Return the cached stories.
	return release.stories, nil
}
