package pivotaltracker

import (
	// Stdlib
	"regexp"

	// Internal
	"github.com/salsaflow/salsaflow/modules/common"

	// Other
	"github.com/salsita/go-pivotaltracker/v5/pivotal"
)

func toCommonStories(stories []*pivotal.Story, config Config) []common.Story {
	commonStories := make([]common.Story, len(stories))
	for i := range stories {
		commonStories[i] = &story{stories[i], config}
	}
	return commonStories
}

// storiesMatchingByLabel returns the stories for which at least one of the labels matches regexp.
func storiesMatchingByLabel(stories []*pivotal.Story, filter *regexp.Regexp) []*pivotal.Story {
	if filter == nil {
		return stories
	}

	var ss []*pivotal.Story
StoryLoop:
	for _, story := range stories {
		for _, label := range story.Labels {
			if filter.MatchString(label.Name) {
				ss = append(ss, story)
				continue StoryLoop
			}
		}
	}

	return ss
}

func labeled(story *pivotal.Story, label string) bool {
	for _, lab := range story.Labels {
		if lab.Name == label {
			return true
		}
	}
	return false
}

func stateAtLeast(story *pivotal.Story, state string) bool {
	return !stateLessThan(story.State, state)
}

func stateLessThan(stateA, stateB string) bool {
	indexA, indexB := stateToIndex(stateA), stateToIndex(stateB)
	return indexA < indexB
}

var stateIndexes = map[string]int{
	pivotal.StoryStateUnscheduled: 1,
	pivotal.StoryStatePlanned:     2,
	pivotal.StoryStateUnstarted:   3,
	pivotal.StoryStateStarted:     4,
	pivotal.StoryStateFinished:    5,
	pivotal.StoryStateDelivered:   6,
	pivotal.StoryStateAccepted:    7,
	pivotal.StoryStateRejected:    8,
}

func stateToIndex(state string) int {
	return stateIndexes[state]
}
