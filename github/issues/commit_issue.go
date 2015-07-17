package issues

import (
	// Stdlib
	"bufio"
	"bytes"
	"fmt"
	"regexp"
	"strings"

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
	ctx.AddCommit(commitSHA, commitTitle, false)
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

func parseCommitReviewIssue(issue *github.Issue) (*CommitReviewIssue, error) {
	var (
		title = *issue.Title
		body  = *issue.Body
	)

	// Parse the title.
	titleRegexp := regexp.MustCompile(`^Review commit ([^:]+): (.+)$`)
	match := titleRegexp.FindStringSubmatch(title)
	if len(match) == 0 {
		return nil, &ErrInvalidTitle{issue}
	}
	commitSHA, commitTitle := match[1], match[2]

	// Parse the body.
	// There is actually nothing else than the common part,
	// so we can simply call parseRemainingIssueBody.
	scanner := bufio.NewScanner(strings.NewReader(body))
	bodyCtx, err := parseRemainingIssueBody(issue, scanner, 0)
	if err != nil {
		return nil, err
	}

	// Return the context.
	return &CommitReviewIssue{commitSHA, commitTitle, bodyCtx}, nil
}
