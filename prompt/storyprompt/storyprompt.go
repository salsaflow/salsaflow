package storyprompt

import (
	// Stdlib
	"fmt"
	"io"
	"os"
	"text/tabwriter"

	// Internal
	"github.com/salsaflow/salsaflow/errs"
	"github.com/salsaflow/salsaflow/modules/common"
	"github.com/salsaflow/salsaflow/prompt"
)

// maxStoryTitleColumnWidth specifies the width of the story title column for story listing.
// The story title is truncated to this width in case it is too long.
const maxStoryTitleColumnWidth = 80

func PromptStory(msg string, stories []common.Story) (common.Story, error) {
	return promptForStory(msg, stories, false)
}

func PromptStoryAllowNone(msg string, stories []common.Story) (common.Story, error) {
	return promptForStory(msg, stories, true)
}

func promptForStory(msg string, stories []common.Story, allowNone bool) (common.Story, error) {
	var task = "Prompt the user to select a story"

	// Make sure there are actually some stories to be printed.
	if len(stories) == 0 && !allowNone {
		return nil, prompt.ErrNoStories
	}

	// Print the intro message.
	fmt.Println(msg)
	fmt.Println()

	// Present the stories to the user.
	if err := ListStories(stories, os.Stdout); err != nil {
		return nil, err
	}
	fmt.Println()

	// Prompt the user to select a story to assign the commit with.
	var (
		index int
		err   error
	)
	if allowNone {
		index, err = prompt.PromptIndex(`Choose a story by inserting its index.
You can also insert '0' not to choose any story
or you can press Enter to abort: `, 0, len(stories))
	} else {
		index, err = prompt.PromptIndex(
			"Choose a story by inserting its index. Or just press Enter to abort: ", 1, len(stories))
	}
	if err != nil {
		if err == prompt.ErrCanceled {
			return nil, prompt.ErrCanceled
		}
		return nil, errs.NewError(task, err)
	}
	// No need to check allowNone since index can be 0 only if allowNone.
	if index == 0 {
		return nil, nil
	}
	return stories[index-1], nil
}

type writeError struct {
	err error
}

func ListStories(stories []common.Story, w io.Writer) (err error) {
	must := func(n int, err error) {
		if err != nil {
			panic(&writeError{err})
		}
	}

	defer func() {
		if r := recover(); r != nil {
			if we, ok := r.(*writeError); ok {
				err = we.err
			} else {
				panic(r)
			}
		}
	}()

	tw := tabwriter.NewWriter(w, 0, 8, 4, '\t', 0)
	must(io.WriteString(tw, "  Index\tStory ID\tStory Title\n"))
	must(io.WriteString(tw, "  =====\t========\t===========\n"))
	for i, story := range stories {
		must(fmt.Fprintf(
			tw, "  %v\t%v\t%v\n", i+1, story.ReadableId(),
			prompt.Shorten(story.Title(), maxStoryTitleColumnWidth)))
	}
	must(0, tw.Flush())

	return nil
}
