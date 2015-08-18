package issues_test

import (
	// issues package
	. "github.com/salsaflow/salsaflow/github/issues"

	// Stdlib
	"fmt"
	"strings"

	// Vendor
	"github.com/google/go-github/github"
)

// Data ------------------------------------------------------------------------

var (
	storyId          = "12345"
	storyURL         = "https://issuetracker/story/12345"
	storySummary     = "As a user I want foobar to do barfoo so that I can barbar"
	storyTrackerName = "Issue Tracker"
	storyKey         = storyId
)

var (
	storyLinkSection     = fmt.Sprintf("Story being reviewed: [%s](%s)", storyId, storyURL)
	storyMetadataSection = fmt.Sprintf(
		"SF-Issue-Tracker: %v\nSF-Story-Key: %v", storyTrackerName, storyKey)
)

// Setup helpers ---------------------------------------------------------------

func newGitHubStoryReviewIssue(body string) *github.Issue {
	title := fmt.Sprintf("Review story %s: %s", storyId, storySummary)
	return &github.Issue{
		Title: &title,
		Body:  &body,
	}
}

func newStoryReviewIssue(
	commits *CommitList,
	reviewBlockers *ReviewBlockerList,
	userContent string,
) *StoryReviewIssue {

	if commits == nil {
		commits = &CommitList{}
	}
	if reviewBlockers == nil {
		reviewBlockers = &ReviewBlockerList{}
	}

	return &StoryReviewIssue{
		StoryId:      storyId,
		StoryURL:     storyURL,
		StorySummary: storySummary,
		TrackerName:  storyTrackerName,
		StoryKey:     storyKey,
		ReviewIssueCommonBody: &ReviewIssueCommonBody{
			CommitList:        commits,
			ReviewBlockerList: reviewBlockers,
			UserContent:       userContent,
		},
	}
}

// Tests -----------------------------------------------------------------------

var _ = Describe("parsing a story review issue", func() {

	assertMatch := func(
		issueBodyLines []string,
		expectedCommits *CommitList,
		expectedReviewBlockers *ReviewBlockerList,
		expectedUserContent string,
	) {
		// Set up the input, which is a GitHub issue.
		issueBody := strings.Join(issueBodyLines, "\n")
		githubIssue := newGitHubStoryReviewIssue(issueBody)

		// Set up the expected CommitReviewIssue object.
		expectedReviewIssue := newStoryReviewIssue(
			expectedCommits, expectedReviewBlockers, expectedUserContent)

		// Try to parse the input and make sure it succeeded.
		It("should yield corresponding StoryReviewIssue instance", func() {
			reviewIssue, err := ParseReviewIssue(githubIssue)
			Expect(reviewIssue).To(Equal(expectedReviewIssue))
			Expect(err).To(BeNil())
		})
	}

	assertParsingFailure := func(
		issueBodyLines []string,
	) {
		// Set up the input, which is a GitHub issue.
		issueBody := strings.Join(issueBodyLines, "\n")
		githubIssue := newGitHubStoryReviewIssue(issueBody)

		// Try to parse the input and make sure it failed.
		It("should return a parsing error", func() {
			reviewIssue, err := ParseReviewIssue(githubIssue)
			Expect(reviewIssue).To(BeNil())
			Expect(err).ToNot(BeNil())
		})
	}

	// Tests, at last!
	Context("containing the commit list, the review blocker list and some user content", func() {

		issueBodyLines := []string{
			storyLinkSection,
			emptyLine,
			storyMetadataSection,
			emptyLine,
			commitListString,
			emptyLine,
			reviewBlockerListString,
			emptyLine,
			userContentSeparator,
			userContentSection,
		}

		assertMatch(
			issueBodyLines,
			commitList,
			reviewBlockerList,
			userContentSection,
		)
	})

	Context("containing the commit list and the review blocker list", func() {

		issueBodyLines := []string{
			storyLinkSection,
			emptyLine,
			storyMetadataSection,
			emptyLine,
			commitListString,
			emptyLine,
			reviewBlockerListString,
			emptyLine,
		}

		assertMatch(
			issueBodyLines,
			commitList,
			reviewBlockerList,
			"",
		)
	})

	Context("containing the commit list and some user content", func() {

		issueBodyLines := []string{
			storyLinkSection,
			emptyLine,
			storyMetadataSection,
			emptyLine,
			commitListString,
			emptyLine,
			userContentSeparator,
			userContentSection,
		}

		assertMatch(
			issueBodyLines,
			commitList,
			nil,
			userContentSection,
		)
	})

	Context("missing the commit list", func() {

		issueBodyLines := []string{
			storyLinkSection,
			emptyLine,
			storyMetadataSection,
			emptyLine,
			reviewBlockerListString,
			emptyLine,
			userContentSeparator,
			userContentSection,
		}

		assertParsingFailure(issueBodyLines)
	})
})

var _ = Describe("formatting a story review issue", func() {

	assertBodyMatches := func(
		commits *CommitList,
		reviewBlockers *ReviewBlockerList,
		userContent string,
		expectedBodyLines []string,
	) {

		// Set up the input, which is a StoryReviewIssue object.
		reviewIssue := newStoryReviewIssue(commits, reviewBlockers, userContent)

		// Generate expected review issue body string.
		expectedBody := strings.Join(expectedBodyLines, "\n")

		// Format the body and try to match against the expected string.
		It("should return the expected GitHub issue body", func() {
			Expect(reviewIssue.FormatBody()).To(Equal(expectedBody))
		})
	}

	// Tests, at last!
	It("should return the expected GitHub issue title", func() {
		reviewIssue := newStoryReviewIssue(nil, nil, "")

		expectedTitle := fmt.Sprintf("Review story %v: %v", storyId, storySummary)
		Expect(reviewIssue.FormatTitle()).To(Equal(expectedTitle))
	})

	Context("containing the commit list only", func() {

		expectedBodyLines := []string{
			storyLinkSection,
			emptyLine,
			storyMetadataSection,
			emptyLine,
			commitListString,
			emptyLine,
			emptyLine,
			userContentSeparator,
			defaultUserContentString,
		}

		assertBodyMatches(
			commitList,
			nil,
			"",
			expectedBodyLines,
		)
	})

	Context("containing the commit list and the review blocker list", func() {

		expectedBodyLines := []string{
			storyLinkSection,
			emptyLine,
			storyMetadataSection,
			emptyLine,
			commitListString,
			emptyLine,
			reviewBlockerListString,
			emptyLine,
			userContentSeparator,
			defaultUserContentString,
		}

		assertBodyMatches(
			commitList,
			reviewBlockerList,
			"",
			expectedBodyLines,
		)
	})

	Context("containing the commit list and some user content", func() {

		expectedBodyLines := []string{
			storyLinkSection,
			emptyLine,
			storyMetadataSection,
			emptyLine,
			commitListString,
			emptyLine,
			emptyLine,
			userContentSeparator,
			userContentSection,
		}

		assertBodyMatches(
			commitList,
			nil,
			userContentSection,
			expectedBodyLines,
		)
	})

	Context("containing the commit list, the review blocker list and some user content", func() {

		expectedBodyLines := []string{
			storyLinkSection,
			emptyLine,
			storyMetadataSection,
			emptyLine,
			commitListString,
			emptyLine,
			reviewBlockerListString,
			emptyLine,
			userContentSeparator,
			userContentSection,
		}

		assertBodyMatches(
			commitList,
			reviewBlockerList,
			userContentSection,
			expectedBodyLines,
		)
	})
})
