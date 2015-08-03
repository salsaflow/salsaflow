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

// Tests -----------------------------------------------------------------------

var _ = Describe("parsing a commit review issue", func() {

	// Input lines to be parsed
	var (
		issueBodyLines []string
	)

	// Parts to asseble the expected review issue object
	var (
		expectedCommits        *CommitList
		expectedReviewBlockers *ReviewBlockerList
		expectedUserContent    string
	)

	// Internal variables for assertion closures
	var (
		githubIssue         *github.Issue
		expectedReviewIssue *CommitReviewIssue
	)

	// Common initialisation before every It run sets the internal variables.
	JustBeforeEach(func() {
		issueTitle := fmt.Sprintf("Review commit %v: %v", commitSHA, commitTitle)
		issueBody := strings.Join(issueBodyLines, "\n")
		githubIssue = &github.Issue{
			Title: &issueTitle,
			Body:  &issueBody,
		}

		expectedReviewIssue = &CommitReviewIssue{
			CommitSHA:   commitSHA,
			CommitTitle: commitTitle,
			ReviewIssueCommonBody: &ReviewIssueCommonBody{
				CommitList:        expectedCommits,
				ReviewBlockerList: expectedReviewBlockers,
				UserContent:       expectedUserContent,
			},
		}
	})

	// Assertion closures
	shouldMatch := func() {
		reviewIssue, err := ParseReviewIssue(githubIssue)
		Expect(reviewIssue).To(Equal(expectedReviewIssue))
		Expect(err).To(BeNil())
	}

	shouldFail := func() {
		reviewIssue, err := ParseReviewIssue(githubIssue)
		Expect(reviewIssue).To(BeNil())
		Expect(err).ToNot(BeNil())
	}

	assertMatch := func() {
		It("should yield corresponding CommitReviewIssue instance", shouldMatch)
	}

	assertParsingFailure := func() {
		It("should return a parsing error", shouldFail)
	}

	// Tests, at last!
	Context("containing the commit list, the review blocker list and some user content", func() {

		BeforeEach(func() {
			// Review issue
			issueBodyLines = []string{
				commitListString,
				emptyLine,
				reviewBlockerListString,
				emptyLine,
				userContentSeparator,
				userContentSection,
			}

			// Expected review issue object
			expectedCommits = commitList
			expectedReviewBlockers = reviewBlockerList
			expectedUserContent = userContentSection
		})

		assertMatch()
	})

	Context("containing the commit list and the review blocker list", func() {

		BeforeEach(func() {
			// Review issue
			issueBodyLines = []string{
				commitListString,
				emptyLine,
				reviewBlockerListString,
				emptyLine,
			}

			// Expected review issue object
			expectedCommits = commitList
			expectedReviewBlockers = reviewBlockerList
			expectedUserContent = ""
		})

		assertMatch()
	})

	Context("containing the commit list and some user content", func() {

		BeforeEach(func() {
			// Review issue
			issueBodyLines = []string{
				commitListString,
				emptyLine,
				userContentSeparator,
				userContentSection,
			}

			// Expected review issue object
			expectedCommits = commitList
			expectedReviewBlockers = &ReviewBlockerList{}
			expectedUserContent = userContentSection
		})

		assertMatch()
	})

	Context("missing the commit list", func() {

		BeforeEach(func() {
			// Review issue
			issueBodyLines = []string{
				reviewBlockerListString,
				emptyLine,
				userContentSeparator,
				userContentSection,
			}
		})

		assertParsingFailure()
	})
})

var _ = Describe("formatting a commit review issue", func() {

	// Parts used to construct the review issue
	var (
		commits        *CommitList
		reviewBlockers *ReviewBlockerList
		userContent    string
	)

	// Expected formatting output lines
	var (
		expectedBodyLines []string
	)

	// Internal variables for assertion closured
	var (
		reviewIssue *CommitReviewIssue

		expectedTitle string
		expectedBody  string
	)

	// Common initialisation before every It run sets the internal variables.
	JustBeforeEach(func() {
		reviewIssue = &CommitReviewIssue{
			CommitSHA:   commitSHA,
			CommitTitle: commitTitle,
			ReviewIssueCommonBody: &ReviewIssueCommonBody{
				CommitList:        commits,
				ReviewBlockerList: reviewBlockers,
				UserContent:       userContent,
			},
		}

		expectedTitle = fmt.Sprintf("Review commit %v: %v", commitSHA, commitTitle)
		expectedBody = strings.Join(expectedBodyLines, "\n")
	})

	// Assertion closures
	matchBody := func() {
		Expect(reviewIssue.FormatBody()).To(Equal(expectedBody))
	}

	matchTitle := func() {
		Expect(reviewIssue.FormatTitle()).To(Equal(expectedTitle))
	}

	assertBodyMatches := func() {
		It("should return the expected GitHub issue body", matchBody)
	}

	// Tests, at last!
	It("should return the expected GitHub issue title", matchTitle)

	Context("containing the commit list only", func() {

		BeforeEach(func() {
			// Review issue
			commits = commitList
			reviewBlockers = &ReviewBlockerList{}
			userContent = ""

			// Expected body.
			expectedBodyLines = []string{
				commitListString,
				emptyLine,
				emptyLine,
				userContentSeparator,
				defaultUserContentString,
			}
		})

		assertBodyMatches()
	})

	Context("containing the commit list and the review blocker list", func() {

		BeforeEach(func() {
			// Review issue
			commits = commitList
			reviewBlockers = reviewBlockerList
			userContent = ""

			// Expected body
			expectedBodyLines = []string{
				commitListString,
				emptyLine,
				reviewBlockerListString,
				emptyLine,
				userContentSeparator,
				defaultUserContentString,
			}
		})

		assertBodyMatches()
	})

	Context("containing the commit list and some user content", func() {

		BeforeEach(func() {
			// Review issue
			commits = commitList
			reviewBlockers = &ReviewBlockerList{}
			userContent = userContentSection

			// Expected body
			expectedBodyLines = []string{
				commitListString,
				emptyLine,
				emptyLine,
				userContentSeparator,
				userContentSection,
			}
		})

		assertBodyMatches()
	})

	Context("containing the commit list, the review blocker list and some user content", func() {

		BeforeEach(func() {
			// Review issue
			commits = commitList
			reviewBlockers = reviewBlockerList
			userContent = userContentSection

			// Expected body
			expectedBodyLines = []string{
				commitListString,
				emptyLine,
				reviewBlockerListString,
				emptyLine,
				userContentSeparator,
				userContentSection,
			}
		})

		assertBodyMatches()
	})
})
