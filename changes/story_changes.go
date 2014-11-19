package changes

import (
	// Stdlib
	"regexp"

	// Internal
	"github.com/salsita/salsaflow/git"
	"github.com/salsita/salsaflow/modules/common"
)

type StoryChangeGroup struct {
	StoryId string
	Changes []*Change
}

// StoryChanges returns the list of changes grouped by Story-Id.
//
// The list can be filtered using include/exclude expressions.
// The steps to decide whether to include or exclude a change are:
//
//     1) Take all commits matching the include filter.
//        (empty include filter = take all)
//     2) Drop all commits matching the exclude filter.
//        (empty exclude filter = drop none)
//     3) Include the change iff there are any commits left.
func StoryChanges(
	stories []common.Story,
	includeSources, excludeSources []*regexp.Regexp,
) ([]*StoryChangeGroup, error) {

	var groups []*StoryChangeGroup

	for _, story := range stories {
		id := story.ReadableId()

		// Get the relevant commits.
		commits, err := git.ListStoryCommits(id)
		if err != nil {
			return nil, err
		}
		if len(commits) == 0 {
			continue
		}

		// Split by Change-Id.
		chs := GroupCommitsByChangeId(commits)

		// Drop the changes being on the release branch already.
		chs = FilterChangesBySource(chs, includeSources, excludeSources)

		// In case there are no changes left, we are done.
		if len(chs) == 0 {
			continue
		}

		groups = append(groups, &StoryChangeGroup{
			StoryId: id,
			Changes: chs,
		})
	}

	return groups, nil
}
