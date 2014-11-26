package releases

import (
	// Stdlib
	"bufio"
	"fmt"

	// Internal
	"github.com/salsaflow/salsaflow/errs"
	"github.com/salsaflow/salsaflow/git"

	// Other
	"github.com/coreos/go-semver/semver"
)

// ListCurrentReleaseCommits returns the list of commits that are new since the last release.
func ListNewTrunkCommits() ([]*git.Commit, error) {
	// Get git config.
	config, err := git.LoadConfig()
	if err != nil {
		return nil, err
	}
	trunkBranch := config.TrunkBranchName()

	// Get sorted release tags.
	tags, err := ListTags()
	if err != nil {
		return nil, err
	}

	// In case there are no tags, take the whole trunk branch.
	if len(tags) == 0 {
		return git.ShowCommitRange(trunkBranch)
	}

	// Return the list of relevant commits.
	lastTag := tags[len(tags)-1]
	return git.ShowCommitRange(fmt.Sprintf("%v..%v", lastTag, trunkBranch))
}

// ListTags returns the list of all release tags, sorted by the versions they represent.
func ListTags() (tags []string, err error) {
	var task = "Get release tags"

	// Get all release tags.
	stdout, err := git.RunCommand("tag", "--list", "v*.*.*")
	if err != nil {
		return nil, errs.NewError(task, err, nil)
	}

	// Parse the output to get sortable versions.
	var vers []*semver.Version
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		line = line[1:] // strip "v"
		ver, _ := semver.NewVersion(line)
		vers = append(vers, ver)
	}
	if err := scanner.Err(); err != nil {
		return nil, errs.NewError(task, err, nil)
	}

	// Sort the versions.
	semver.Sort(vers)

	// Convert versions back to tag names and return.
	tgs := make([]string, 0, len(vers))
	for _, ver := range vers {
		tgs = append(tgs, "v"+ver.String())
	}
	return tgs, nil
}
