package issues

import (
	// Stdlib
	"bytes"
	"fmt"
	"regexp"

	// Vendor
	"github.com/google/go-github/github"
)

// StoryReviewIssue represents the review issue associated with a story.
type StoryReviewIssue struct {
	StoryId      string
	StoryURL     string
	StorySummary string
	TrackerName  string
	StoryKey     string

	*ReviewIssueCommonBody
}

func NewStoryReviewIssue(
	storyId string,
	storyURL string,
	storySummary string,
	issueTracker string,
	storyKey string,
) *StoryReviewIssue {

	return &StoryReviewIssue{
		StoryId:               storyId,
		StoryURL:              storyURL,
		StorySummary:          storySummary,
		TrackerName:           issueTracker,
		StoryKey:              storyKey,
		ReviewIssueCommonBody: newReviewIssueCommonBody(),
	}
}

// Formatting ------------------------------------------------------------------

func (issue *StoryReviewIssue) FormatTitle() string {
	return fmt.Sprintf("Review story %v: %v", issue.StoryId, issue.StorySummary)
}

const (
	TagIssueTracker = "SF-Issue-Tracker"
	TagStoryKey     = "SF-Story-Key"
)

var storyReviewIssueBodyTemplate = fmt.Sprintf(`Story being reviewed: [{{.StoryId}}]({{.StoryURL}})

%v: {{.TrackerName}}
%v: {{.StoryKey}}

`, TagIssueTracker, TagStoryKey)

func (issue *StoryReviewIssue) FormatBody() string {
	var buffer bytes.Buffer
	execTemplate(&buffer, "story review issue body", storyReviewIssueBodyTemplate, issue)
	issue.ReviewIssueCommonBody.execTemplate(&buffer)
	return buffer.String()
}

// Parsing ---------------------------------------------------------------------

var (
	storyIssueTitleRegexp       = regexp.MustCompile(`^Review story ([^:]+): (.+)$`)
	storyIssueIntroLineRegexp   = regexp.MustCompile(`^Story being reviewed: \[([^\]]+)\]\(([^ ]+)\)`)
	storyIssueTrackerNameRegexp = regexp.MustCompile(fmt.Sprintf("^%v: (.+)$", TagIssueTracker))
	storyIssueStoryKeyRegexp    = regexp.MustCompile(fmt.Sprintf("^%v: (.+)$", TagStoryKey))
)

func parseStoryReviewIssue(issue *github.Issue) (*StoryReviewIssue, error) {
	var (
		title = *issue.Title
		body  = *issue.Body
	)

	// Parse the title.
	match := storyIssueTitleRegexp.FindStringSubmatch(title)
	if len(match) == 0 {
		return nil, &ErrInvalidTitle{issue}
	}

	// match[1] is the story ID, which we don't need right now.
	// It is returned from parseStoryReviewIssueIntro.
	storySummary := match[2]

	// Parse the body.
	var err error
	scanner := newBodyScanner(issue, body)

	// Parse the intro line.
	storyId, storyURL := parseStoryReviewIssueIntro(&err, scanner)

	// An empty line follows.
	readEmptyLine(&err, scanner)

	// Parse the metadata.
	issueTracker, storyKey := parseStoryReviewIssueMetadata(&err, scanner)

	// An empty line follows.
	readEmptyLine(&err, scanner)

	// Parse the common body.
	commonBody := parseRemainingIssueBody(&err, scanner)

	// Check the error, at last.
	if err != nil {
		return nil, err
	}

	// Return the context.
	return &StoryReviewIssue{
		StoryId:               storyId,
		StoryURL:              storyURL,
		StorySummary:          storySummary,
		TrackerName:           issueTracker,
		StoryKey:              storyKey,
		ReviewIssueCommonBody: commonBody,
	}, nil
}

func parseStoryReviewIssueIntro(
	err *error,
	scanner *bodyScanner,
) (storyId, storySummary string) {
	if *err != nil {
		return "", ""
	}

	// Read the intro line.
	line, _, ex := scanner.ReadLine()
	if ex != nil {
		*err = ex
		return
	}

	// Parse the line.
	match := storyIssueIntroLineRegexp.FindStringSubmatch(line)
	if len(match) != 3 {
		*err = scanner.CurrentLineInvalid()
		return "", ""
	}
	storyId, storySummary = match[1], match[2]

	// Return the results.
	return storyId, storySummary
}

func readEmptyLine(err *error, scanner *bodyScanner) {
	if *err != nil {
		return
	}

	// Read the next line.
	line, _, ex := scanner.ReadLine()
	if ex != nil {
		*err = ex
		return
	}

	// Make sure that it is empty.
	if line != "" {
		*err = scanner.CurrentLineInvalid()
	}
}

func parseStoryReviewIssueMetadata(
	err *error,
	scanner *bodyScanner,
) (issueTracker, storyKey string) {
	if *err != nil {
		return "", ""
	}

	// Read the tracker name line.
	line, _, ex := scanner.ReadLine()
	if ex != nil {
		*err = ex
		return "", ""
	}

	// Parse the tracker name tag.
	match := storyIssueTrackerNameRegexp.FindStringSubmatch(line)
	if len(match) != 2 {
		*err = scanner.TagNotFound(TagIssueTracker)
		return "", ""
	}
	issueTracker = match[1]

	// Read the story key line.
	line, _, ex = scanner.ReadLine()
	if ex != nil {
		*err = ex
		return "", ""
	}

	// Parse the story key tag.
	match = storyIssueStoryKeyRegexp.FindStringSubmatch(line)
	if len(match) != 2 {
		*err = scanner.TagNotFound(TagStoryKey)
		return "", ""
	}
	storyKey = match[1]

	// Return the results.
	return issueTracker, storyKey
}
