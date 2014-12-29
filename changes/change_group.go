package changes

import (
	// Stdlib
	"bytes"
	"fmt"
	"io"
	"sort"
	"text/tabwriter"

	// Internal
	"github.com/salsaflow/salsaflow/git"
	"github.com/salsaflow/salsaflow/modules/common"
)

type StoryChangeGroup struct {
	StoryIdTag string
	Changes    []*Change
}

func GroupChangesByStoryId(changes []*Change) []*StoryChangeGroup {
	// Group the changes by story ID.
	groupMap := make(map[string]*StoryChangeGroup, len(changes))

	for _, change := range changes {
		group, ok := groupMap[change.StoryIdTag]
		if !ok {
			group = &StoryChangeGroup{
				StoryIdTag: change.StoryIdTag,
				Changes:    make([]*Change, 0, 1),
			}
			groupMap[change.StoryIdTag] = group
		}
		group.Changes = append(group.Changes, change)
	}

	// Convert the story changes map into a slice and return.
	groups := make([]*StoryChangeGroup, 0, len(groupMap))

	for _, group := range groupMap {
		groups = append(groups, group)
	}
	return groups
}

// StoryChanges returns the list of changes grouped by Story-Id.
func StoryChanges(stories []common.Story) ([]*StoryChangeGroup, error) {
	// Prepare the regexp to use to select commits by commit messages.
	// This regexp is ORing the chosen Story-Id tag values.
	var grepFlag bytes.Buffer
	fmt.Fprintf(&grepFlag, "^Story-Id: (%v", stories[0].Tag())
	for _, story := range stories[1:] {
		fmt.Fprintf(&grepFlag, "|%v", story.Tag())
	}
	fmt.Fprint(&grepFlag, ")$")

	// Get the relevant commits.
	commits, err := git.GrepCommitsCaseInsensitive(grepFlag.String(), "--all")
	if err != nil {
		return nil, err
	}
	if len(commits) == 0 {
		return nil, nil
	}

	// Split by Change-Id.
	changeGroups := GroupCommitsByChangeId(commits)

	// In case there are no changes left, we are done.
	if len(changeGroups) == 0 {
		return nil, nil
	}

	// Fix the commit sources.
	if err := git.FixCommitSources(commits); err != nil {
		return nil, err
	}

	// Return the changes grouped by story ID.
	return GroupChangesByStoryId(changeGroups), nil
}

func SortStoryChanges(groups []*StoryChangeGroup, stories []common.Story) []*StoryChangeGroup {
	// Create a map to be able to find stories by ReadableId quickly.
	storyMap := make(map[string]common.Story, len(stories))
	for _, story := range stories {
		storyMap[story.Tag()] = story
	}

	// Wrap the change groups to be able to sort them.
	wrappers := make([]*sortWrapper, 0, len(groups))
	for _, group := range groups {
		story := storyMap[group.StoryIdTag]
		wrappers = append(wrappers, &sortWrapper{group, story})
	}

	// Sort the change groups.
	sort.Sort(sort.Reverse(sortWrapperSlice(wrappers)))

	// Insert the results into the original slice.
	sorted := make([]*StoryChangeGroup, len(wrappers))
	for i, wrap := range wrappers {
		sorted[i] = wrap.group
	}
	return sorted
}

type sortWrapper struct {
	group *StoryChangeGroup
	story common.Story
}

// sortWrapperSlice implements sort.Interface, hence it can be sorted.
type sortWrapperSlice []*sortWrapper

func (slice sortWrapperSlice) Len() int {
	return len(slice)
}

func (slice sortWrapperSlice) Less(i, j int) bool {
	return slice[i].story.LessThan(slice[j].story)
}

func (slice sortWrapperSlice) Swap(i, j int) {
	slice[i], slice[j] = slice[j], slice[i]
}

// DumpStoryChanges writes a nicely formatted output to the io.Writer passed in.
//
// In case the porcelain argument is true, the output is printed in a more machine-friendly way.
func DumpStoryChanges(
	writer io.Writer,
	groups []*StoryChangeGroup,
	tracker common.IssueTracker,
	porcelain bool,
) error {

	tw := tabwriter.NewWriter(writer, 0, 8, 2, '\t', 0)

	if !porcelain {
		_, err := io.WriteString(tw, "Story\tChange\tCommit SHA\tCommit Source\tCommit Title\n")
		if err != nil {
			return err
		}
		_, err = io.WriteString(tw, "=====\t======\t==========\t=============\t============\n")
		if err != nil {
			return err
		}
	}
	for _, group := range groups {
		storyId, err := tracker.StoryTagToReadableStoryId(group.StoryIdTag)
		if err != nil {
			return err
		}

		for _, change := range group.Changes {
			changeId := change.ChangeIdTag

			// Print the first line.
			commit := change.Commits[0]
			_, err := fmt.Fprintf(tw, "%v\t%v\t%v\t%v\t%v\n",
				storyId, changeId, commit.SHA, commit.Source, commit.MessageTitle)
			if err != nil {
				return err
			}

			// Make some of the columns empty in case we are not porcelain.
			if !porcelain {
				storyId = ""
				changeId = ""
			}

			// Print the rest with the chosen columns being empty.
			for _, commit := range change.Commits[1:] {
				_, err := fmt.Fprintf(tw, "%v\t%v\t%v\t%v\t%v\n",
					storyId, changeId, commit.SHA, commit.Source, commit.MessageTitle)
				if err != nil {
					return err
				}
			}
		}
	}

	return tw.Flush()
}
