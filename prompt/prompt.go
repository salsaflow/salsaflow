package prompt

import (
	// Stdlib
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"

	// Internal
	"github.com/salsita/salsaflow/errs"
	"github.com/salsita/salsaflow/modules/common"
)

// maxStoryTitleColumnWidth specifies the width of the story title column for story listing.
// The story title is truncated to this width in case it is too long.
const maxStoryTitleColumnWidth = 80

type InvalidInputError struct {
	input string
}

func (i *InvalidInputError) Error() string {
	return "Invalid input: " + i.input
}

type OutOfBoundsError struct {
	input string
}

func (i *OutOfBoundsError) Error() string {
	return "Index out of bounds: " + i.input
}

func Confirm(question string) (bool, error) {
	printQuestion := func() {
		fmt.Print(question)
		fmt.Print(" [y/N]: ")
	}
	printQuestion()

	var line string
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line = strings.ToLower(scanner.Text())
		switch line {
		case "":
			line = "n"
		case "y":
		case "n":
		default:
			printQuestion()
			continue
		}
		break
	}
	if err := scanner.Err(); err != nil {
		return false, err
	}

	return line == "y", nil
}

func Prompt(msg string) (string, error) {
	fmt.Print(msg)
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	if err := scanner.Err(); err != nil {
		return "", err
	}
	return scanner.Text(), nil
}

func PromptIndex(msg string, min, max int) (int, error) {
	line, err := Prompt(msg)
	if err != nil {
		return 0, err
	}
	if line == "" {
		return 0, ErrCanceled
	}

	index, err := strconv.Atoi(line)
	if err != nil {
		return 0, &InvalidInputError{line}
	}

	if index < min || index > max {
		return 0, &OutOfBoundsError{line}
	}

	return index, nil
}

func PromptStory(msg string, stories []common.Story) (common.Story, error) {
	var task = "Prompt the user to select a story"

	// Make sure there are actually some stories to be printed.
	if len(stories) == 0 {
		return nil, errs.NewError(task, errors.New("no stories to choose from"),
			bytes.NewBufferString(`
There are no stories that the unassigned commits can be assigned to.
In other words, there are no stories in the right state for that.

`))
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
	index, err := PromptIndex(
		"Choose a story by inserting its index. Just press Enter to abort: ", 0, len(stories)-1)
	if err != nil {
		if err == ErrCanceled {
			return nil, ErrCanceled
		}
		return nil, errs.NewError(task, err, nil)
	}
	return stories[index], nil
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
			tw, "  %v\t%v\t%v\n", i, story.ReadableId(), formatStoryTitle(story.Title())))
	}
	must(0, tw.Flush())

	return nil
}

func formatStoryTitle(title string) string {
	if len(title) < maxStoryTitleColumnWidth {
		return title
	}

	// maxStoryTitleColumnWidth incorporates the trailing " ...",
	// so that is why we subtract len(" ...") when truncating.
	truncatedTitle := title[:maxStoryTitleColumnWidth-4]
	if title[maxStoryTitleColumnWidth-4] != ' ' {
		if i := strings.LastIndex(truncatedTitle, " "); i != -1 {
			truncatedTitle = truncatedTitle[:i]
		}
	}
	return truncatedTitle + " ..."
}
