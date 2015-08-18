package issues_test

import (
	. "github.com/salsaflow/salsaflow/github/issues"
)

var _ = Describe("CommitList", func() {
	var (
		reviewed           = false
		commitSHA          = "12345"
		commitTitle        = "Commit title"
		anotherCommitSHA   = "23456"
		anotherCommitTitle = "Another commit title"
	)

	var list *CommitList

	BeforeEach(func() {
		list = &CommitList{}
	})

	Describe("CommitList", func() {
		It("should not accept duplicate commit SHAs", func() {
			Expect(len(list.CommitItems())).To(Equal(0))

			added := list.AddCommit(reviewed, commitSHA, commitTitle)
			Expect(added).To(BeTrue())
			Expect(len(list.CommitItems())).To(Equal(1))

			added = list.AddCommit(reviewed, commitSHA, commitTitle)
			Expect(added).ToNot(BeTrue())
			Expect(len(list.CommitItems())).To(Equal(1))

			added = list.AddCommit(reviewed, anotherCommitSHA, anotherCommitTitle)
			Expect(added).To(BeTrue())
			Expect(len(list.CommitItems())).To(Equal(2))
		})
	})
})
