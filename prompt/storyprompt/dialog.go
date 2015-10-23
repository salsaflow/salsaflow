package storyprompt

import (
	// Stdlib
	"errors"
	"fmt"
	"os"
	"strings"

	// Internal
	"github.com/salsaflow/salsaflow/modules/common"
	"github.com/salsaflow/salsaflow/prompt"
)

var (
	// ErrContinue can be returned from SelectStory function
	// to show the same dialog again.
	ErrContinue = errors.New("continue")

	// ErrReturn can be returned from SelectStory function
	// to return to the dialog one level higher.
	ErrReturn = errors.New("return")

	// ErrAbort can be returned from SelectStory function
	// to abort the dialog with panicking using prompt.ErrCanceled.
	ErrAbort = errors.New("abort")
)

// DialogOption represents a single option the user has when presented with a dialog.
// It represents a single input choice and the associated action.
type DialogOption struct {
	// Description is the text the user is presented when the option is active.
	// The slice represents the lines of text printed to the user.
	Description []string

	// IsActive is a function that decides whether the option is active for the dialog.
	// When the option is not active, it is not presented to the user. It is handy
	// to limit the options when e.g. there are no stories and so on.
	IsActive func(stories []common.Story, depth int) bool

	// MatchesInput is a function that decides whether SelectStory should be executed.
	// It basically checks the input and when it matches, it returns true, otherwise
	// the input is passed to the next option in the chain.
	MatchesInput func(input string, stories []common.Story) bool

	// SelectStory represents the option body, the task being to pick a single story
	// from the given story list. The function can be pretty much anything
	// as long as it matches the signature. It can run a subdialog, filter stories etc.
	SelectStory func(input string, stories []common.Story, current *Dialog) (common.Story, error)
}

// Dialog represents a dialog to be used to let the user pick a single story from the list.
type Dialog struct {
	opts  []*DialogOption
	depth int
	isSub bool
}

// NewDialog creates and returns a new dialog with no options set.
func NewDialog() *Dialog {
	return &Dialog{}
}

// PushOptions can be used to add options to the option chain.
//
// The dialog will try to find the matching option based on the order
// the options are matching, so in case there is an option matching
// any input pushed as the first option, no other option body will
// ever be executed.
func (dialog *Dialog) PushOptions(opts ...*DialogOption) {
	dialog.opts = append(dialog.opts, opts...)
}

// NewSubdialog can be used to create a new dialog based on the current dialog.
// The option list is empty again, just the dialog depth is inherited.
func (dialog *Dialog) NewSubdialog() *Dialog {
	return &Dialog{
		depth: dialog.depth,
		isSub: true,
	}
}

// Run starts the dialog after the options are set. It uses the given story list
// to prompt the user for a story using the given options.
func (dialog *Dialog) Run(stories []common.Story) (common.Story, error) {
	// Return an error when no options are set.
	if len(dialog.opts) == 0 {
		return nil, errors.New("storyprompt.Dialog.Run(): no options were specified")
	}

	// Increment the dialog depth on enter.
	dialog.depth++
	// Decrement the dialog depth on return.
	defer func() {
		dialog.depth--
	}()

	// Enter the dialog loop.
DialogLoop:
	for {
		var (
			opts  = dialog.opts
			depth = dialog.depth
		)

		// Present the stories to the user.
		if err := ListStories(stories, os.Stdout); err != nil {
			return nil, err
		}

		// Print the options based on the dialog depth.
		fmt.Println()
		fmt.Println("Now you can do one of the following:")
		fmt.Println()

		// Collect the list of active options.
		activeOpts := make([]*DialogOption, 0, len(opts))
		for _, opt := range opts {
			if isActive := opt.IsActive; isActive != nil && isActive(stories, depth) {
				activeOpts = append(activeOpts, opt)
			}
		}

		// Print the description for the active options.
		for _, opt := range activeOpts {
			if desc := opt.Description; len(desc) != 0 {
				fmt.Println("  -", strings.Join(desc, "\n    "))
			}
		}
		fmt.Println()

		// Prompt the user for their choice.
		fmt.Println("Current dialog depth:", depth)
		input, err := prompt.Prompt("Choose what to do next: ")
		// We ignore prompt.ErrCanceled here and simply continue.
		// That is because an empty input is a valid input here as well.
		if err != nil && err != prompt.ErrCanceled {
			return nil, err
		}
		input = strings.TrimSpace(input)

		// Find the first matching option.
		var matchingOpt *DialogOption
		for _, opt := range activeOpts {
			if matchFunc := opt.MatchesInput; matchFunc != nil && matchFunc(input, stories) {
				matchingOpt = opt
				break
			}
		}
		// Loop again in case no match is found.
		if matchingOpt == nil {
			fmt.Println()
			fmt.Println("Error: no matching option found")
			fmt.Println()
			continue DialogLoop
		}

		// Run the selected select function.
		if selectFunc := matchingOpt.SelectStory; selectFunc != nil {
			story, err := selectFunc(input, stories, dialog)
			if err != nil {
				switch err {
				case ErrContinue:
					// Continue looping on ErrContinue.
					fmt.Println()
					continue DialogLoop
				case ErrReturn:
					// Go one dialog up by returning ErrContinue.
					// This makes the dialog loop of the parent dialog continue,
					// effectively re-printing and re-running that dialog.
					if dialog.isSub {
						return nil, ErrContinue
					}

					// In case this is a top-level dialog, abort.
					fallthrough
				case ErrAbort:
					// Panic prompt.ErrCanceled on ErrAbort, returning immediately
					// from any dialog depth.
					prompt.PanicCancel()
				}

				// In case the error is not any of the recognized control errors,
				// print the error and loop again, making the user choose again.
				fmt.Println()
				fmt.Println("Error:", err)
				fmt.Println()
				continue DialogLoop
			}
			return story, nil
		}

		// No SelectStory function specified for the matching option,
		// that is a programming error, let's just panic.
		panic(errors.New("SelectStory function not specified"))
	}
}
