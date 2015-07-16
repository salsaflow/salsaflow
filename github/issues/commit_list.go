package issues

// CommitItem represents a line in the commit checklist.
type CommitItem struct {
	CommitSHA   string
	CommitTitle string
	Done        bool
}

// CommitList is a placeholder for multiple commit items.
type CommitList struct {
	items []*CommitItem
}

func (list *CommitList) CommitItems() []*CommitItem {
	return list.items
}

// AddCommit adds the commit to the list unless the commit is already there.
func (list *CommitList) AddCommit(commitSHA, commitTitle string, done bool) bool {
	for _, item := range list.items {
		if item.CommitSHA == commitSHA {
			return false
		}
	}

	list.items = append(list.items, &CommitItem{
		CommitSHA:   commitSHA,
		CommitTitle: commitTitle,
		Done:        done,
	})
	return true
}
