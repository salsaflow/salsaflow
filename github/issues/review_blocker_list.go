package issues

// ReviewBlockerItem represents a line in the review blocker checkbox.
type ReviewBlockerItem struct {
	Fixed          bool
	CommentURL     string
	CommitSHA      string
	BlockerNumber  int
	BlockerSummary string
}

// ReviewBlockerList is a placeholder for multiple review blockers.
type ReviewBlockerList struct {
	items []*ReviewBlockerItem
}

func NewReviewBlockerList(items []*ReviewBlockerItem) *ReviewBlockerList {
	return &ReviewBlockerList{items}
}

func (list *ReviewBlockerList) ReviewBlockerItems() []*ReviewBlockerItem {
	return list.items
}

// AddReviewBlocker adds the blocker to the list unless the blocker is already there.
// The field that is check and must be unique is the comment URL.
func (list *ReviewBlockerList) AddReviewBlocker(
	fixed bool,
	commentURL string,
	commitSHA string,
	blockerSummary string,
) bool {

	for _, item := range list.items {
		if item.CommentURL == commentURL {
			return false
		}
	}

	list.items = append(list.items, &ReviewBlockerItem{
		Fixed:          fixed,
		CommentURL:     commentURL,
		CommitSHA:      commitSHA,
		BlockerNumber:  len(list.items) + 1,
		BlockerSummary: blockerSummary,
	})
	return true
}
