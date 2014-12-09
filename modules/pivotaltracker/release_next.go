package pivotaltracker

import (
	// Stdlib
	//"errors"
	//"fmt"
	//"io"
	//"os"
	//"text/tabwriter"

	// Internal
	//"github.com/salsaflow/salsaflow/errs"
	//"github.com/salsaflow/salsaflow/log"
	"github.com/salsaflow/salsaflow/action"
	//"github.com/salsaflow/salsaflow/prompt"
	"github.com/salsaflow/salsaflow/version"

	// Other
	//"gopkg.in/salsita/go-pivotaltracker.v0/v5/pivotal"
)

type nextRelease struct{}

func newNextRelease(trunkVersion, nextTrunkVersion *version.Version) (*nextRelease, error) {
	panic("Not implemented")

	/*
		stories, err := listNextReleaseStories()
		if err != nil {
			return nil, err
		}
		return &nextRelease{ver, stories}, nil
	*/
}

func (release *nextRelease) PromptUserToConfirmStart() (bool, error) {
	panic("Not implemented")

	/*
		// Exit if there are not candidate stories.
		if len(release.stories) == 0 {
			return false, errors.New("no candidate stories found in Pivotal Tracker")
		}

		// Warn the user about the point me label.
		var (
			pmLabel   = config.PointMeLabel()
			pmStories []*pivotal.Story
		)
		for _, story := range release.stories {
			if storyLabeled(story, pmLabel) {
				pmStories = append(pmStories, story)
			}
		}
		if len(pmStories) != 0 {
			tw := tabwriter.NewWriter(os.Stdout, 0, 8, 2, '\t', 0)
			fmt.Fprintf(tw, "\nThe following stories are labeled '%v':\n\n", pmLabel)
			io.WriteString(tw, "Story Name\tStory URL\n")
			io.WriteString(tw, "========= \t=========\n")
			for _, story := range release.stories {
				name := story.Name
				if len(name) > 50 {
					name = name[:50] + " ..."
				}
				fmt.Fprintf(tw, "%v\t%v\n", name, story.URL)
			}
			io.WriteString(tw, "\n")
			tw.Flush()

			ok, err := prompt.Confirm("Are you sure you want to continue?")
			if err != nil {
				return false, err
			}
			if !ok {
				return false, nil
			}
		}

		// Prompt the user to confirm the release.
		confirmed, err := prompt.ConfirmStories(
			"The following stories will be included in the next release:",
			release.stories)
		if err != nil {
			return false, err
		}
		if !confirmed {
			// Don't print the fail message.
			return false, nil
		}
		fmt.Println()
		return true, nil
	*/
}

func (release *nextRelease) Start() (action.Action, error) {
	panic("Not implemented")

	/*
		// Add release labels to the relevant stories.
		task := "Label the stories with the release label"
		log.Run(task)
		stories, stderr, err := addLabel(release.stories, releaseLabel(release.ver))
		if err != nil {
			return nil, errs.NewError(task, err, stderr)
		}
		release.stories = stories

		// Return the rollback action, which removes the release labels that were appended.
		return common.ActionFunc(func() error {
			log.Rollback(task)
			stories, stderr, err := removeLabel(release.stories, releaseLabel(release.ver))
			if err != nil {
				return errs.NewError(task, err, stderr)
			}
			release.stories = stories
			return nil
		}), nil
	*/
}
