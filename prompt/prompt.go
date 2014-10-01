package prompt

import (
	// Stdlib
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"

	// Other
	"gopkg.in/salsita/go-pivotaltracker.v0/v5/pivotal"
)

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

func PromptIndex(msg string, min, max int) (int, error) {
	line, err := Prompt(msg)
	if err != nil {
		return -1, err
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

func Prompt(msg string) (string, error) {
	var line string

	fmt.Print(msg)

	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	line = strings.ToLower(scanner.Text())
	if err := scanner.Err(); err != nil {
		return "", err
	}
	return line, nil
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

	io.WriteString(tw, "\n")
	tw.Flush()

	fmt.Print("Do you want to proceed? [y/N]: ")
}
