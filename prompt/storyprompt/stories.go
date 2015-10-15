package storyprompt

import (
	// Stdlib
	"fmt"
	"io"
	"text/tabwriter"

	// Internal
	"github.com/salsaflow/salsaflow/modules/common"
	"github.com/salsaflow/salsaflow/prompt"
)

// maxStoryTitleColumnWidth specifies the width of the story title column for story listing.
// The story title is truncated to this width in case it is too long.
const maxStoryTitleColumnWidth = 80

func ListStories(stories []common.Story, w io.Writer) error {
	tw := tabwriter.NewWriter(w, 0, 8, 4, '\t', 0)

	var err error
	write := func(format string, v ...interface{}) {
		if err != nil {
			return
		}
		if _, ex := fmt.Fprintf(tw, format, v...); ex != nil {
			err = ex
		}
	}

	write("  Index\tStory ID\tStory Title\n")
	write("  =====\t========\t===========\n")
	for i, story := range stories {
		write("  %v\t%v\t%v\n", i+1, story.ReadableId(),
			prompt.Shorten(story.Title(), maxStoryTitleColumnWidth))
	}
	if err != nil {
		return err
	}
	return tw.Flush()
}
