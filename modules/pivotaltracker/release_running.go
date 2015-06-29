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
	"github.com/salsita/go-pivotaltracker/v5/pivotal"
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

	var details bytes.Buffer
	tw := tabwriter.NewWriter(&details, 0, 8, 4, '\t', 0)
	io.WriteString(tw, "\n")
	io.WriteString(tw, "Story URL\tError\n")
	io.WriteString(tw, "=========\t=====\n")

	// For a story to be stageable, it must be in the Finished stage.
	// That by definition means that it has been reviewed and verified.
	for _, story := range stories {
		if !stateAtLeast(story, pivotal.StoryStateFinished) {
			fmt.Fprintf(tw, "%v\t%v\n", story.URL, "story not finished yet")
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
	stageTask := "Mark the stories as delivered in Pivotal Tracker"
	log.Run(stageTask)

	// Load the assigned stories.
	stories, err := release.loadStories()
	if err != nil {
		return nil, errs.NewError(stageTask, err)
	}

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
		config    = release.tracker.config
		client    = pivotal.NewClient(config.UserToken())
		projectId = config.ProjectId()
	)
	updatedStories, err := updateStories(client, projectId, stories, updateFunc, rollbackFunc)
	if err != nil {
		return nil, errs.NewError(stageTask, err)
	}
	release.stories = updatedStories

	// Return the rollback function.
	return action.ActionFunc(func() error {
		// On error, set the states back to the original ones.
		log.Rollback(stageTask)
		task := "Reset the story states back to the original ones"
		updatedStories, err := updateStories(client, projectId, release.stories, rollbackFunc, nil)
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
		var (
			config       = release.tracker.config
			client       = pivotal.NewClient(config.UserToken())
			projectId    = config.ProjectId()
			releaseLabel = getReleaseLabel(release.version)
		)
		stories, err := searchStories(client, projectId, "label:%v", releaseLabel)
		if err != nil {
			return nil, errs.NewError(task, err)
		}
		release.stories = stories
	}

	// Return the cached stories.
	return release.stories, nil
}
