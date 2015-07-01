package releases

import (
	"github.com/salsaflow/salsaflow/changes"
	"github.com/salsaflow/salsaflow/git"
)

func StoryChangesToCherryPick(
	groups []*changes.StoryChangeGroup,
) ([]*changes.StoryChangeGroup, error) {

	// Get the changes that are reachable from the release branch.
	gitConfig, err := git.LoadConfig()
	if err != nil {
		return nil, err
	}
	releaseBranch := gitConfig.ReleaseBranchName()

	reachableCommits, err := git.ShowCommitRange(releaseBranch)
	if err != nil {
		return nil, err
	}

	reachableChanges := make(map[string]struct{}, len(reachableCommits))
	for _, commit := range reachableCommits {
		// Would not probably harm much not to have the condition here,
		// but hey, let's keep the Change-Id set clean.
		if commit.ChangeIdTag != "" {
			reachableChanges[commit.ChangeIdTag] = struct{}{}
		}
	}

	// Get the changes that needs to be cherry-picked.
	var toCherryPick []*changes.StoryChangeGroup

	for _, group := range groups {
		// Prepare a new StoryChangeGroup to hold missing changes.
		storyGroup := &changes.StoryChangeGroup{
			StoryIdTag: group.StoryIdTag,
		}

		// A change needs cherry-picking in case it's not reachable
		// from the release branch, right?
		for _, change := range group.Changes {
			// Skip the group representing commits with no Change-Id tag.
			if change.ChangeIdTag == "" {
				continue
			}
			// Append the group in case the Change-Id is not reachable.
			if _, ok := reachableChanges[change.ChangeIdTag]; !ok {
				storyGroup.Changes = append(storyGroup.Changes, change)
			}
		}

		// Append the whole story group in case any associated change group
		// is not reachable from the release branch.
		if len(storyGroup.Changes) != 0 {
			toCherryPick = append(toCherryPick, storyGroup)
		}
	}

	return toCherryPick, nil
}
