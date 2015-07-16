package issues

// ReviewBlockerItem represents a line in the review blocker checkbox.
type ReviewBlockerItem struct {
	CommitSHA      string
	CommentURL     string
	BlockerNumber  int
	BlockerSummary string
	Fixed          bool
}

// ReviewBlockerList is a placeholder for multiple review blockers.
type ReviewBlockerList struct {
	items []*ReviewBlockerItem
}

func (list *ReviewBlockerList) ReviewBlockerItems() []*ReviewBlockerItem {
	return list.items
}

// AddReviewBlocker adds the blocker to the list unless the blocker is already there.
// The field that is check and must be unique is the comment URL.
func (list *ReviewBlockerList) AddReviewBlocker(
	commitSHA string,
	commentURL string,
	summary string,
	fixed bool,
) bool {

	for _, item := range list.items {
		if item.CommentURL == commentURL {
			return false
		}
	}

	list.items = append(list.items, &ReviewBlockerItem{
		CommitSHA:      commitSHA,
		CommentURL:     commentURL,
		BlockerNumber:  len(list.items) + 1,
		BlockerSummary: summary,
		Fixed:          fixed,
	})
	return true
}
