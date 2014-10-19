package pivotaltracker

import (
	// Stdlib
	"fmt"
	"strconv"

	// Internal
	"github.com/salsita/salsaflow/log"
	"github.com/salsita/salsaflow/modules/common"
	"github.com/salsita/salsaflow/version"

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

func (tracker *issueTracker) SelectActiveStoryIds(ids []string) (activeIds []string, err error) {
	return selectActiveIds(ids)
}

func (tracker *issueTracker) NextRelease(ver *version.Version) (common.NextRelease, error) {
	return newNextRelease(ver)
}

func (tracker *issueTracker) RunningRelease(ver *version.Version) (common.RunningRelease, error) {
	return newRunningRelease(ver)
}

func selectActiveIds(ids []string) (activeIds []string, err error) {
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

func (tracker *issueTracker) StartableStories() (stories []common.Story, err error) {
	pivotalStories, err := listStories("(type:bug OR type:feature) AND (state:unstarted OR state:started)")
	if err != nil {
		return nil, err
	}

	return toCommonStories(pivotalStories), nil
}

func (tracker *issueTracker) StoriesInProgress() (stories []common.Story, err error) {
	pivotalStories, err := listStories("(type:bug OR type:feature) AND (state:started OR state:finished)")
	if err != nil {
		return nil, err
	}

	return toCommonStories(pivotalStories), nil
}

func toCommonStories(stories []*pivotal.Story) []common.Story {
	commonStories := make([]common.Story, len(stories))
	for i := range stories {
		commonStories[i] = &story{stories[i]}
	}
	return commonStories
}
