package issues

// CommitItem represents a line in the commit checklist.
type CommitItem struct {
	Reviewed    bool
	CommitSHA   string
	CommitTitle string
}

// CommitList is a placeholder for multiple commit items.
type CommitList struct {
	items []*CommitItem
}

func NewCommitList(items []*CommitItem) *CommitList {
	return &CommitList{items}
}

func (list *CommitList) CommitItems() []*CommitItem {
	return list.items
}

// AddCommit adds the commit to the list unless the commit is already there.
func (list *CommitList) AddCommit(reviewed bool, commitSHA, commitTitle string) bool {
	for _, item := range list.items {
		if item.CommitSHA == commitSHA {
			return false
		}
	}

	list.items = append(list.items, &CommitItem{
		Reviewed:    reviewed,
		CommitSHA:   commitSHA,
		CommitTitle: commitTitle,
	})
	return true
}
