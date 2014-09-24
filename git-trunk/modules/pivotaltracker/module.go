package pivotaltracker

import (
	// Stdlib
	"fmt"
	"strconv"

	// Internal
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

func (tracker *issueTracker) CurrentUser() (common.User, error) {
	me, err := fetchMe()
	if err != nil {
		return nil, err
	}
	return &user{me}, nil
}

func (tracker *issueTracker) ActiveStoryIds(ids []string) (activeIds []string, err error) {
	return onlyActiveIds(ids)
}

func (tracker *issueTracker) NextRelease(ver *version.Version) (common.NextRelease, error) {
	return newNextRelease(ver)
}

func (tracker *issueTracker) RunningRelease(ver *version.Version) (common.RunningRelease, error) {
	return newRunningRelease(ver)
}

func onlyActiveIds(ids []string) (activeIds []string, err error) {
	info := log.V(log.Info)

	// Fetch the relevant stories
	msg := "Fetch the relevant stories"
	info.Run(msg)

	stories, err := listStoriesById(ids)
	if err != nil {
		return nil, err
	}

	storyMap := make(map[string]*pivotal.Story)
	for _, story := range stories {
		storyMap[strconv.Itoa(story.Id)] = story
	}

	// Filter the stories according to the story state.
	msg = "Filter the stories according to the story state"
	var active []string
	for _, id := range ids {
		story, ok := storyMap[id]
		if !ok {
			info.Fail(msg)
			err = fmt.Errorf("story with id %v not found", id)
			return nil, err
		}

		switch story.State {
		case pivotal.StoryStateFinished:
		case pivotal.StoryStateDelivered:
		case pivotal.StoryStateAccepted:
		default:
			active = append(active, id)
		}
	}

	return active, nil
}

func (tracker *issueTracker) GetStartableStories() (stories []common.Story, err error) {
	pivotalStories, err := listStories("(type:bug OR type:feature) AND (state:unstarted OR state:started)")
	if err != nil {
		return nil, err
	}
	// Cast to `[]pivotaltracker.story`.
	ptStories := make([]story, len(pivotalStories))
	for i := range pivotalStories {
		ptStories[i] = story{pivotalStories[i]}
	}
	// Cast to `[]common.Story`.
	commonStories := make([]common.Story, len(ptStories))
	for i := range ptStories {
		commonStories[i] = &ptStories[i]
	}
	return commonStories, nil
}
