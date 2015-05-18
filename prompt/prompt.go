package prompt

import (
	// Stdlib
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"

	// Internal
	"github.com/salsaflow/salsaflow/errs"
	"github.com/salsaflow/salsaflow/modules/common"
)

// maxStoryTitleColumnWidth specifies the width of the story title column for story listing.
// The story title is truncated to this width in case it is too long.
const maxStoryTitleColumnWidth = 80

var ErrNoStories = errors.New("no stories to choose from")

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
	// Opening the console for O_RDWR doesn't work on Windows,
	// hence we one the device twice with a different flag set.
	// This works everywhere.
	task := "Open console for reading"
	stdin, err := OpenConsole(os.O_RDONLY)
	if err != nil {
		return false, errs.NewError(task, err, nil)
	}
	defer stdin.Close()

	task = "Open console for writing"
	stdout, err := OpenConsole(os.O_WRONLY)
	if err != nil {
		return false, errs.NewError(task, err, nil)
	}
	defer stdout.Close()

	printQuestion := func() {
		fmt.Fprint(stdout, question)
		fmt.Fprint(stdout, " [y/N]: ")
	}
	printQuestion()

	var line string
	scanner := bufio.NewScanner(stdin)
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
	stdin, err := OpenConsole(os.O_RDONLY)
	if err != nil {
		return "", err
	}
	defer stdin.Close()

	fmt.Print(msg)
	scanner := bufio.NewScanner(stdin)
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
	return promptForStory(msg, stories, false)
}

func PromptStoryAllowNone(msg string, stories []common.Story) (common.Story, error) {
	return promptForStory(msg, stories, true)
}

func promptForStory(msg string, stories []common.Story, allowNone bool) (common.Story, error) {
	var task = "Prompt the user to select a story"

	// Make sure there are actually some stories to be printed.
	if len(stories) == 0 && !allowNone {
		return nil, ErrNoStories
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
		index, err = PromptIndex(`Choose a story by inserting its index.
You can also insert '0' not to choose any story
or you can press Enter to abort: `, 0, len(stories))
	} else {
		index, err = PromptIndex(
			"Choose a story by inserting its index. Or just press Enter to abort: ", 1, len(stories))
	}
	if err != nil {
		if err == ErrCanceled {
			return nil, ErrCanceled
		}
		return nil, errs.NewError(task, err, nil)
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
			Shorten(story.Title(), maxStoryTitleColumnWidth)))
	}
	must(0, tw.Flush())

	return nil
}

func OpenConsole(flag int) (io.ReadWriteCloser, error) {
	return os.OpenFile(ConsoleDevice, flag, 0600)
}
