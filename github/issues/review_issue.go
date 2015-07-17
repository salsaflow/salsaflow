package issues

import (
	// Stdlib
	"fmt"
	"strings"

	// Vendor
	"github.com/google/go-github/github"
)

// ReviewIssue represents the common interface for all review issue types.
type ReviewIssue interface {

	// AddCommit adds the commit to the commit checklist.
	AddCommit(reviewed bool, commitSHA, commitTitle string) (added bool)

	// CommitItems returns the list of commits contained in the commit checklist.
	CommitItems() []*CommitItem

	// AddReviewBlocker adds the review blocker to the blocker checkbox.
	AddReviewBlocker(fixed bool, commentURL, commitSHA, blockerSummary string) (added bool)

	// ReviewBlockerItems returns the list of blockers contained in the blocker checklist.
	ReviewBlockerItems() []*ReviewBlockerItem

	// FormatTitle returns the review issue title for the given issue type.
	FormatTitle() string

	// FormatBody returns the review issue body for the given issue type.
	FormatBody() string
}

// ParseReviewIssue parses the given GitHub review issue and returns
// a *StoryReviewIssue or *CommitReviewIssue based on the issue type,
// which both implement ReviewIssue interface.
func ParseReviewIssue(issue *github.Issue) (ReviewIssue, error) {
	// Use the title prefix to decide the review issue type.
	switch {
	case strings.HasPrefix(*issue.Title, "Review story"):
		return parseStoryReviewIssue(issue)
	case strings.HasPrefix(*issue.Title, "Review commit"):
		return parseCommitReviewIssue(issue)
	default:
		return nil, &ErrUnknownReviewIssueType{issue}
	}
}

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
	return findReviewIssueByTitle(client, owner, repo, substring)
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
	return findReviewIssueByTitle(client, owner, repo, substring)
}

func findReviewIssueByTitle(
	client *github.Client,
	owner string,
	repo string,
	substring string,
) (*github.Issue, error) {

	query := fmt.Sprintf(
		`"%v" repo:"%v/%v" label:review type:issue state:open state:closed in:title`,
		substring, owner, repo)

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
			if strings.Contains(*issue.Title, substring) {
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
