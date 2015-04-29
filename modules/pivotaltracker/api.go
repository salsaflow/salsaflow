package pivotaltracker

import (
	// Stdlib
	"bytes"
	"errors"
	"fmt"
	"io"
	"strconv"

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

	// Generate the query.
	query := fmt.Sprintf(format, v...)

	// Automatically limit the story type.
	query = fmt.Sprintf("(type:%v OR type:%v) AND (%v)",
		pivotal.StoryTypeFeature, pivotal.StoryTypeBug, query)

	stories, _, err := client.Stories.List(projectId, query)
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

	if len(ids) == 0 {
		return []*pivotal.Story{}, nil
	}

	var filter bytes.Buffer
	fmt.Fprintf(&filter, "id:%v", ids[0])
	for _, id := range ids[1:] {
		fmt.Fprintf(&filter, " OR id:%v", id)
	}
	return searchStories(client, projectId, filter.String())
}

func listStoriesByIdOrdered(
	client *pivotal.Client,
	projectId int,
	ids []string,
) ([]*pivotal.Story, error) {

	// Fetch the stories.
	stories, err := listStoriesById(client, projectId, ids)
	if err != nil {
		return nil, err
	}

	// Order them.
	idMap := make(map[string]*pivotal.Story, len(ids))
	for _, story := range stories {
		idMap[strconv.Itoa(story.Id)] = story
	}

	ordered := make([]*pivotal.Story, 0, len(ids))
	for _, id := range ids {
		if story, ok := idMap[id]; ok {
			ordered = append(ordered, story)
			continue
		}

		panic("unreachable code reached")
	}

	return ordered, nil
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
		return nil, errs.NewError("Update Pivotal Tracker stories", err, stderr)
	}
	return updatedStories, nil
}
