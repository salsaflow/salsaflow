package pivotaltracker

import (
	// Stdlib
	"bytes"
	"errors"
	"fmt"
	"io"

	// Internal
	"github.com/salsaflow/salsaflow/errs"

	// Other
	"github.com/salsita/go-pivotaltracker/v5/pivotal"
)

var (
	ErrReleaseNotDeliverable = errors.New("Pivotal Tracker: the release is not deliverable")
	ErrApiCall               = errors.New("Pivotal Tracker: API call failed")
)

func fetchMe() (*pivotal.Me, error) {
	config, err := LoadConfig()
	if err != nil {
		return nil, err
	}

	client := pivotal.NewClient(config.UserToken())
	me, _, err := client.Me.Get()
	return me, err
}

func searchStories(
	client *pivotal.Client,
	projectId int,
	format string, v ...interface{},
) ([]*pivotal.Story, error) {

	// Automatically limit the story type.
	format = fmt.Sprintf("(type:%v OR type:%v) AND %v",
		pivotal.StoryTypeFeature, pivotal.StoryTypeBug, format)

	stories, _, err := client.Stories.List(projectId, fmt.Sprintf(format, v...))
	return stories, err
}

type storyGetResult struct {
	story *pivotal.Story
	err   error
}

func listStoriesById(
	client *pivotal.Client,
	projectId int,
	ids []string,
) ([]*pivotal.Story, error) {

	var filter bytes.Buffer
	for _, id := range ids {
		if filter.Len() != 0 {
			if _, err := filter.WriteString("OR "); err != nil {
				return nil, err
			}
		}
		if _, err := filter.WriteString("id:"); err != nil {
			return nil, err
		}
		if _, err := filter.WriteString(id); err != nil {
			return nil, err
		}
	}
	return searchStories(client, projectId, filter.String())
}

func releaseDeliverable(stories []*pivotal.Story) (ok bool, details *bytes.Buffer) {
	panic("Not implemented")

	/*
			var (
				deliverable = true
				out         bytes.Buffer
			)
			tw := tabwriter.NewWriter(&out, 0, 8, 2, '\t', 0)
			io.WriteString(tw, "\n")
			io.WriteString(tw, "Story URL\tError\n")
			io.WriteString(tw, "=========\t=====\n")

		StoryLoop:
			for _, story := range stories {
				// Skip the check when the relevant label is there.
				for _, label := range config.SkipCheckLabels() {
					if storyLabeled(story, label) {
						continue StoryLoop
					}
				}

				// Otherwise make sure the story is accepted.
				if !storyLabeled(story, config.ReviewedLabel()) {
					fmt.Fprintf(tw, "%v\t%v\n", story.URL, "not accepted by the reviewer")
					deliverable = false
				}
				if !storyLabeled(story, config.VerifiedLabel()) {
					fmt.Fprintf(tw, "%v\t%v\n", story.URL, "not accepted by the QA")
					deliverable = false
				}
			}

			io.WriteString(tw, "\n")
			if !deliverable {
				tw.Flush()
				return false, &out
			}
			return true, nil
	*/
}

func addLabelFunc(label string) storyUpdateFunc {
	return func(story *pivotal.Story) *pivotal.Story {
		// Make sure the label is not already there.
		labels := story.Labels
		for _, l := range labels {
			if l.Name == label {
				return nil
			}
		}

		// Return the update request.
		return &pivotal.Story{
			Labels: append(labels, &pivotal.Label{Name: label}),
		}
	}
}

func removeLabelFunc(label string) storyUpdateFunc {
	return func(story *pivotal.Story) *pivotal.Story {
		// Make sure the label is there.
		index := -1
		labels := story.Labels
		for i, l := range labels {
			if l.Name == label {
				index = i
				break
			}
		}
		if index == -1 {
			return nil
		}

		// Return the update request.
		return &pivotal.Story{
			Labels: append(labels[:index], labels[index+1:]...),
		}
	}
}

func addLabel(
	client *pivotal.Client,
	projectId int,
	stories []*pivotal.Story,
	label string,
) ([]*pivotal.Story, error) {

	return updateStories(client, projectId, stories, addLabelFunc(label), removeLabelFunc(label))
}

func removeLabel(
	client *pivotal.Client,
	projectId int,
	stories []*pivotal.Story,
	label string,
) ([]*pivotal.Story, error) {

	return updateStories(client, projectId, stories, removeLabelFunc(label), addLabelFunc(label))
}

type storyUpdateFunc func(story *pivotal.Story) (updateRequest *pivotal.Story)

type storyUpdateResult struct {
	story *pivotal.Story
	err   error
}

func updateStories(
	client *pivotal.Client,
	projectId int,
	stories []*pivotal.Story,
	updateFunc storyUpdateFunc,
	rollbackFunc storyUpdateFunc,
) ([]*pivotal.Story, error) {

	// Send all the request at once.
	retCh := make(chan *storyUpdateResult, len(stories))
	for _, story := range stories {
		go func(story *pivotal.Story) {
			// Get the update request.
			// Returning nil means that no request is sent.
			updateRequest := updateFunc(story)
			if updateRequest == nil {
				retCh <- &storyUpdateResult{story, nil}
				return
			}

			// Send the update request and collect the result.
			updatedStory, _, err := client.Stories.Update(projectId, story.Id, updateRequest)
			if err == nil {
				// On success, return the updated story.
				retCh <- &storyUpdateResult{updatedStory, nil}
			} else {
				// On error, keep the original story, add the error.
				retCh <- &storyUpdateResult{story, err}
			}
		}(story)
	}

	// Wait for the requests to complete.
	var (
		stderr          = bytes.NewBufferString("\nUpdate Errors\n-------------\n")
		updatedStories  = make([]*pivotal.Story, 0, len(stories))
		errUpdateFailed = errors.New("failed to update some Pivotal Tracker stories")
		err             error
	)
	for _ = range stories {
		if ret := <-retCh; ret.err != nil {
			fmt.Fprintln(stderr, ret.err)
			err = errUpdateFailed
		} else {
			updatedStories = append(updatedStories, ret.story)
		}
	}
	fmt.Fprintln(stderr)

	if err != nil {
		if rollbackFunc != nil {
			// Spawn the rollback goroutines.
			// Basically the same thing as with the update requests.
			retCh := make(chan *storyUpdateResult)
			for _, story := range updatedStories {
				go func(story *pivotal.Story) {
					rollbackRequest := rollbackFunc(story)
					if rollbackRequest == nil {
						retCh <- &storyUpdateResult{story, nil}
						return
					}

					updatedStory, _, err := client.Stories.Update(
						projectId, story.Id, rollbackRequest)
					if err == nil {
						retCh <- &storyUpdateResult{updatedStory, nil}
					} else {
						retCh <- &storyUpdateResult{story, err}
					}
				}(story)
			}

			// Collect the rollback results.
			rollbackStderr := bytes.NewBufferString("\nRollback Errors\n---------------\n")
			for _ = range updatedStories {
				if ret := <-retCh; ret.err != nil {
					fmt.Fprintln(rollbackStderr, ret.err)
				}
			}

			// Append the rollback error output to the update error output.
			if _, err := io.Copy(stderr, rollbackStderr); err != nil {
				panic(err)
			}
		}

		// Return the aggregate error.
		return nil, errs.NewError("Update Pivotal Tracker stories", err, stderr)
	}
	return updatedStories, nil
}
