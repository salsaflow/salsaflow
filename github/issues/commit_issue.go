package issues

import (
	// Stdlib
	"bytes"
	"fmt"
	"regexp"

	// Vendor
	"github.com/google/go-github/github"
)

// CommitReviewIssue represents a review issue associated with a commit.
type CommitReviewIssue struct {
	// Title
	CommitSHA   string
	CommitTitle string

	// Body
	*ReviewIssueCommonBody
}

func NewCommitReviewIssue(commitSHA, commitTitle string) *CommitReviewIssue {
	ctx := &CommitReviewIssue{
		CommitSHA:             commitSHA,
		CommitTitle:           commitTitle,
		ReviewIssueCommonBody: newReviewIssueCommonBody(),
	}
	ctx.AddCommit(false, commitSHA, commitTitle)
	return ctx
}

// Formatting ------------------------------------------------------------------

func (ctx *CommitReviewIssue) FormatTitle() string {
	return fmt.Sprintf("Review commit %v: %v", ctx.CommitSHA, ctx.CommitTitle)
}

func (ctx *CommitReviewIssue) FormatBody() string {
	var buffer bytes.Buffer
	ctx.ReviewIssueCommonBody.execTemplate(&buffer)
	return buffer.String()
}

// Parsing ---------------------------------------------------------------------

var commitIssueTitleRegexp = regexp.MustCompile(`^Review commit ([^:]+): (.+)$`)

func parseCommitReviewIssue(issue *github.Issue) (*CommitReviewIssue, error) {
	var (
		title = *issue.Title
		body  = *issue.Body
	)

	// Parse the title.
	match := commitIssueTitleRegexp.FindStringSubmatch(title)
	if len(match) == 0 {
		return nil, &ErrInvalidTitle{issue}
	}
	commitSHA, commitTitle := match[1], match[2]

	// Parse the body.
	// There is actually nothing else than the common part,
	// so we can simply call parseRemainingIssueBody.
	var err error
	commonBody := parseRemainingIssueBody(&err, newBodyScanner(issue, body))
	if err != nil {
		return nil, err
	}

	// Return the context.
	return &CommitReviewIssue{commitSHA, commitTitle, commonBody}, nil
}
