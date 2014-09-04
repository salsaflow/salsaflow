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
	logScanAuthor
	logScanAuthorDate
	logScanCommitter
	logScanCommitDate
	logScanMsgTitle
	logScanMsgBody
)

const dateLayout = "Mon Jan 2 15:04:05 2006 -0700"

type Commit struct {
	SHA        string
	Author     string
	AuthorDate time.Time
	Committer  string
	CommitDate time.Time
	Title      string
	ChangeId   string
	StoryId    string
	Source     string
}

// ListStoryCommits returns the list of all commits that are associated with the given story.
func ListStoryCommits(storyId string) (commits []*Commit, stderr *bytes.Buffer, err error) {
	// Get the raw Git output.
	args := []string{
		"log",
		"--all",
		"--source",
		"--abbrev-commit",
		"--pretty=fuller",
		fmt.Sprintf("--grep=Story-Id: %v", storyId),
	}
	stdout, stderr, err := Git(args...)
	if err != nil {
		return
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

	commits = make([]*Commit, 0)

	var (
		lineNum     int
		nextState   int
		maybeHead   = true
		commit      *Commit
		headPattern = regexp.MustCompile("^commit[ \t]+([0-9a-f]+)[ \t]+(.+)$")
		scanner     = bufio.NewScanner(stdout)
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
				err = fmt.Errorf("invalid commit log (line %v): %v", lineNum, line)
				return
			}
			if commit != nil {
				commits = append(commits, commit)
			}
			commit = &Commit{
				SHA:    parts[1],
				Source: parts[2],
			}
			nextState = logScanAuthor

		case logScanAuthor:
			if strings.HasPrefix(line, "Merge") {
				continue
			}
			if strings.HasPrefix(line, "Author:     ") {
				commit.Author = line[12:]
				nextState = logScanAuthorDate
			} else {
				err = fmt.Errorf("invalid commit log (line %v): %v", lineNum, line)
				return
			}

		case logScanAuthorDate:
			if strings.HasPrefix(line, "AuthorDate: ") {
				var date time.Time
				dateString := line[12:]
				date, err = time.Parse(dateLayout, dateString)
				if err != nil {
					err = fmt.Errorf("invalid commit log (line %v): %v", lineNum, line)
					return
				}
				commit.AuthorDate = date
				nextState = logScanCommitter
			} else {
				err = fmt.Errorf("invalid commit log (line %v): %v", lineNum, line)
				return
			}

		case logScanCommitter:
			if strings.HasPrefix(line, "Commit:     ") {
				commit.Committer = line[12:]
				nextState = logScanCommitDate
			} else {
				err = fmt.Errorf("invalid commit log (line %v): %v", lineNum, line)
				return
			}

		case logScanCommitDate:
			if strings.HasPrefix(line, "CommitDate: ") {
				var date time.Time
				dateString := line[12:]
				date, err = time.Parse(dateLayout, dateString)
				if err != nil {
					err = fmt.Errorf("invalid commit log (line %v): %v", lineNum, line)
					return
				}
				commit.CommitDate = date
				nextState = logScanMsgTitle
			}

		case logScanMsgTitle:
			if line == "" {
				continue
			}
			commit.Title = strings.TrimSpace(line)
			nextState = logScanMsgBody

		case logScanMsgBody:
			if line == "" {
				continue
			}
			line = strings.TrimSpace(line)
			switch {
			case strings.HasPrefix(line, "Change-Id: "):
				if commit.ChangeId != "" {
					err = fmt.Errorf(
						"invalid commit log (line): duplicate Change-Id tag",
						lineNum)
					return
				}
				commit.ChangeId = line[11:]
				maybeHead = true
			case strings.HasPrefix(line, "Story-Id: "):
				if commit.StoryId != "" {
					err = fmt.Errorf(
						"invalid commit log (line %v): duplicate Story-Id tag",
						lineNum)
					return

				}
				commit.StoryId = line[10:]
				maybeHead = true
			}
		}
	}
	if commit != nil {
		commits = append(commits, commit)
	}

	err = scanner.Err()
	return
}
