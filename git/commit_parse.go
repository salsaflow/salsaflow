package git

import (
	"bufio"
	"bytes"
	"fmt"
	"regexp"
	"strings"
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
)

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
				if commit.ChangeIdTag != "" {
					err = fmt.Errorf("git log [commit %v]: duplicate Change-Id tag", commit.SHA)
					return
				}
				parts := ChangeIdTagPattern.FindStringSubmatch(trimmedLine)
				if len(parts) != 2 {
					err = fmt.Errorf("git log [commit %v]: invalid Change-Id tag", commit.SHA)
					return
				}
				commit.ChangeIdTag = parts[1]
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
