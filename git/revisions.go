package git

import (
	// Stdlib
	"fmt"
	"strings"

	// Internal
	"github.com/salsaflow/salsaflow/errs"
)

// RevisionsToCommitList uses git rev-list --no-walk to turn given list
// of revision specifications into a list of commits.
//
// The list is passed to git rev-list --no-walk unchanges, so check the docs
// to understand the exact behaviour of this function.
func RevisionsToCommitList(revisions ...string) ([]string, error) {
	task := fmt.Sprintf("Convert give revision list to a list of commits: %v", revisions)

	// Ask git to give us the revision list.
	argList := make([]string, 2, 2+len(revisions))
	argList[0] = "rev-list"
	argList[1] = "--no-walk"
	argList = append(argList, revisions...)
	stdout, err := Run(argList...)
	if err != nil {
		return nil, errs.NewError(task, err)
	}

	// Split the output into lines.
	lines := strings.Split(stdout.String(), "\n")

	// Drop the last element, it is an empty string.
	lines = lines[:len(lines)-1]

	// Reverse the list.
	hashes := make([]string, len(lines))
	for i, line := range lines {
		hashes[len(hashes)-i-1] = strings.TrimSpace(line)
	}

	// Return the hashes.
	return hashes, nil
}
