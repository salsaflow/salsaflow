package changes

type StoryChangeGroup struct {
	StoryId int
	Changes []*Change
}

func GroupChangesByStoryId(changes []*Change) []*StoryChangeGroup {
	groups := make([]*StoryChangeGroup, 0)

ChangesLoop:
	for _, change := range changes {
		// Try to add the change to one of the existing groups.
		for _, group := range groups {
			if group.StoryId == change.StoryId {
				group.Changes = append(group.Changes, change)
				continue ChangesLoop
			}
		}
		// Otherwise create a new group and append it.
		groups = append(groups, &StoryChangeGroup{
			StoryId: change.StoryId,
			Changes: []*Change{change},
		})
	}

	return groups
}
