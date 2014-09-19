package pivotaltracker

import (
	// Stdlib
	"fmt"
	"strconv"

	// Internal
	"github.com/salsita/SalsaFlow/git-trunk/errors"
	"github.com/salsita/SalsaFlow/git-trunk/log"
	"github.com/salsita/SalsaFlow/git-trunk/modules/common"
	"github.com/salsita/SalsaFlow/git-trunk/version"

	// Other
	"gopkg.in/salsita/go-pivotaltracker.v0/v5/pivotal"
)

type issueTracker struct{}

func Factory() (common.IssueTracker, error) {
	if err := loadConfig(); err != nil {
		return nil, err
	}
	return &issueTracker{}, nil
}

func (tracker *issueTracker) ListActiveStoryIds(ids []string) (activeIds []string, err error) {
	return listActiveStoryIds(ids)
}

func (tracker *issueTracker) NextRelease(ver *version.Version) (common.NextRelease, error) {
	return newNextRelease(ver)
}

func (tracker *issueTracker) RunningRelease(ver *version.Version) (common.RunningRelease, error) {
	return newRunningRelease(ver)
}

func listActiveStoryIds(ids []string) (activeIds []string, err error) {
	info := log.V(log.Info)
	errCh := make(chan error, 2)

	// Start fetching the relevant stories.
	msg := "Fetch the relevant stories"
	info.Go(msg)

	storyMap := make(map[string]*pivotal.Story)
	go func(taskMsg string) {
		stories, err := listStoriesById(ids)
		if err != nil {
			errors.NewError(taskMsg, nil, err).Log(info)
			errCh <- err
			return
		}

		for _, story := range stories {
			storyMap[strconv.Itoa(story.Id)] = story
		}

		info.Ok(taskMsg)
		errCh <- nil
	}(msg)

	// Start fetching the current user's record.
	msg = "Fetch your Pivotal Tracker user record"
	info.Go(msg)

	var me *pivotal.Me
	go func(taskMsg string) {
		var err error
		me, err = fetchMe()
		if err == nil {
			info.Ok(msg)
		} else {
			info.Fail(msg)
		}
		errCh <- err
	}(msg)

	// Wait for all the goroutines to finish, unless there is an error.
	for i := 0; i < cap(errCh); i++ {
		if err := <-errCh; err != nil {
			return nil, err
		}
	}

	// Filter the ids according to the story state.
	msg = "Filter the branches according to the story state and owner"
	var filteredIds []string
	for _, id := range ids {
		story, ok := storyMap[id]
		if !ok {
			info.Fail(msg)
			err = fmt.Errorf("story with id %v not found", id)
			return
		}

		// Check the story owner.
		for _, ownerId := range story.OwnerIds {
			if ownerId == me.Id {
				goto CheckState
			}
		}
		// No matching owner found, skip the reference.
		continue

		// Check the story state.
	CheckState:
		switch story.State {
		case pivotal.StoryStateFinished:
		case pivotal.StoryStateDelivered:
		case pivotal.StoryStateAccepted:
		default:
			filteredIds = append(filteredIds, id)
		}
	}

	return filteredIds, nil
}
