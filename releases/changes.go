package releases

import (
	"github.com/salsaflow/salsaflow/changes"
	"github.com/salsaflow/salsaflow/git"
)

func StoryChangesToCherryPick(
	groups []*changes.StoryChangeGroup,
) ([]*changes.StoryChangeGroup, error) {

	// Get the commits that are reachable from the release branch.
	gitConfig, err := git.LoadConfig()
	if err != nil {
		return nil, err
	}
	releaseBranch := gitConfig.ReleaseBranchName()

	reachableCommits, err := git.ShowCommitRange(releaseBranch)
	if err != nil {
		return nil, err
	}

	reachableCommitMap := make(map[string]struct{}, len(reachableCommits))
	for _, commit := range reachableCommits {
		reachableCommitMap[commit.SHA] = struct{}{}
	}

	// Get the changes that needs to be cherry-picked.
	var toCherryPick []*changes.StoryChangeGroup

	for _, group := range groups {
		// Prepare a new StoryChangeGroup to hold missing changes.
		storyGroup := &changes.StoryChangeGroup{
			StoryIdTag: group.StoryIdTag,
		}

	ChangesLoop:
		// Loop over the story changes and the commits associated with
		// these changes. A change needs cherry-picking in case there are
		// some commits left when we drop the commits reachable from
		// the release branch.
		for _, change := range group.Changes {
			for _, commit := range change.Commits {
				if _, ok := reachableCommitMap[commit.SHA]; ok {
					continue ChangesLoop
				}
			}

			storyGroup.Changes = append(storyGroup.Changes, change)
		}

		if len(storyGroup.Changes) != 0 {
			toCherryPick = append(toCherryPick, storyGroup)
		}
	}

	return toCherryPick, nil
}
