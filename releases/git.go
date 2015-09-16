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
	"github.com/salsaflow/salsaflow/repo"
	"github.com/salsaflow/salsaflow/version"
)

// ListNewTrunkCommits returns the list of commits that are new since the last release.
// By the last release we mean the last release being tested, staged or released.
func ListNewTrunkCommits() ([]*git.Commit, error) {
	// Get git config.
	config, err := git.LoadConfig()
	if err != nil {
		return nil, err
	}
	var (
		remoteName    = config.RemoteName
		trunkBranch   = config.TrunkBranchName
		releaseBranch = config.ReleaseBranchName
		stagingBranch = config.StagingBranchName
	)

	// By default, use the staging branch as the --not part.
	// In other words, list commits that are on trunk,
	// but which are not reachable from the staging branch.
	// In case the staging branch doesn't exist, take the whole trunk.
	// That probably means that no release has ever been started,
	// so the staging branch has not been created yet.
	var revRange string
	for _, branch := range [...]string{releaseBranch, stagingBranch} {
		err := git.CheckOrCreateTrackingBranch(branch, remoteName)
		// In case the branch is ok, we use it.
		if err == nil {
			revRange = fmt.Sprintf("%v..%v", branch, trunkBranch)
			break
		}
		// In case the branch does not exist, it's ok and we continue.
		if _, ok := err.(*git.ErrRefNotFound); ok {
			continue
		}
		// Otherwise we return the error since something has just exploded.
		// This can mean that the branch is not up to date, but that is an error as well.
		return nil, err
	}
	if revRange == "" {
		revRange = trunkBranch
	}

	// Get the commits in range.
	commits, err := git.ShowCommitRange(revRange)
	if err != nil {
		return nil, err
	}

	// Limit the commits by date.
	repoConfig, err := repo.LoadConfig()
	if err != nil {
		return nil, err
	}

	enabledTimestamp := repoConfig.SalsaFlowEnabledTimestamp()
	commits = git.FilterCommits(commits, func(commit *git.Commit) bool {
		return commit.AuthorDate.After(enabledTimestamp)
	})

	return commits, nil
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
