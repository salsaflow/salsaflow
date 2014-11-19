package changes

import (
	"fmt"
	"io"
	"text/tabwriter"
)

func DumpStoryChanges(groups []*StoryChangeGroup, w io.Writer, porcelain bool) error {
	// Dump the change details into the console.
	tw := tabwriter.NewWriter(w, 0, 8, 2, '\t', 0)

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
		storyId := group.StoryId

		for _, change := range group.Changes {
			changeId := change.ChangeId

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
