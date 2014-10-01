package git

import (
	"bufio"
	"bytes"
	"fmt"
	"regexp"
	"strconv"
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
	StoryId    int
	Source     string
}

// ListStoryCommits returns the list of all commits that are associated with the given story.
func ListStoryCommits(storyId string) (commits []*Commit, stderr *bytes.Buffer, err error) {
	return GrepCommits("Story-Id: " + storyId)
}

func GrepCommits(filter string) (commits []*Commit, stderr *bytes.Buffer, err error) {
	// Get the raw Git output.
	args := []string{
		"log",
		"--all",
		"--source",
		"--abbrev-commit",
		"--pretty=fuller",
		"--grep=" + filter,
	}
	sout, serr, err := Git(args...)
	if err != nil {
		return nil, serr, err
	}

	return parseCommitLog(sout)
}

// Returns list of commit on branch `ref` compared to branch `parent`.
func ListBranchCommits(ref string, parent string) (commits []*Commit, stderr *bytes.Buffer, err error) {
	args := []string{
		"log",
		"--source",
		"--abbrev-commit",
		"--pretty=fuller",
		parent + ".." + ref,
	}
	sout, serr, err := Git(args...)
	if err != nil {
		return nil, serr, err
	}

	return parseCommitLog(sout)
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
func parseCommitLog(sout *bytes.Buffer) (commits []*Commit, stderr *bytes.Buffer, err error) {

	cs := make([]*Commit, 0)

	var (
		lineNum     int
		nextState   int
		maybeHead   = true
		commit      *Commit
		headPattern = regexp.MustCompile("^commit[ \t]+([0-9a-f]+)[ \t]+(.+)$")
		scanner     = bufio.NewScanner(sout)
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
				err = fmt.Errorf("commit log (line %v): %v", lineNum, line)
				return
			}
			if commit != nil {
				cs = append(cs, commit)
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
				err = fmt.Errorf("commit log (line %v, commit %v): %v", lineNum, commit.SHA, line)
				return
			}

		case logScanAuthorDate:
			if strings.HasPrefix(line, "AuthorDate: ") {
				var date time.Time
				dateString := line[12:]
				date, err = time.Parse(dateLayout, dateString)
				if err != nil {
					err = fmt.Errorf("commit log (line %v, commit %v): %v",
						lineNum, commit.SHA, line)
					return
				}
				commit.AuthorDate = date
				nextState = logScanCommitter
			} else {
				err = fmt.Errorf("commit log (line %v, commit %v): %v", lineNum, commit.SHA, line)
				return
			}

		case logScanCommitter:
			if strings.HasPrefix(line, "Commit:     ") {
				commit.Committer = line[12:]
				nextState = logScanCommitDate
			} else {
				err = fmt.Errorf("commit log (line %v, commit %v): %v", lineNum, commit.SHA, line)
				return
			}

		case logScanCommitDate:
			if strings.HasPrefix(line, "CommitDate: ") {
				var date time.Time
				dateString := line[12:]
				date, err = time.Parse(dateLayout, dateString)
				if err != nil {
					err = fmt.Errorf("commit log (line %v, commit %v): %v",
						lineNum, commit.SHA, line)
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
						"commit log (line %v, commit %v): duplicate Change-Id tag",
						lineNum, commit.SHA)
					return
				}
				commit.ChangeId = line[11:]
				maybeHead = true
			case strings.HasPrefix(line, "Story-Id: "):
				if commit.StoryId != 0 {
					err = fmt.Errorf(
						"commit log (line %v): duplicate Story-Id tag",
						lineNum, commit.SHA)
					return

				}
				var (
					storyIdString = line[10:]
					storyId       int
				)
				storyId, err = strconv.Atoi(storyIdString)
				if err != nil {
					err = fmt.Errorf(
						"commit log (line %v, commit %v): invalid Story-Id: %v",
						lineNum, commit.SHA, storyIdString)
					return
				}
				commit.StoryId = storyId
				maybeHead = true
			}
		}
	}
	if commit != nil {
		cs = append(cs, commit)
	}

	if err := scanner.Err(); err != nil {
		return nil, nil, err
	}
	return cs, nil, nil
}
