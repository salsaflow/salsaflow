package sprintly

import (
	// Internal
	"github.com/salsaflow/salsaflow/modules/common"

	// Other
	"github.com/salsita/go-sprintly/sprintly"
)

func toCommonStories(items []sprintly.Item) []common.Story {
	commonStories := make([]common.Story, len(items))
	for i := range items {
		commonStories[i] = &item{&items[i]}
	}
	return commonStories
}

func tagged(item *sprintly.Item, tag string) bool {
	for _, t := range item.Tags {
		if t == tag {
			return true
		}
	}
	return false
}
