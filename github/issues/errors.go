package issues

import (
	// Stdlib
	"fmt"

	// Vendor
	"github.com/google/go-github/github"
)

// ErrUnknownReviewIssueType is returned when the issue type cannot be recognized.
type ErrUnknownReviewIssueType struct {
	issue *github.Issue
}

func (err *ErrUnknownReviewIssueType) Error() string {
	return fmt.Sprintf("failed to detect review issue type for %v", *err.issue.HTMLURL)
}

// ErrInvalidTitle is returned when the issue title is malformed.
type ErrInvalidTitle struct {
	issue *github.Issue
}

func (err *ErrInvalidTitle) Error() string {
	return fmt.Sprintf("issue %v: invalid issue title", *err.issue.HTMLURL)
}

// ErrInvalidBody is returned when the issue body is malformed.
type ErrInvalidBody struct {
	issue  *github.Issue
	lineNo int
	line   string
}

func (err *ErrInvalidBody) Error() string {
	return fmt.Sprintf("issue %v: invalid issue body (line %v): %v",
		*err.issue.HTMLURL, err.lineNo, err.line)
}

// ErrTagNotFound is returned when there is a SalsaFlow metadata tag missing in the body.
type ErrTagNotFound struct {
	issue *github.Issue
	tag   string
}

func (err *ErrTagNotFound) Error() string {
	return fmt.Sprintf("issue %v: tag not found: %v", *err.issue.HTMLURL, err.tag)
}
