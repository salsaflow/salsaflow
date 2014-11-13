package git

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"regexp"
	"strings"
	"text/template"
	"time"
)

const (
	logScanHead = iota + 1
	logScanMerge
	logScanAuthor
	logScanAuthorDate
	logScanCommitter
	logScanCommitDate
	logScanMsgTitle
	logScanMsgBody
	logScanDiff
)

const dateLayout = "Mon Jan 2 15:04:05 2006 -0700"

var (
	ChangeIdTagPattern = regexp.MustCompile("^(?i)[ \t]*Change-Id:[ \t]+([^ \t]+)")
	StoryIdTagPattern  = regexp.MustCompile("^(?i)[ \t]*Story-Id:[ \t]+([^ \t]+)")
)

type Commit struct {
	SHA          string
	Source       string
	Merge        string
	Author       string
	AuthorDate   time.Time
	Committer    string
	CommitDate   time.Time
	MessageTitle string
	Message      string
	ChangeId     string
	StoryId      string
}

var commitTemplate = `commit {{.SHA}} {{.Source}}{{with .Merge}}Merge: {{.}}{{"\n"}}{{end}}
Author:     {{.Author}}
AuthorDate: {{.AuthorDate}}
Commit:     {{.Committer}}
CommitDate: {{.CommitDate}}

{{.Message | indent }}
`

// Dump writes a human-friendly representation of the commit to the writer.
// The output closely resembles the output of git log.
func (commit *Commit) Dump(wr io.Writer) error {
	funcMap := template.FuncMap{
		"indent": func(content string) (string, error) {
			var out bytes.Buffer
			scanner := bufio.NewScanner(strings.NewReader(content))
			for scanner.Scan() {
				if _, err := fmt.Fprintln(&out, "    ", scanner.Text()); err != nil {
					return "", err
				}
			}
			if err := scanner.Err(); err != nil {
				return "", err
			}
			return out.String(), nil
		},
	}

	tpl := template.Must(template.New("commit").Funcs(funcMap).Parse(commitTemplate))
	return tpl.Execute(wr, commit)
}

// StoryIds returns the list of unique story IDs that can be found in the commits.
func StoryIds(commits []*Commit) []string {
	idMap := make(map[string]struct{}, len(commits))
	for _, commit := range commits {
		if commit.StoryId != "" && commit.StoryId != "unassigned" {
			idMap[commit.StoryId] = struct{}{}
		}
	}

	idList := make([]string, 0, len(idMap))
	for id := range idMap {
		idList = append(idList, id)
	}
	return idList
}

// ListStoryCommits returns the list of all commits that are associated with the given story.
func ListStoryCommits(storyId string) ([]*Commit, error) {
	return GrepCommits("Story-Id: " + storyId)
}

func GrepCommits(filter string) ([]*Commit, error) {
	args := []string{
		"log",
		"--all",
		"--source",
		"--abbrev-commit",
		"--pretty=fuller",
		"--grep=" + filter,
	}
	stdout, err := Run(args...)
	if err != nil {
		return nil, err
	}

	return ParseCommits(stdout.Bytes())
}

// ShowCommits returns the list of commits associated with the given revisions.
func ShowCommits(revisions ...string) ([]*Commit, error) {
	args := make([]string, 4, 4+len(revisions))
	args[0] = "show"
	args[1] = "--source"
	args[2] = "--abbrev-commit"
	args[3] = "--pretty=fuller"
	args = append(args, revisions...)

	stdout, err := Run(args...)
	if err != nil {
		return nil, err
	}

	return ParseCommits(stdout.Bytes())
}

// ShowCommitRange returns the list of commits specified by the given Git revision range.
func ShowCommitRange(revisionRange string) ([]*Commit, error) {
	args := []string{
		"log",
		"--source",
		"--abbrev-commit",
		"--pretty=fuller",
		revisionRange,
	}
	stdout, err := Run(args...)
	if err != nil {
		return nil, err
	}

	return ParseCommits(stdout.Bytes())
}

// Parse git log output, which is a sequence of Git commits looking like
//
// commit $hexsha  $source
// Author: $author
// Date:   $date
//
//    $title
//
//    $body
//
//    Change-Id: $changeId
//    Story-Id: $storyId
//
// The reason why we are parsing the regular output (not using --pretty=format:)
// is that not all formatting options are supported. For example, log --all --source
// contains some information that cannot be easily taken by using --pretty=format:
func ParseCommits(input []byte) (commits []*Commit, err error) {
	cs := make([]*Commit, 0)

	var (
		lineNum   int
		nextState int
		maybeHead = true

		commit  *Commit
		message []string

		headPattern = regexp.MustCompile("^commit[ \t]+([0-9a-f]+)[ \t]+(.+)$")
		numSpaces   int
		scanner     = bufio.NewScanner(bytes.NewReader(input))
	)
	for scanner.Scan() {
		line := scanner.Text()
		lineNum++

		if maybeHead {
			if headPattern.MatchString(line) {
				nextState = logScanHead
				maybeHead = false
			}
		}

		switch nextState {
		case logScanHead:
			parts := headPattern.FindStringSubmatch(line)
			if len(parts) != 3 {
				err = fmt.Errorf("failed to parse git log [line %v]: %v", lineNum, line)
				return
			}
			// Close the previous commit.
			if commit != nil {
				cs = append(cs, finaliseCommit(commit, message))
			}
			// Start a new commit.
			commit = &Commit{
				SHA:    parts[1],
				Source: parts[2],
			}
			message = make([]string, 0)
			nextState = logScanMerge

		case logScanMerge:
			// Only present when this is a merge commit.
			if strings.HasPrefix(line, "Merge: ") {
				commit.Merge = line[7:]
				nextState = logScanAuthor
				continue
			}
			// In case the line is not present, fall through to Author.
			fallthrough

		case logScanAuthor:
			if strings.HasPrefix(line, "Author:     ") {
				commit.Author = line[12:]
				nextState = logScanAuthorDate
			} else {
				err = fmt.Errorf("failed to parse git log [commit %v]: %v", commit.SHA, line)
				return
			}

		case logScanAuthorDate:
			if strings.HasPrefix(line, "AuthorDate: ") {
				var date time.Time
				dateString := line[12:]
				date, err = time.Parse(dateLayout, dateString)
				if err != nil {
					err = fmt.Errorf("failed to parse git log [commit %v]: %v", commit.SHA, line)
					return
				}
				commit.AuthorDate = date
				nextState = logScanCommitter
			} else {
				err = fmt.Errorf("failed to parse git log [commit %v]: %v", commit.SHA, line)
				return
			}

		case logScanCommitter:
			if strings.HasPrefix(line, "Commit:     ") {
				commit.Committer = line[12:]
				nextState = logScanCommitDate
			} else {
				err = fmt.Errorf("failed to parse git log [commit %v]: %v", commit.SHA, line)
				return
			}

		case logScanCommitDate:
			if strings.HasPrefix(line, "CommitDate: ") {
				var date time.Time
				dateString := line[12:]
				date, err = time.Parse(dateLayout, dateString)
				if err != nil {
					err = fmt.Errorf("failed to parse git log [commit %v]: %v", commit.SHA, line)
					return
				}
				commit.CommitDate = date
				nextState = logScanMsgTitle
			}

		case logScanMsgTitle:
			if line == "" {
				continue
			}
			trimmedLine := strings.TrimSpace(line)
			numSpaces = strings.Index(line, trimmedLine)
			commit.MessageTitle = trimmedLine
			message = append(message, line[numSpaces:])
			nextState = logScanMsgBody

		case logScanMsgBody:
			trimmedLine := strings.TrimSpace(line)
			switch {
			// In case we are parsing the output of git show,
			// we have to handle the diff section as well.
			case strings.HasPrefix(line, "diff --git"):
				nextState = logScanDiff
				continue
			case ChangeIdTagPattern.MatchString(trimmedLine):
				if commit.ChangeId != "" {
					err = fmt.Errorf("git log [commit %v]: duplicate Change-Id tag", commit.SHA)
					return
				}
				parts := ChangeIdTagPattern.FindStringSubmatch(trimmedLine)
				if len(parts) != 2 {
					err = fmt.Errorf("git log [commit %v]: invalid Change-Id tag", commit.SHA)
					return
				}
				commit.ChangeId = parts[1]
			case StoryIdTagPattern.MatchString(trimmedLine):
				if commit.StoryId != "" {
					err = fmt.Errorf("git log [commit %v]: duplicate Story-Id tag", commit.SHA)
					return
				}
				parts := StoryIdTagPattern.FindStringSubmatch(trimmedLine)
				if len(parts) != 2 {
					err = fmt.Errorf("git log [commit %v]: invalid Story-Id tag", commit.SHA)
					return
				}
				commit.StoryId = parts[1]
			}
			if len(line) >= numSpaces {
				line = line[numSpaces:]
			}
			message = append(message, line)
			maybeHead = true

		case logScanDiff:
			continue
		}
	}
	if commit != nil {
		cs = append(cs, finaliseCommit(commit, message))
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	// Reverse the list of commits so that the first commit in the chain
	// is also the first commit in the list being returned.
	reversed := make([]*Commit, len(cs))
	for i, commit := range cs {
		reversed[len(reversed)-1-i] = commit
	}
	return reversed, nil
}

func finaliseCommit(commit *Commit, messageLines []string) *Commit {
	// Make sure the commit message ends with no empty lines.
	// This also means that there is no newline at the end.
	for messageLines[len(messageLines)-1] == "" {
		messageLines = messageLines[:len(messageLines)-1]
	}
	// Concatenate the lines and set the resulting commit message.
	commit.Message = strings.Join(messageLines, "\n")
	return commit
}
