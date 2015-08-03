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

	list := &CommitList{}
	add := func(reviewed bool, commitSHA, commitTitle string) {
		list.AddCommit(reviewed, commitSHA, commitTitle)
	}
	length := func() int {
		return len(list.CommitItems())
	}

	Describe("CommitList", func() {
		It("should not accept duplicate commit SHAs", func() {
			Expect(length()).To(Equal(0))

			add(reviewed, commitSHA, commitTitle)
			Expect(length()).To(Equal(1))

			add(reviewed, commitSHA, commitTitle)
			Expect(length()).To(Equal(1))

			add(reviewed, anotherCommitSHA, anotherCommitTitle)
			Expect(length()).To(Equal(2))
		})
	})
})
