package issues

import (
	// Stdlib
	"fmt"
	"regexp"
	"strings"

	// Vendor
	"github.com/google/go-github/github"
)

// FindReviewIssueForCommit returns the review issue
// associated with the given commit.
func FindReviewIssueForCommit(
	client *github.Client,
	owner string,
	repo string,
	commitSHA string,
) (*github.Issue, error) {

	// Find the relevant review issue.
	substring := fmt.Sprintf("Review commit %v:", commitSHA[:7])
	return findReviewIssue(client, owner, repo, substring, "in:title", matchesTitle(substring))
}

// FindReviewIssueForStory returns the review issue
// associated with the given story.
func FindReviewIssueForStory(
	client *github.Client,
	owner string,
	repo string,
	storyId string,
) (*github.Issue, error) {

	// Find the relevant review issue.
	substring := fmt.Sprintf("Review story %v:", storyId)
	return findReviewIssue(client, owner, repo, substring, "in:title", matchesTitle(substring))
}

// FindReviewIssueByCommitItem searches review issues for the one that
// contains the given commit in its commit checklist.
func FindReviewIssueByCommitItem(
	client *github.Client,
	owner string,
	repo string,
	commitSHA string,
) (*github.Issue, error) {

	// Use the first 7 chars from the commit hexsha.
	shortSHA := commitSHA[:7]

	// Commit item regexp.
	re := regexp.MustCompile(fmt.Sprintf(`[-*] \[[xX ]\] %v[:]`, shortSHA))

	// Perform the search.
	return findReviewIssue(client, owner, repo, shortSHA, "in:body", matchesBodyRe(re))
}

type matcherFunc func(*github.Issue) bool

func findReviewIssue(
	client *github.Client,
	owner string,
	repo string,
	substring string,
	queryOpts string,
	matches matcherFunc,
) (*github.Issue, error) {

	query := fmt.Sprintf(
		`"%v" repo:"%v/%v" label:review type:issue state:open state:closed %v`,
		substring, owner, repo, queryOpts)

	searchOpts := &github.SearchOptions{}
	searchOpts.Page = 1
	searchOpts.PerPage = 50

	var searched int

	for {
		// Fetch another page.
		result, _, err := client.Search.Issues(query, searchOpts)
		if err != nil {
			return nil, err
		}

		// Check the issues for exact string match.
		for _, issue := range result.Issues {
			if matches(&issue) {
				return &issue, nil
			}
		}

		// Check whether we have reached the end or not.
		searched += len(result.Issues)
		if searched == *result.Total {
			return nil, nil
		}

		// Check the next page in the next iteration.
		searchOpts.Page += 1
	}
}

func matchesTitle(substring string) matcherFunc {
	return func(issue *github.Issue) bool {
		return strings.Contains(*issue.Title, substring)
	}
}

func matchesBodyRe(re *regexp.Regexp) matcherFunc {
	return func(issue *github.Issue) bool {
		return re.MatchString(*issue.Body)
	}
}
