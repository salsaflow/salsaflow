package hooks

import (
	// Stdlib
	"bytes"
	"fmt"
	"io"

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

	yellow := color.New(color.FgYellow).SprintFunc()
	refPrefixLen := len("refs/heads/")
	for _, commit := range commits {
		fmt.Fprintf(&output, "  %v | %v | %v\n",
			yellow(commit.SHA), yellow(commit.Source[refPrefixLen:]),
			prompt.ShortenCommitTitle(commit.MessageTitle))
	}

	// Write the output to the writer.
	return io.Copy(ansicolor.NewAnsiColorWriter(writer), &output)
}
