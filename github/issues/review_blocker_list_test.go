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

	var list *ReviewBlockerList

	BeforeEach(func() {
		list = &ReviewBlockerList{}
	})

	It("should not accept duplicate comment URLs", func() {
		Expect(len(list.ReviewBlockerItems())).To(Equal(0))

		added := list.AddReviewBlocker(fixed, commentURL, commitSHA, blockerSummary)
		Expect(added).To(BeTrue())
		Expect(len(list.ReviewBlockerItems())).To(Equal(1))

		added = list.AddReviewBlocker(fixed, commentURL, commitSHA, blockerSummary)
		Expect(added).ToNot(BeTrue())
		Expect(len(list.ReviewBlockerItems())).To(Equal(1))

		added = list.AddReviewBlocker(fixed, anotherCommentURL, commitSHA, blockerSummary)
		Expect(added).To(BeTrue())
		Expect(len(list.ReviewBlockerItems())).To(Equal(2))
	})
})
