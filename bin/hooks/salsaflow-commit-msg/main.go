package main

import (
	// Stdlib
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	// Internal
	"github.com/salsaflow/salsaflow/asciiart"
	"github.com/salsaflow/salsaflow/errs"
	"github.com/salsaflow/salsaflow/git"
	"github.com/salsaflow/salsaflow/hooks"
	"github.com/salsaflow/salsaflow/uuid"
)

const diffSeparator = "# ------------------------ >8 ------------------------"

func main() {
	// Set up the identification command line flag.
	hooks.IdentifyYourself()

	// Tell the user what is happening.
	fmt.Println("---> Running the SalsaFlow commit-msg hook")

	// The hook is always invoked as `commit-msg <message-filename>`.
	if len(os.Args) != 2 {
		fmt.Fprintf(os.Stderr, "Usage: %v <message-filename>\n", os.Args[0])
		errs.Fatal(fmt.Errorf("invalid arguments: %#v\n", os.Args[1:]))
	}

	// Run the main function.
	if err := run(os.Args[1]); err != nil {
		asciiart.PrintGrimReaper("COMMIT ABORTED")
		errs.Fatal(err)
	}
}

func run(messagePath string) error {
	// Open the commit message file.
	file, err := os.OpenFile(messagePath, os.O_RDWR, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	// Read and parse the commit message.
	var (
		dropEmptyLines bool = true
		changeIdSeen   bool
		lines          []string
	)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		// Read the next line.
		var (
			line        = scanner.Text()
			trimmedLine = strings.TrimSpace(line)
		)

		// Drop leading empty lines.
		if dropEmptyLines {
			if trimmedLine == "" {
				continue
			}
			dropEmptyLines = false
		}

		// Drop the diff that can be appended do the commit message when
		// git commit -v is used. Git would drop the diff later anyway.
		if line == diffSeparator {
			break
		}

		// Drop other comments. This must happen after we check for the separator,
		// otherwise the separator is just dropped as a comment.
		if strings.HasPrefix(trimmedLine, "#") {
			continue
		}

		// Check for the Change-Id tag.
		if git.ChangeIdTagPattern.MatchString(trimmedLine) {
			if changeIdSeen {
				return errors.New("multiple Change-Id tags detected")
			}
			changeIdSeen = true
		}

		// Finally, append the line to the commit message.
		lines = append(lines, line)
	}
	if err := scanner.Err(); err != nil {
		return err
	}

	// Do nothing in case the file is empty.
	if len(lines) == 0 {
		return nil
	}

	// Do nothing in case Change-Id is already there.
	if changeIdSeen {
		return nil
	}

	// Make sure a single empty line is following the current content.
	// Do not insert an empty line in case the last line is Story-Id.
	for lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	if line := strings.ToLower(lines[len(lines)-1]); !strings.HasPrefix(line, "story-id") {
		lines = append(lines, "")
	}

	// Append the Change-Id tag.
	changeId, err := uuid.New()
	if err != nil {
		return err
	}
	lines = append(lines, fmt.Sprintf("Change-Id: %v", changeId))

	// Write the content back to the disk (truncate the file first).
	_, err = file.Seek(0, os.SEEK_SET)
	if err != nil {
		return err
	}
	if err := file.Truncate(0); err != nil {
		return err
	}

	_, err = io.Copy(file, strings.NewReader(strings.Join(lines, "\n")))
	return err
}
