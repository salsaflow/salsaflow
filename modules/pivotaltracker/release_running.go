package pivotaltracker

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

	// Other
	"github.com/salsita/go-pivotaltracker/v5/pivotal"
)

type runningRelease struct {
	version *version.Version
	stories []*pivotal.Story
	config  Config
}

func newRunningRelease(releaseVersion *version.Version, config Config) (*runningRelease, error) {
	return &runningRelease{
		version: releaseVersion,
		config:  config,
	}, nil
}

func (release *runningRelease) Version() *version.Version {
	return release.version
}

func (release *runningRelease) Stories() ([]common.Story, error) {
	// Fetch the stories unless cached.
	if release.stories == nil {
		task := "Fetch data from Pivotal Tracker"
		log.Run(task)
		var (
			client       = pivotal.NewClient(release.config.UserToken())
			projectId    = release.config.ProjectId()
			releaseLabel = getReleaseLabel(release.version)
		)
		stories, err := searchStories(client, projectId, "label:%v", releaseLabel)
		if err != nil {
			return nil, errs.NewError(task, err, nil)
		}
		release.stories = stories
	}

	// Return the cached stories.
	return toCommonStories(release.stories, release.config), nil
}

func (release *runningRelease) EnsureStageable() error {
	task := "Make sure the stories can be staged"
	log.Run(task)

	// Make sure the stories are fetched.
	if err := release.ensureStoriesLoaded(); err != nil {
		return err
	}
	stories := release.stories

	var details bytes.Buffer
	tw := tabwriter.NewWriter(&details, 0, 8, 4, '\t', 0)
	io.WriteString(tw, "\n")
	io.WriteString(tw, "Story URL\tError\n")
	io.WriteString(tw, "=========\t=====\n")

	// For a story to be stageable, it must be in the Finished stage.
	// That by definition means that it has been reviewed and verified.
	var (
		err             error
		errNotStageable = errors.New("release not stageable")
	)
	for _, story := range stories {
		if !stateAtLeast(story, pivotal.StoryStateFinished) {
			fmt.Fprintf(tw, "%v\t%v\n", story.URL, "story not finished yet")
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
	task := "Mark the stories as Delivered in Pivotal Tracker"
	log.Run(task)

	// Make sure the stories are loaded.
	if err := release.ensureStoriesLoaded(); err != nil {
		return nil, errs.NewError(task, err, nil)
	}
	stories := release.stories

	// Save the original states into a map.
	originalStates := make(map[int]string, len(stories))
	for _, story := range stories {
		originalStates[story.Id] = story.State
	}

	// Set all the states to Delivered.
	updateRequest := &pivotal.Story{State: pivotal.StoryStateDelivered}
	updateFunc := func(story *pivotal.Story) *pivotal.Story {
		return updateRequest
	}
	// On rollback, get the original state from the map.
	rollbackFunc := func(story *pivotal.Story) *pivotal.Story {
		return &pivotal.Story{State: originalStates[story.Id]}
	}

	// Update the stories.
	var (
		client    = pivotal.NewClient(release.config.UserToken())
		projectId = release.config.ProjectId()
	)
	updatedStories, err := updateStories(client, projectId, stories, updateFunc, rollbackFunc)
	if err != nil {
		return nil, errs.NewError(task, err, nil)
	}
	release.stories = updatedStories

	// Return the rollback function.
	return action.ActionFunc(func() error {
		// On error, set the states back to the original ones.
		task := "Reset the story states back to the original ones"
		updatedStories, err := updateStories(client, projectId, release.stories, rollbackFunc, nil)
		if err != nil {
			return errs.NewError(task, err, nil)
		}
		release.stories = updatedStories
		return nil
	}), nil
}

func (release *runningRelease) Releasable() (bool, error) {
	task := "Make sure the stories can be released"
	log.Run(task)

	// Make sure the stories are loaded.
	if err := release.ensureStoriesLoaded(); err != nil {
		return false, errs.NewError(task, err, nil)
	}
	stories := release.stories

	// Make sure all relevant stories are Accepted.
	// This includes the stories with SkipCheckLabels.
	// These should be Accepted as well, maybe by a daemon.
	for _, story := range stories {
		if story.State != pivotal.StoryStateAccepted {
			return false, nil
		}
	}
	return true, nil
}

func (release *runningRelease) Release() error {
	// There is no release step in the Pivotal Tracker really.
	// All the stories are accepted, nothing to be done here.
	return nil
}

func (release *runningRelease) ensureStoriesLoaded() error {
	if _, err := release.Stories(); err != nil {
		return err
	}
	if release.stories == nil {
		panic("bug(stories == nil)")
	}
	return nil
}
