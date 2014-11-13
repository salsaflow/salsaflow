package releases

import (
	// Stdlib
	"bufio"
	"fmt"

	// Internal
	"github.com/salsita/salsaflow/git"

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

	// Get the list of all git tags.
	stdout, err := git.RunCommand("tag", "--list", "v*.*.*")
	if err != nil {
		return nil, err
	}

	// Parse the output to get the list of all the version tags.
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
		return nil, err
	}

	// In case there are no tags, take the whole trunk branch.
	if len(vers) == 0 {
		return git.ShowCommitRange(trunkBranch)
	}

	// Sort the versions and pick up the highest.
	semver.Sort(vers)
	lastRelease := vers[len(vers)-1]

	// Return the list of relevant commits.
	return git.ShowCommitRange(fmt.Sprintf("v%v..%v", lastRelease.String(), trunkBranch))
}
