package issues

import (
	// Stdlib
	"bufio"
	"io"
	"strings"

	// Vendor
	"github.com/google/go-github/github"
)

type bodyScanner struct {
	issue   *github.Issue
	scanner *bufio.Scanner

	line   string
	lineNo int
	err    error
}

func newBodyScanner(issue *github.Issue, issueBody string) *bodyScanner {
	return &bodyScanner{
		issue:   issue,
		scanner: bufio.NewScanner(strings.NewReader(issueBody)),
	}
}

func (bs *bodyScanner) CurrentLine() (line string, lineNo int, err error) {
	return bs.line, bs.lineNo, bs.err
}

func (bs *bodyScanner) ReadLine() (line string, lineNo int, err error) {
	// Keep returning the error once encountered.
	if err := bs.err; err != nil {
		return "", 0, err
	}

	// Read the next line.
	scanner := bs.scanner
	done := !scanner.Scan()
	if done {
		// In case the scanner stopped,
		// set the error to the scanning error.
		// Use io.EOF in case everything is ok.
		err := scanner.Err()
		if err == nil {
			err = io.EOF
		} else {
			err = &ErrScanning{bs.issue, bs.lineNo, bs.line, err}
		}
		bs.line = ""
		bs.lineNo = 0
		bs.err = err
	} else {
		bs.line = strings.TrimSpace(scanner.Text())
		bs.lineNo++
	}

	return bs.line, bs.lineNo, bs.err
}

func (bs *bodyScanner) CurrentLineInvalid() error {
	return &ErrInvalidBody{bs.issue, bs.lineNo, bs.line}
}

func (bs *bodyScanner) TagNotFound(tag string) error {
	return &ErrTagNotFound{bs.issue, tag}
}
