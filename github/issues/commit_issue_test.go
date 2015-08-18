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
	commitSHA   = "1234567"
	commitTitle = "Update README"
)

// Setup helpers ---------------------------------------------------------------

func newGitHubCommitReviewIssue(body string) *github.Issue {
	title := fmt.Sprintf("Review commit %v: %v", commitSHA, commitTitle)
	return &github.Issue{
		Title: &title,
		Body:  &body,
	}
}

func newCommitReviewIssue(
	commits *CommitList,
	reviewBlockers *ReviewBlockerList,
	userContent string,
) *CommitReviewIssue {

	if commits == nil {
		commits = &CommitList{}
	}
	if reviewBlockers == nil {
		reviewBlockers = &ReviewBlockerList{}
	}

	return &CommitReviewIssue{
		CommitSHA:   commitSHA,
		CommitTitle: commitTitle,
		ReviewIssueCommonBody: &ReviewIssueCommonBody{
			CommitList:        commits,
			ReviewBlockerList: reviewBlockers,
			UserContent:       userContent,
		},
	}
}

// Tests -----------------------------------------------------------------------

var _ = Describe("parsing a commit review issue", func() {

	assertMatch := func(
		issueBodyLines []string,
		expectedCommits *CommitList,
		expectedReviewBlockers *ReviewBlockerList,
		expectedUserContent string,
	) {
		// Set up the input, which is a GitHub issue.
		issueBody := strings.Join(issueBodyLines, "\n")
		githubIssue := newGitHubCommitReviewIssue(issueBody)

		// Set up the expected CommitReviewIssue object.
		expectedReviewIssue := newCommitReviewIssue(
			expectedCommits, expectedReviewBlockers, expectedUserContent)

		// Try to parse the input and make sure it succeeded.
		It("should yield corresponding CommitReviewIssue instance", func() {
			reviewIssue, err := ParseReviewIssue(githubIssue)
			Expect(reviewIssue).To(Equal(expectedReviewIssue))
			Expect(err).To(BeNil())
		})
	}

	assertParsingFailure := func(
		issueBodyLines []string,
	) {
		// Set up the input, which is a GitHub issue.
		issueTitle := fmt.Sprintf("Review commit %v: %v", commitSHA, commitTitle)
		issueBody := strings.Join(issueBodyLines, "\n")
		githubIssue := &github.Issue{
			Title: &issueTitle,
			Body:  &issueBody,
		}

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
			reviewBlockerListString,
			emptyLine,
			userContentSeparator,
			userContentSection,
		}

		assertParsingFailure(issueBodyLines)
	})
})

var _ = Describe("formatting a commit review issue", func() {

	assertBodyMatches := func(
		commits *CommitList,
		reviewBlockers *ReviewBlockerList,
		userContent string,
		expectedBodyLines []string,
	) {

		// Set up the input, which is a CommitReviewIssue object.
		reviewIssue := newCommitReviewIssue(commits, reviewBlockers, userContent)

		// Generate expected review issue body string.
		expectedBody := strings.Join(expectedBodyLines, "\n")

		// Format the body and try to match against the expected string.
		It("should return the expected GitHub issue body", func() {
			Expect(reviewIssue.FormatBody()).To(Equal(expectedBody))
		})
	}

	// Tests, at last!
	It("should return the expected GitHub issue title", func() {
		reviewIssue := newCommitReviewIssue(nil, nil, "")

		expectedTitle := fmt.Sprintf("Review commit %v: %v", commitSHA, commitTitle)
		Expect(reviewIssue.FormatTitle()).To(Equal(expectedTitle))
	})

	Context("containing the commit list only", func() {

		expectedBodyLines := []string{
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
