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
	"gopkg.in/salsita/go-pivotaltracker.v1/v5/pivotal"
)

var (
	ErrReleaseNotDeliverable = errors.New("Pivotal Tracker: the release is not deliverable")
	ErrApiCall               = errors.New("Pivotal Tracker: API call failed")
)

func addLabelFunc(label string) storyUpdateFunc {
	return func(story *pivotal.Story) *pivotal.StoryRequest {
		// Make sure the label is not already there.
		labels := story.Labels
		for _, l := range labels {
			if l.Name == label {
				return nil
			}
		}

		// Return the update request.
		ls := append(labels, &pivotal.Label{Name: label})
		return &pivotal.StoryRequest{Labels: &ls}
	}
}

func removeLabelFunc(label string) storyUpdateFunc {
	return func(story *pivotal.Story) *pivotal.StoryRequest {
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
		ls := append(labels[:index], labels[index+1:]...)
		return &pivotal.StoryRequest{Labels: &ls}
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

type storyUpdateFunc func(story *pivotal.Story) (updateRequest *pivotal.StoryRequest)

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

	// Allow just a single pending request at a time.
	// This is because PT is lame and crashes otherwise.
	semCh := make(chan struct{}, 1)
	down := func() {
		semCh <- struct{}{}
	}
	up := func() {
		<-semCh
	}

	// Send all the request at once.
	retCh := make(chan *storyUpdateResult, len(stories))
	for _, story := range stories {
		go func(story *pivotal.Story) {
			down()
			defer up()

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
					down()
					defer up()

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
			rollbackStderr := bytes.NewBufferString("Rollback Errors\n---------------\n")
			for _ = range updatedStories {
				if ret := <-retCh; ret.err != nil {
					fmt.Fprintln(rollbackStderr, ret.err)
				}
			}
			fmt.Fprintln(stderr)

			// Append the rollback error output to the update error output.
			if _, err := io.Copy(stderr, rollbackStderr); err != nil {
				panic(err)
			}
		}

		// Return the aggregate error.
		return nil, errs.NewErrorWithHint("Update Pivotal Tracker stories", err, stderr.String())
	}
	return updatedStories, nil
}
