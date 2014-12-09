package pivotaltracker

import (
	// Stdlib
	"regexp"

	// Internal
	"github.com/salsaflow/salsaflow/modules/common"

	// Other
	"github.com/salsita/go-pivotaltracker/v5/pivotal"
)

func toCommonStories(stories []*pivotal.Story) []common.Story {
	commonStories := make([]common.Story, len(stories))
	for i := range stories {
		commonStories[i] = &story{stories[i]}
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
