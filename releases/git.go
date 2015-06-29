package releases

import (
	// Stdlib
	"bufio"
	"fmt"
	"sort"

	// Internal
	"github.com/salsaflow/salsaflow/errs"
	"github.com/salsaflow/salsaflow/git"
	"github.com/salsaflow/salsaflow/modules/common"
	"github.com/salsaflow/salsaflow/version"
)

// ListNewTrunkCommits returns the list of commits that are new since the last release.
func ListNewTrunkCommits() ([]*git.Commit, error) {
	// Get git config.
	config, err := git.LoadConfig()
	if err != nil {
		return nil, err
	}
	var (
		remoteName    = config.RemoteName()
		trunkBranch   = config.TrunkBranchName()
		stagingBranch = config.StagingBranchName()
	)

	// By default, use the staging branch as the --not part.
	// In other words, list commits that are on trunk,
	// but which are not reachable from the staging branch.
	// In case the staging branch doesn't exist, take the whole trunk.
	// That probably means that no release has ever been started,
	// so the staging branch has not been created yet.
	startingReference := stagingBranch
	err = git.CheckOrCreateTrackingBranch(stagingBranch, remoteName)
	if err != nil {
		if _, ok := err.(*git.ErrRefNotFound); ok {
			startingReference = trunkBranch
		}
		return nil, err
	}

	// Return the list of relevant commits.
	return git.ShowCommitRange(fmt.Sprintf("%v..%v", startingReference, trunkBranch))
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
		return nil, errs.NewError(task, err)
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
		return nil, errs.NewError(task, err)
	}

	// Parse the output to get sortable versions.
	var vers []*version.Version
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		line = line[1:] // strip "v"
		ver, _ := version.Parse(line)
		vers = append(vers, ver)
	}
	if err := scanner.Err(); err != nil {
		return nil, errs.NewError(task, err)
	}

	// Sort the versions.
	sort.Sort(version.Versions(vers))

	// Convert versions back to tag names and return.
	tgs := make([]string, 0, len(vers))
	for _, ver := range vers {
		tgs = append(tgs, "v"+ver.String())
	}
	return tgs, nil
}
