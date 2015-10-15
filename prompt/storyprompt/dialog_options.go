package storyprompt

import (
	// Stdlib
	"fmt"
	"regexp"
	"strconv"

	// Internal
	"github.com/salsaflow/salsaflow/modules/common"
)

// NewIndexOption returns an option that can be used to choose a story by its index.
func NewIndexOption() *DialogOption {
	return &DialogOption{
		Description: []string{
			"Choose a story by inserting the associated index",
		},
		IsActive: func(stories []common.Story, depth int) bool {
			return len(stories) != 0
		},
		MatchesInput: func(input string, stories []common.Story) bool {
			index, err := strconv.Atoi(input)
			if err != nil {
				return false
			}
			return 1 <= index && index <= len(stories)
		},
		SelectStory: func(
			input string,
			stories []common.Story,
			currentDialog *Dialog,
		) (common.Story, error) {

			index, _ := strconv.Atoi(input)
			return stories[index-1], nil
		},
	}
}

// NewFilterOption returns an option that can be used to filter stories
// by matching the title against the given regexp.
func NewFilterOption() *DialogOption {
	return &DialogOption{
		Description: []string{
			"Insert a regular expression to filter the story list.",
			"(case insensitive match against the story title)",
		},
		IsActive: func(stories []common.Story, depth int) bool {
			return len(stories) != 0
		},
		MatchesInput: func(input string, stories []common.Story) bool {
			return true
		},
		SelectStory: func(
			input string,
			stories []common.Story,
			currentDialog *Dialog,
		) (common.Story, error) {

			fmt.Println()
			fmt.Printf("Using '%v' to filter stories ...\n", input)
			fmt.Println()

			filter, err := regexp.Compile("(?i)" + input)
			if err != nil {
				return nil, err
			}
			filteredStories := common.FilterStories(stories, func(story common.Story) bool {
				return filter.MatchString(story.Title())
			})

			subdialog := currentDialog.NewSubdialog()
			subdialog.opts = currentDialog.opts
			return subdialog.Run(filteredStories)
		},
	}
}

// NewReturnOrAbortOptions returns a set of options that handle
//
//   - press Enter -> return one level up
//   - insert 'q'  -> abort the dialog
func NewReturnOrAbortOptions() []*DialogOption {
	return []*DialogOption{
		&DialogOption{
			Description: []string{
				"Press Enter to return to the previous dialog.",
			},
			IsActive: func(stories []common.Story, depth int) bool {
				return depth != 1
			},
			MatchesInput: func(input string, stories []common.Story) bool {
				return input == ""
			},
			SelectStory: func(
				input string,
				stories []common.Story,
				currentDialog *Dialog,
			) (common.Story, error) {

				fmt.Println()
				fmt.Println("Going back to the previous dialog ...")
				return nil, ErrReturn
			},
		},
		&DialogOption{
			Description: []string{
				"Insert 'q' to abort the dialog.",
			},
			IsActive: func(stories []common.Story, depth int) bool {
				return depth != 1
			},
			MatchesInput: func(input string, stories []common.Story) bool {
				return input == "q"
			},
			SelectStory: func(
				input string,
				stories []common.Story,
				currentDialog *Dialog,
			) (common.Story, error) {

				fmt.Println()
				fmt.Println("Aborting the dialog ...")
				return nil, ErrAbort
			},
		},
		&DialogOption{
			Description: []string{
				"Press Enter or insert 'q' to abort the dialog.",
			},
			IsActive: func(stories []common.Story, depth int) bool {
				return depth == 1
			},
			MatchesInput: func(input string, stories []common.Story) bool {
				return input == "" || input == "q"
			},
			SelectStory: func(
				input string,
				stories []common.Story,
				currentDialog *Dialog,
			) (common.Story, error) {

				fmt.Println()
				fmt.Println("Aborting the dialog ...")
				return nil, ErrAbort
			},
		},
	}
}
