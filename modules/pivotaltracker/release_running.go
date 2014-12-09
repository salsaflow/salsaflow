package pivotaltracker

import (
	// Internal
	//"github.com/salsaflow/salsaflow/errs"
	//"github.com/salsaflow/salsaflow/log"
	"github.com/salsaflow/salsaflow/action"
	"github.com/salsaflow/salsaflow/modules/common"
	"github.com/salsaflow/salsaflow/version"

	// Other
	//"gopkg.in/salsita/go-pivotaltracker.v0/v5/pivotal"
)

type runningRelease struct{}

func newRunningRelease(releaseVersion *version.Version) (*runningRelease, error) {
	panic("Not implemented")

	/*
		stories, err := listStoriesByRelease(ver)
		if err != nil {
			return nil, err
		}
		return &runningRelease{stories}, nil
	*/
}

func (release *runningRelease) Version() *version.Version {
	panic("Not implemented")
}

func (release *runningRelease) Stories() ([]common.Story, error) {
	panic("Not implemented")

	/*
		var stories []common.Story
		for _, s := range release.stories {
			stories = append(stories, &story{s})
		}
		return stories, nil
	*/
}

func (release *runningRelease) EnsureStageable() error {
	panic("Not implemented")

	/*
		// Make sure that all stories are reviewed and QA'd.
		task := "Make sure that the relevant stories are deliverable"
		log.Run(task)
		ok, details := releaseDeliverable(release.stories)
		if !ok {
			log.FailWithDetails(task, details)
			return ErrReleaseNotDeliverable
		}
		return nil
	*/
}

func (release *runningRelease) Stage() (action.Action, error) {
	panic("Not implemented")

	/*
		// Deliver the stories in Pivotal Tracker.
		task := "Deliver the stories"
		log.Run(task)
		stories, stderr, err := setStoriesState(release.stories, pivotal.StoryStateDelivered)
		if err != nil {
			errs.NewError(task, err, stderr).Log(log.V(log.Info))
			return nil, err
		}
		release.stories = stories
		return common.ActionFunc(func() error {
			// On error, set the story state back to Finished.
			stories, stderr, err := setStoriesState(release.stories, pivotal.StoryStateFinished)
			if err != nil {
				errs.NewError(task, err, stderr).Log(log.V(log.Info))
				return err
			}
			release.stories = stories
			return nil
		}), nil
	*/
}

func (release *runningRelease) Releasable() (bool, error) {
	panic("Not implemented")
}

func (release *runningRelease) Release() error {
	panic("Not implemented")
}
