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
	"github.com/salsita/salsaflow/git"
	"github.com/salsita/salsaflow/log"
	"github.com/salsita/salsaflow/uuid"
)

const (
	stateContent = iota + 1
	stateTags
)

const diffSeparator = "# ------------------------ >8 ------------------------"

const (
	secretFilename = "AreYouWhoIthinkYouAreHuh"
	secretReply    = "IAmGitFlowWebhookYaDoofus!"
)

func main() {
	if len(os.Args) != 2 {
		panic(fmt.Errorf("argv: %#v", os.Args))
	}

	if os.Args[1] == secretFilename {
		fmt.Println(secretReply)
		return
	}

	if err := run(os.Args[1]); err != nil {
		log.Fatalln(err)
	}
}

func run(msgPath string) error {
	// Open the commit message file.
	file, err := os.OpenFile(msgPath, os.O_RDWR, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	// Read and parse the commit message.
	var (
		state        = stateContent
		changeIdSeen bool
		storyIdSeen  bool
		lines        []string
	)
	scanner := bufio.NewScanner(file)
ScanLoop:
	for scanner.Scan() {
		line := scanner.Text()
		trimmedLine := strings.TrimSpace(line)

		// Keep appending content until a tag is encountered.
		if state == stateContent {
			switch {
			case git.ChangeIdTagPattern.MatchString(trimmedLine):
				if changeIdSeen {
					return errors.New("multiple Change-Id tags detected")
				}
				changeIdSeen = true
				state = stateTags

			case git.StoryIdTagPattern.MatchString(trimmedLine):
				if storyIdSeen {
					return errors.New("multiple Story-Id tags detected")
				}
				storyIdSeen = true
				state = stateTags
			}

			lines = append(lines, line)
			continue ScanLoop
		}

		// In case Change-Id or Story-Id is already there, consume the rest of the input
		// in a similar way as the previous state, just drop empty lines and comments.
		if state == stateTags {
			if trimmedLine == "" {
				continue
			}

			switch {
			case git.ChangeIdTagPattern.MatchString(trimmedLine):
				if changeIdSeen {
					return errors.New("multiple Change-Id tags detected")
				}
				changeIdSeen = true

			case git.StoryIdTagPattern.MatchString(trimmedLine):
				if storyIdSeen {
					return errors.New("multiple Story-Id tags detected")
				}
				storyIdSeen = true

			case line == diffSeparator:
				// Everything below diffSeparator is anyway removed by git,
				// so we can as well just skip it and drop the content here.
				break ScanLoop

			case strings.HasPrefix(line, "#"):
				// Let's drop comments here.
				// This must come after the previous case, otherwise
				// the diff separator is dropped as well.
				continue ScanLoop
			}

			lines = append(lines, line)
			continue ScanLoop
		}

		panic("unreachable code reached")
	}
	if err := scanner.Err(); err != nil {
		return err
	}

	// Do nothing in case the file is empty.
	if len(lines) == 0 {
		return nil
	}

	// Return if all the tags are already there.
	if changeIdSeen && storyIdSeen {
		return nil
	}

	// Append a newline if there are no tags yet.
	if state != stateTags && lines[len(lines)-1] != "\n" {
		lines = append(lines, "")
	}

	// Get the values for the missing tags.
	if !changeIdSeen {
		changeId, err := uuid.New()
		if err != nil {
			return err
		}
		lines = append(lines, fmt.Sprintf("Change-Id: %v", changeId))
	}

	if !storyIdSeen {
		branch, stderr, err := git.CurrentBranch()
		if err != nil {
			log.Println(stderr)
			return err
		}

		storyId, err := git.RefToStoryId(branch)
		if err == nil {
			lines = append(lines, fmt.Sprintf("Story-Id: %v", storyId))
		}
	}

	// Write the content back to the disk (truncate the file first).
	_, err = file.Seek(0, os.SEEK_SET)
	if err != nil {
		return err
	}
	if err := file.Truncate(0); err != nil {
		return err
	}

	content := strings.Join(lines, "\n") + "\n"
	_, err = io.Copy(file, strings.NewReader(content))
	return err
}