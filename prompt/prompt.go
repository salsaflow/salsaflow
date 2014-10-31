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
	"github.com/salsita/salsaflow/errs"
	"github.com/salsita/salsaflow/modules/common"

	// Other
	"gopkg.in/salsita/go-pivotaltracker.v0/v5/pivotal"
)

var ErrCanceled = errors.New("operation canceled")

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
	// Print the into message.
	fmt.Println(msg)
	fmt.Println()

	// Present the stories to the user.
	tw := tabwriter.NewWriter(os.Stdout, 0, 8, 4, '\t', 0)
	io.WriteString(tw, "  Index\tStory ID\tStory Title\n")
	io.WriteString(tw, "  =====\t========\t===========\n")
	for i, story := range stories {
		fmt.Fprintf(tw, "  %v\t%v\t%v\n", i, story.ReadableId(), story.Title())
	}
	io.WriteString(tw, "\n")
	tw.Flush()

	// Prompt the user to select a story to assign the commit with.
	task := "Prompt the user to select a story"
	index, err := PromptIndex("Choose a story by inserting its index: ", 0, len(stories)-1)
	if err != nil {
		if err == ErrCanceled {
			return nil, ErrCanceled
		}
		return nil, errs.NewError(task, err, nil)
	}
	return stories[index], nil
}

func ConfirmStories(headerLine string, stories []*pivotal.Story) (bool, error) {
	printStoriesConfirmationDialog(headerLine, stories)

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
			printStoriesConfirmationDialog(headerLine, stories)
			continue
		}
		break
	}
	if err := scanner.Err(); err != nil {
		return false, err
	}

	return line == "y", nil
}

func printStoriesConfirmationDialog(headerLine string, stories []*pivotal.Story) {
	tw := tabwriter.NewWriter(os.Stdout, 0, 8, 2, '\t', 0)

	io.WriteString(tw, "\n")
	io.WriteString(tw, headerLine)
	io.WriteString(tw, "\n\n")
	io.WriteString(tw, "Story Name\tStory URL\n")
	io.WriteString(tw, "==========\t=========\n")

	for _, story := range stories {
		fmt.Fprintf(tw, "%v\t%v\n", story.Name, story.URL)
	}

	io.WriteString(tw, "\nDo you want to proceed? [y/N]:")
	tw.Flush()
}
