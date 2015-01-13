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
