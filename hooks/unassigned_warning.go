package hooks

import (
	// Stdlib
	"bytes"
	"fmt"
	"io"
	"regexp"

	// Internal
	"github.com/salsaflow/salsaflow/git"
	"github.com/salsaflow/salsaflow/prompt"

	// Vendor
	"github.com/fatih/color"
	"github.com/shiena/ansicolor"
)

func PrintUnassignedWarning(writer io.Writer, commits []*git.Commit) (n int64, err error) {
	var output bytes.Buffer

	// Let's be colorful!
	redBold := color.New(color.FgRed).Add(color.Bold).SprintFunc()
	fmt.Fprint(&output,
		redBold("Warning: There are some commits missing the Story-Id tag.\n"))

	red := color.New(color.FgRed).SprintFunc()
	fmt.Fprint(&output,
		red("Make sure that this is alright before proceeding further.\n\n"))

	hashRe := regexp.MustCompile("^[0-9a-f]{40}$")

	yellow := color.New(color.FgYellow).SprintFunc()

	for _, commit := range commits {
		var (
			sha    = commit.SHA
			source = commit.Source
			title  = prompt.ShortenCommitTitle(commit.MessageTitle)
		)
		if hashRe.MatchString(commit.Source) {
			source = "unknown commit source branch"
		}

		fmt.Fprintf(&output, "  %v | %v | %v\n", yellow(sha), yellow(source), title)
	}

	// Write the output to the writer.
	return io.Copy(ansicolor.NewAnsiColorWriter(writer), &output)
}
