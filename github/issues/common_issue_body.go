package issues

import (
	// Stdlib
	"bufio"
	"regexp"
	"strings"

	// Vendor
	"github.com/google/go-github/github"
)

// ReviewIssueCommonBody represents the issue body part that is shared by all issue types,
// i.e. by both story review issues and commit review issues.
type ReviewIssueCommonBody struct {
	*CommitList
	*ReviewBlockerList

	UserContent string
}

func newReviewIssueCommonBody() *ReviewIssueCommonBody {
	return &ReviewIssueCommonBody{
		CommitList:        &CommitList{},
		ReviewBlockerList: &ReviewBlockerList{},
	}
}

// Parsing ---------------------------------------------------------------------

const (
	stateCommitList = iota + 1
	stateCommitItems
	stateReviewBlockerList
	stateReviewBlockerItems
	stateUserContent
)

const separator = "----------"

var (
	commitItemRegexp  = regexp.MustCompile(`^- \[([ xX])\] ([0-9a-f]+): (.+)$`)
	blockerItemRegexp = regexp.MustCompile(`^- \[([ xX])\] \[blocker ([0-9]+)\]\(([^)]+)\) \(commit ([0-9a-f]+)\): (.+)$`)
)

func parseRemainingIssueBody(
	issue *github.Issue,
	scanner *bufio.Scanner,
	lineNo int,
) (*ReviewIssueCommonBody, error) {

	ctx := newReviewIssueCommonBody()

	// The common issue body begins at the commit list intro line.
	state := stateCommitList

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		lineNo++

		// In case this is the separator, we start adding to the user content,
		// i.e. the content manually added by the user.
		if line == separator {
			state = stateUserContent
			continue
		}

		switch state {
		// stateCommitList - the line preceeding the commit item list.
		case stateCommitList:
			state = stateCommitItems

		// stateCommitItems - the lines forming the commit checkbox.
		case stateCommitItems:
			// And empty line is separating the commit list from the review blocker list.
			if line == "" {
				state = stateReviewBlockerList
				continue
			}

			// Parse the line as a commit item.
			match := commitItemRegexp.FindStringSubmatch(line)
			if len(match) == 0 {
				return nil, &ErrInvalidBody{issue, lineNo, line}
			}

			// Add the commit to the commit list.
			done := match[1] != " "
			sha, title := match[2], match[3]
			ctx.AddCommit(sha, title, done)

		// stateReviewBlockerList - the line preceeding the review blocker item list.
		case stateReviewBlockerList:
			state = stateReviewBlockerItems

		// stateReviewBlockerItems - the lines forming the review blocker checkbox.
		case stateReviewBlockerItems:
			// Let's just skip empty lines.
			// No need to be too strict here.
			if line == "" {
				continue
			}

			// Parse the line as a review blocker.
			match := blockerItemRegexp.FindStringSubmatch(line)
			if len(match) == 0 {
				return nil, &ErrInvalidBody{issue, lineNo, line}
			}

			// Add the blocker to the blocker list.
			fixed := match[1] != " "
			commentURL, commitSHA, summary := match[3], match[4], match[5]
			ctx.AddReviewBlocker(commitSHA, commentURL, summary, fixed)

		// stateUserContent - content added by the user manually.
		case stateUserContent:
			ctx.UserContent += line + "\n"

		default:
			panic("unknown state")
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return ctx, nil
}
