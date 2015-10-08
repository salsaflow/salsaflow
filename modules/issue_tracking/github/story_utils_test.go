package github

import (
	// Stdlib
	"fmt"

	// Internal
	"github.com/salsaflow/salsaflow/modules/common"

	// Vendor
	"github.com/google/go-github/github"
)

var _ = Describe("getting the abstract story state for given GitHub issue", func() {

	config := &moduleConfig{
		ApprovedLabel:         "approved",
		BeingImplementedLabel: "being implemented",
		ImplementedLabel:      "implemented",
		ReviewedLabel:         "reviewed",
		SkipReviewLabel:       "no review",
		PassedTestingLabel:    "qa+",
		FailedTestingLabel:    "qa-",
		SkipTestingLabel:      "no qa",
		StagedLabel:           "staged",
		RejectedLabel:         "rejected",
	}

	data := []struct {
		issueState            string
		issueLabelNames       []string
		expectedAbstractState common.StoryState
	}{
		{
			"open",
			[]string{"approved"},
			common.StoryStateApproved,
		},
		{
			"open",
			[]string{"being implemented"},
			common.StoryStateBeingImplemented,
		},
		{
			"open",
			[]string{"implemented"},
			common.StoryStateImplemented,
		},
		{
			"open",
			[]string{"reviewed"},
			common.StoryStateReviewed,
		},
		{
			"open",
			[]string{"reviewed", "qa-"},
			common.StoryStateReviewed,
		},
		{
			"open",
			[]string{"reviewed", "qa+"},
			common.StoryStateTested,
		},
		{
			"open",
			[]string{"reviewed", "no qa"},
			common.StoryStateTested,
		},
		{
			"open",
			[]string{"no review", "no qa"},
			common.StoryStateTested,
		},
		{
			"open",
			[]string{"staged"},
			common.StoryStateStaged,
		},
		{
			"open",
			[]string{"rejected"},
			common.StoryStateRejected,
		},
		{
			"closed",
			nil,
			common.StoryStateAccepted,
		},
	}

	toLabelObjects := func(labelNames []string) []github.Label {
		labels := make([]github.Label, 0, len(labelNames))
		for _, name := range labelNames {
			labels = append(labels, github.Label{
				Name: github.String(name),
			})
		}
		return labels
	}

	for i := range data {
		func(i int) {

			td := data[i]

			Context(fmt.Sprintf("where %+v", td), func() {

				issue := &github.Issue{
					State:  &td.issueState,
					Labels: toLabelObjects(td.issueLabelNames),
				}

				It("should return expected abstract story state", func() {

					Expect(abstractState(issue, config)).To(Equal(td.expectedAbstractState))
				})
			})
		}(i)
	}

})
