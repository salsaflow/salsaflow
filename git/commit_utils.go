package git

import (
	"fmt"
)

const StoryIdUnassignedTagValue = "unassigned"

func GrepCommitsCaseInsensitive(filter string, args ...string) ([]*Commit, error) {
	argsList := make([]string, 6, 6+len(args))
	argsList[0] = "--source"
	argsList[1] = "--abbrev-commit"
	argsList[2] = "--pretty=fuller"
	argsList[3] = "--extended-regexp"
	argsList[4] = "--regexp-ignore-case"
	argsList[5] = "--grep=" + filter
	argsList = append(argsList, args...)

	stdout, err := RunCommand("log", argsList...)
	if err != nil {
		return nil, err
	}

	return ParseCommits(stdout.Bytes())
}

// ShowCommit returns the commit object representing the given revision.
func ShowCommit(revision string) ([]*Commit, error) {
	args := []string{
		"log",
		"--source",
		"--abbrev-commit",
		"--pretty=fuller",
		"--max-count=1",
		revision,
	}
	stdout, err := Run(args...)
	if err != nil {
		return nil, err
	}

	return ParseCommits(stdout.Bytes())
}

// ShowCommitRange returns the list of commits specified by the given Git revision range.
func ShowCommitRange(revisionRange string) ([]*Commit, error) {
	args := []string{
		"log",
		"--source",
		"--abbrev-commit",
		"--pretty=fuller",
		revisionRange,
	}
	stdout, err := Run(args...)
	if err != nil {
		return nil, err
	}

	return ParseCommits(stdout.Bytes())
}

type CommitFilterFunc func(*Commit) bool

// FilterCommits can be used to filter commits by the given filter function.
func FilterCommits(commits []*Commit, filterFunc CommitFilterFunc) []*Commit {
	cs := make([]*Commit, 0, len(commits))
	for _, commit := range commits {
		if filterFunc(commit) {
			cs = append(cs, commit)
		}
	}
	return cs
}

// FixCommitSources can be used to set the Source field for the given commits
// to the right value in respect to the branching model.
//
// We need this function becase using git log --all --source actually
// does not always yield the source values we want. It can set the source
// for the commits on the trunk branch and on the release branch to
// one of the feature branches that are branched off the given branch.
// That is not what we want, we want to have the Source field set to the
// trunk branch or the release branch in case the commit is reachable from
// one of these branches.
//
// More precisely, the Source field is set in the following way:
//
//    commits on trunk                   -> refs/heads/trunk
//    commits on trunk..origin/trunk     -> refs/remotes/origin/trunk
//    commits on trunk..release          -> refs/heads/release
//    commits on release..origin/release -> refs/remotes/origin/release
//
func FixCommitSources(commits []*Commit) error {
	config, err := LoadConfig()
	if err != nil {
		return err
	}
	var (
		remoteName    = config.RemoteName()
		trunkBranch   = config.TrunkBranchName()
		releaseBranch = config.ReleaseBranchName()
	)

	// Get the trunk commits.
	trunkCommits, err := ShowCommitRange(trunkBranch)
	if err != nil {
		return err
	}

	remoteTrunkCommits, err := ShowCommitRange(
		fmt.Sprintf("%v..%v@{upstream}", trunkBranch, trunkBranch))
	if err != nil {
		return err
	}

	// Check whether the release branch exists.
	releaseExists := true
	if err := CheckOrCreateTrackingBranch(releaseBranch, remoteName); err != nil {
		// No need to do a rollback.
		// In the worst case there will be a new local tracking branch.
		if _, ok := err.(*ErrRefNotFound); ok {
			releaseExists = false
		} else if _, ok := err.(*ErrRefNotInSync); !ok {
			// Return the err. But we don't care when the local branch is not in sync.
			return err
		}
	}

	// Get the release commits in case the release branch exists.
	var (
		releaseCommits       []*Commit
		remoteReleaseCommits []*Commit
	)
	if releaseExists {
		// Collect the commits.
		var err error
		releaseCommits, err = ShowCommitRange(
			fmt.Sprintf("%v..%v", trunkBranch, releaseBranch))
		if err != nil {
			return err
		}

		remoteReleaseCommits, err = ShowCommitRange(
			fmt.Sprintf("%v..%v@{upstream}", releaseBranch, releaseBranch))
		if err != nil {
			return err
		}
	}

	// Collect the right commit sources.
	sourceMap := make(map[string]string,
		len(trunkCommits)+len(remoteTrunkCommits)+len(releaseCommits)+len(remoteReleaseCommits))

	src := fmt.Sprintf("refs/heads/%v", trunkBranch)
	for _, commit := range trunkCommits {
		sourceMap[commit.SHA] = src
	}

	src = fmt.Sprintf("refs/remotes/%v/%v", remoteName, trunkBranch)
	for _, commit := range remoteTrunkCommits {
		sourceMap[commit.SHA] = src
	}

	if releaseExists {
		src = fmt.Sprintf("refs/heads/%v", releaseBranch)
		for _, commit := range releaseCommits {
			sourceMap[commit.SHA] = src
		}

		src = fmt.Sprintf("refs/remotes/%v/%v", remoteName, releaseBranch)
		for _, commit := range remoteReleaseCommits {
			sourceMap[commit.SHA] = src
		}
	}

	// Fix the commit sources.
	for _, commit := range commits {
		if src, ok := sourceMap[commit.SHA]; ok {
			commit.Source = src
		}
	}

	return nil
}
