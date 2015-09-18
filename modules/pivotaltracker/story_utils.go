package pivotaltracker

import (
	// Internal
	"github.com/salsaflow/salsaflow/modules/common"

	// Other
	"gopkg.in/salsita/go-pivotaltracker.v1/v5/pivotal"
)

func toCommonStories(stories []*pivotal.Story, tracker *issueTracker) []common.Story {
	commonStories := make([]common.Story, len(stories))
	for i, s := range stories {
		if s != nil {
			commonStories[i] = &story{s, tracker}
		}
	}
	return commonStories
}

func labeled(story *pivotal.Story, label string) bool {
	for _, lab := range story.Labels {
		if lab.Name == label {
			return true
		}
	}
	return false
}

func filterStories(stories []*pivotal.Story, filter func(*pivotal.Story) bool) []*pivotal.Story {
	ss := make([]*pivotal.Story, 0, len(stories))
	for _, story := range stories {
		if filter(story) {
			ss = append(ss, story)
		}
	}
	return ss
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
