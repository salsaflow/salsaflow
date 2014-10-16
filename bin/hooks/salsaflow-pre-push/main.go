package main

import (
	// Stdlib
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"

	// Internal
	"github.com/salsita/salsaflow/errs"
	"github.com/salsita/salsaflow/git"
	"github.com/salsita/salsaflow/log"
)

const (
	secretRemote = "AreYouWhoIThinkYouAreHuh"
	secretReply  = "IAmSalsaFlowHookYaDoofus!"
)

func main() {
	// `repo init` uses this secret check to see whether this hook is installed.
	if len(os.Args) == 2 && os.Args[1] == secretRemote {
		fmt.Println(secretReply)
		return
	}

	// The hook is always invoked as `pre-push <remote-name> <push-url>`.
	if len(os.Args) != 3 {
		panic(fmt.Errorf("argv: %#v", os.Args))
	}

	// Run the main function.
	msg := "Perform SalsaFlow check"
	log.Run(msg)
	if err := run(os.Args[1], os.Args[2]); err != nil {
		errs.LogFail(msg, err)
		fmt.Println()
		os.Exit(1)
	}
}

func run(remoteName, pushURL string) error {
	// The commits that are being pushed are listed on stdin.
	// The format is <local ref> <local sha1> <remote ref> <remote sha1>,
	// so we parse the input and collect all the local hexshas.
	msg := "Parse the hook input"
	var hexshas []string
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Split(line, " ")
		if len(parts) != 4 {
			return errs.NewError(msg, nil, errors.New("invalid input line: "+line))
		}
		hexshas = append(hexshas, parts[1])
	}
	if err := scanner.Err(); err != nil {
		return errs.NewError(msg, nil, err)
	}

	// Get the relevant commit objects.
	msg = "Get the commit objects to be pushed"
	args := make([]string, 4, 4+len(hexshas))
	args[0] = "show"
	args[1] = "--abbrev-commit"
	args[2] = "--pretty=fuller"
	args[3] = "--source"
	args = append(args, hexshas...)
	stdout, stderr, err := git.Git(args...)
	if err != nil {
		return errs.NewError(msg, stderr, err)
	}

	// Parse the relevant commit objects.
	msg = "Parse the relevant git commits"
	commits, err := git.ParseCommits(stdout)
	if err != nil {
		return errs.NewError(msg, nil, err)
	}

	// Validate the commit messages.
	msg = "Validate the commit messages"
	var invalid bool

	stderr = new(bytes.Buffer)
	tw := tabwriter.NewWriter(stderr, 0, 8, 4, '\t', 0)
	io.WriteString(tw, "\n")
	io.WriteString(tw, "Commit SHA\tError\n")
	io.WriteString(tw, "==========\t=====\n")

	for _, commit := range commits {
		// All non-merge commits must have the Story-Id tag present in the commit message.
		if commit.StoryId == "" && commit.Merge == "" {
			fmt.Fprintf(tw, "%v\t%v\n", commit.SHA, "commit message: Story-Id tag missing")
			invalid = true
		}
	}

	if invalid {
		stderr.WriteString("\n")
		return errs.NewError(msg, stderr, nil)
	}
	return nil
}
