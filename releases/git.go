package releases

import (
	// Stdlib
	"bufio"
	"fmt"

	// Internal
	"github.com/salsaflow/salsaflow/errs"
	"github.com/salsaflow/salsaflow/git"
	"github.com/salsaflow/salsaflow/modules/common"

	// Other
	"github.com/coreos/go-semver/semver"
)

// ListNewTrunkCommits returns the list of commits that are new since the last release.
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

// ListStoryIdsToBeAssigned lists the story IDs that are associated with
// the commits that modified trunk since the last release, i.e. with the commits
// as returned by ListNewTrunkCommits.
//
// Only the story IDs matching the issue tracker that is passed in are returned.
func ListStoryIdsToBeAssigned(tracker common.IssueTracker) ([]string, error) {
	// Get the commits that modified trunk.
	task := "Get the commits that modified trunk"
	commits, err := ListNewTrunkCommits()
	if err != nil {
		return nil, errs.NewError(task, err, nil)
	}

	// Collect the story IDs.
	idSet := make(map[string]struct{}, len(commits))
	for _, commit := range commits {
		// Skip empty tags.
		if commit.StoryIdTag == "" {
			continue
		}

		// Parse the tag to get the story ID.
		storyId, err := tracker.StoryTagToReadableStoryId(commit.StoryIdTag)
		if err != nil {
			continue
		}

		// Add the ID to the set.
		idSet[storyId] = struct{}{}
	}

	// Convert the set to a list.
	idList := make([]string, 0, len(idSet))
	for id := range idSet {
		idList = append(idList, id)
	}

	// Return the final list of story IDs.
	return idList, nil
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
