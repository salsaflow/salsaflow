package github

import (
	// Internal
	"github.com/salsaflow/salsaflow/modules/common"

	// Other
	"github.com/google/go-github/github"
)

// toCommonStories turns []*github.Issue into []common.Story
func toCommonStories(issues []*github.Issue, tracker *issueTracker) []common.Story {
	commonStories := make([]common.Story, len(issues))
	for i, issue := range issues {
		if issue != nil {
			commonStories[i] = &story{issue, tracker}
		}
	}
	return commonStories
}

// labeled returns true when the given issue is labeled with the given label.
func labeled(issue *github.Issue, label string) bool {
	for _, l := range issue.Labels {
		if *l.Name == label {
			return true
		}
	}
	return false
}

// pruneStateLabels gets the label slice passed in and splits it
// into the state labels as used by SalsaFlow and the rest.
func pruneStateLabels(
	config *moduleConfig,
	labels []github.Label,
) (remainingLabels, prunedLabels []github.Label) {

	stateLabels := map[string]struct{}{
		config.ApprovedLabel:         struct{}{},
		config.BeingImplementedLabel: struct{}{},
		config.ImplementedLabel:      struct{}{},
		config.ReviewedLabel:         struct{}{},
		config.SkipReviewLabel:       struct{}{},
		config.PassedTestingLabel:    struct{}{},
		config.FailedTestingLabel:    struct{}{},
		config.SkipTestingLabel:      struct{}{},
		config.StagedLabel:           struct{}{},
		config.RejectedLabel:         struct{}{},
	}

	var (
		remaining = make([]github.Label, 0, len(labels))
		pruned    = make([]github.Label, 0, len(labels))
	)
	for _, label := range labels {
		if _, ok := stateLabels[*label.Name]; ok {
			pruned = append(pruned, label)
		} else {
			remaining = append(remaining, label)
		}
	}
	return remaining, pruned
}

// abstractState gets the given GitHub issues and returns the associated abstract state.
func abstractState(issue *github.Issue, config *moduleConfig) common.StoryState {
	if *issue.State == "closed" {
		return common.StoryStateAccepted
	}

	var (
		approvedLabel         = config.ApprovedLabel
		beingImplementedLabel = config.BeingImplementedLabel
		implementedLabel      = config.ImplementedLabel
		reviewedLabel         = config.ReviewedLabel
		skipReviewLabel       = config.SkipReviewLabel
		passedTestingLabel    = config.PassedTestingLabel
		skipTestingLabel      = config.SkipTestingLabel
		stagedLabel           = config.StagedLabel
		rejectedLabel         = config.RejectedLabel
	)

	for _, label := range issue.Labels {
		switch *label.Name {
		case approvedLabel:
			return common.StoryStateApproved

		case beingImplementedLabel:
			return common.StoryStateBeingImplemented

		case implementedLabel:
			return common.StoryStateImplemented

		case reviewedLabel:
			fallthrough
		case skipReviewLabel:
			fallthrough
		case passedTestingLabel:
			fallthrough
		case skipTestingLabel:
			reviewed := labeled(issue, reviewedLabel) || labeled(issue, skipReviewLabel)
			tested := labeled(issue, passedTestingLabel) || labeled(issue, skipTestingLabel)

			switch {
			case reviewed && tested:
				return common.StoryStateTested
			case reviewed:
				return common.StoryStateReviewed
			case tested:
				return common.StoryStateImplemented
			}

		case stagedLabel:
			return common.StoryStateStaged

		case rejectedLabel:
			return common.StoryStateRejected
		}
	}

	return common.StoryStateInvalid
}

// filterIssues is just the regular filter function for *github.Issue type.
func filterIssues(issues []*github.Issue, filter func(*github.Issue) bool) []*github.Issue {
	iss := make([]*github.Issue, 0, len(issues))
	for _, issue := range issues {
		if filter(issue) {
			iss = append(iss, issue)
		}
	}
	return iss
}

// dedupeIssues makes sure the list contains only 1 issue object for every issue number.
func dedupeIssues(issues []*github.Issue) []*github.Issue {
	set := make(map[int]struct{}, len(issues))
	return filterIssues(issues, func(issue *github.Issue) bool {
		num := *issue.Number
		if _, seen := set[num]; seen {
			return false
		}
		set[num] = struct{}{}
		return true
	})
}
