package issues_test

import (
	. "github.com/salsaflow/salsaflow/github/issues"
)

var _ = Describe("ReviewBlockerList", func() {
	var (
		fixed             = false
		commentURL        = "https://github.com/someurl"
		anotherCommentURL = "https://github.com/someotherurl"
		commitSHA         = "12345"
		blockerSummary    = "Please fix this and that"
	)

	list := &ReviewBlockerList{}
	add := func(fixed bool, commentURL, commitSHA, blockerSummar string) bool {
		return list.AddReviewBlocker(fixed, commentURL, commitSHA, blockerSummary)
	}
	length := func() int {
		return len(list.ReviewBlockerItems())
	}

	It("should not accept duplicate comment URLs", func() {
		Expect(length()).To(Equal(0))

		added := add(fixed, commentURL, commitSHA, blockerSummary)
		Expect(added).To(Equal(true))
		Expect(length()).To(Equal(1))

		added = add(fixed, commentURL, commitSHA, blockerSummary)
		Expect(added).To(Equal(false))
		Expect(length()).To(Equal(1))

		added = add(fixed, anotherCommentURL, commitSHA, blockerSummary)
		Expect(added).To(Equal(true))
		Expect(length()).To(Equal(2))
	})
})
