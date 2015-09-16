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

	// Internal
	"github.com/salsaflow/salsaflow/errs"
)

var ErrNoStories = errors.New("no stories to choose from")

type InvalidInputError struct {
	Input string
}

func (i *InvalidInputError) Error() string {
	return "invalid input: " + i.Input
}

type OutOfBoundsError struct {
	Input string
}

func (i *OutOfBoundsError) Error() string {
	return "index out of bounds: " + i.Input
}

func Confirm(question string, defaultChoice bool) (bool, error) {
	// Opening the console for O_RDWR doesn't work on Windows,
	// hence we one the device twice with a different flag set.
	// This works everywhere.
	task := "Open console for reading"
	stdin, err := OpenConsole(os.O_RDONLY)
	if err != nil {
		return false, errs.NewError(task, err)
	}
	defer stdin.Close()

	task = "Open console for writing"
	stdout, err := OpenConsole(os.O_WRONLY)
	if err != nil {
		return false, errs.NewError(task, err)
	}
	defer stdout.Close()

	printQuestion := func() {
		fmt.Fprint(stdout, question)
		if defaultChoice {
			fmt.Fprint(stdout, " [Y/n]: ")
		} else {
			fmt.Fprint(stdout, " [y/N]: ")
		}
	}
	printQuestion()

	var choice bool
	scanner := bufio.NewScanner(stdin)
	for scanner.Scan() {
		switch strings.ToLower(scanner.Text()) {
		case "":
			choice = defaultChoice
		case "y":
			choice = true
		case "n":
			choice = false
		default:
			printQuestion()
			continue
		}
		break
	}
	if err := scanner.Err(); err != nil {
		return false, err
	}

	return choice, nil
}

// Prompt prints the given message and waits for user input.
// In case the input is empty, ErrCanceled is returned.
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

	input := scanner.Text()
	if input == "" {
		return "", ErrCanceled
	}
	return input, nil
}

func PromptIndex(msg string, min, max int) (int, error) {
	line, err := Prompt(msg)
	if err != nil {
		return 0, err
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

func PromptDefault(msg, defaultValue string) (string, error) {
	answer, err := Prompt(fmt.Sprintf("%v (default = %v): ", msg, defaultValue))
	if err != nil {
		if err == ErrCanceled {
			return defaultValue, nil
		}
		return "", err
	}
	return answer, nil
}

func OpenConsole(flag int) (io.ReadWriteCloser, error) {
	var (
		file io.ReadWriteCloser
		err  error
	)
	for _, deviceName := range ConsoleDevices {
		file, err = os.OpenFile(deviceName, flag, 0600)
		if err == nil {
			return file, nil
		}
	}
	return nil, err
}
