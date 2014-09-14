package changes

import (
	// Stdlib
	"regexp"
	"sort"

	// Internal
	"github.com/tchap/git-trunk/git"
)

type Change struct {
	StoryId  int
	ChangeId string
	Commits  []*git.Commit
}

func GroupCommitsByChangeId(commits []*git.Commit) []*Change {
	changes := make([]*Change, 0)

CommitsLoop:
	for _, commit := range commits {
		// Try to add the commit to one of the existing changes.
		for _, change := range changes {
			if change.ChangeId == commit.ChangeId {
				change.addCommit(commit)
				continue CommitsLoop
			}
		}
		// Otherwise create a new change and append it.
		changes = append(changes, &Change{
			StoryId:  commit.StoryId,
			ChangeId: commit.ChangeId,
			Commits:  []*git.Commit{commit},
		})
	}

	// Sort according to the initial commit date and return.
	sort.Sort(changeList(changes))
	return changes
}

func (change *Change) addCommit(commit *git.Commit) {
	// Insert into the sorted list of commits.
	var (
		i     int
		begin int
		end   int = len(change.Commits)
	)
	for {
		// Insert the commit in case we found the right place.
		if begin == end {
			sorted := make([]*git.Commit, end, len(change.Commits)+1)
			copy(sorted, change.Commits[:end])
			sorted = append(sorted, commit)
			sorted = append(sorted, change.Commits[end:]...)
			change.Commits = sorted
			return
		}

		// Try the middle of the [begin, end] commit sequence.
		i = begin + (end-begin)/2
		if commit.CommitDate.Before(change.Commits[i].CommitDate) {
			// In case the current commit happened before the pivot commit,
			// move the interval end to the current position and try again.
			end = i
		} else {
			// Otherwise move the interval beginning and try again.
			begin = i + 1
		}
	}

}

func (change *Change) initialCommit() *git.Commit {
	// The commits are sorted according to CommitDate.
	return change.Commits[0]
}

// Implementation of sort.Interface for []*Change
// The sorting happens according to the initial commit, it is based on CommitDate.

type changeList []*Change

func (list changeList) Len() int {
	return len(list)
}

func (list changeList) Less(i, j int) bool {
	return list[i].initialCommit().CommitDate.Before(list[j].initialCommit().CommitDate)
}

func (list changeList) Swap(i, j int) {
	tmp := list[i]
	list[i] = list[j]
	list[j] = tmp
}

// Utility functions -----------------------------------------------------------

func FilterChangesBySource(changes []*Change, include, exclude []*regexp.Regexp) []*Change {
	filtered := make([]*Change, 0)

	for _, change := range changes {
		includeMatches := false
		if len(include) == 0 {
			includeMatches = true
		} else {
		IncludeLoop:
			for _, commit := range change.Commits {
				for _, pattern := range include {
					if pattern.MatchString(commit.Source) {
						includeMatches = true
						break IncludeLoop
					}
				}
			}
		}

		excludeMatches := false
		if len(exclude) != 0 {
		ExcludeLoop:
			for _, commit := range change.Commits {
				for _, pattern := range exclude {
					if pattern.MatchString(commit.Source) {
						excludeMatches = true
						break ExcludeLoop
					}
				}
			}
		}

		if includeMatches && !excludeMatches {
			filtered = append(filtered, change)
		}
	}

	return filtered
}
