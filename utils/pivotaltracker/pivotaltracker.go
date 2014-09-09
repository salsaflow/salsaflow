package pivotaltracker

import (
	// Stdlib
	"bytes"
	"errors"
	"fmt"

	// Internal
	"github.com/tchap/git-trunk/config"
	"github.com/tchap/git-trunk/version"

	// Other
	"gopkg.in/salsita/go-pivotaltracker.v0/v5/pivotal"
)

var (
	ErrReleaseNotDeliverable = errors.New("Pivotal Tracker: The release is not deliverable")
	ErrApiCall               = errors.New("Pivotal Tracker: API call failed")
)

func ListStories(filter string) ([]*pivotal.Story, error) {
	var (
		token     = config.PivotalTracker.ApiToken()
		projectId = config.PivotalTracker.ProjectId()
	)
	client := pivotal.NewClient(token)
	stories, _, err := client.Stories.List(projectId, filter)
	return stories, err
}

func ListReleaseCandidateStories() ([]*pivotal.Story, error) {
	filter := fmt.Sprintf(
		"type:%v,%v state:%v -label:/%v/",
		pivotal.StoryTypeFeature,
		pivotal.StoryTypeBug,
		pivotal.StoryStateFinished,
		"release-"+version.MatcherString)
	return ListStories(filter)
}

func ListReleaseStories(version string) ([]*pivotal.Story, error) {
	filter := fmt.Sprintf(
		"type:%v,%v state:%v label:%v",
		pivotal.StoryTypeFeature,
		pivotal.StoryTypeBug,
		pivotal.StoryStateFinished,
		ReleaseLabel(version),
	)
	return ListStories(filter)
}

func ReleaseDeliverable(stories []*pivotal.Story) (stderr *bytes.Buffer, err error) {
	stderr = new(bytes.Buffer)
	for _, story := range stories {
		if !StoryLabeled(story, config.PivotalTracker.ReviewedLabel()) {
			fmt.Fprintf(
				stderr,
				"\tStory %v has not been accepted by the reviewer.\n",
				story.URL)
			err = ErrReleaseNotDeliverable
		}
		if !StoryLabeled(story, config.PivotalTracker.VerifiedLabel()) {
			fmt.Fprintf(
				stderr,
				"\tStory %v has not been accepted by the QA.\n",
				story.URL)
			err = ErrReleaseNotDeliverable
		}
	}
	return
}

func SetStoriesState(stories []*pivotal.Story, state string) ([]*pivotal.Story, *bytes.Buffer, error) {
	updateRequest := &pivotal.Story{State: state}
	return updateStories(stories, func(story *pivotal.Story) *pivotal.Story {
		return updateRequest
	})
}

func AddLabel(stories []*pivotal.Story, label string) ([]*pivotal.Story, *bytes.Buffer, error) {
	return updateStories(stories, func(story *pivotal.Story) *pivotal.Story {
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
	})
}

func RemoveLabel(stories []*pivotal.Story, label string) ([]*pivotal.Story, *bytes.Buffer, error) {
	return updateStories(stories, func(story *pivotal.Story) *pivotal.Story {
		// Drop the label that matches.
		labels := make([]*pivotal.Label, 0, len(story.Labels))
		for _, l := range story.Labels {
			if l.Name != label {
				labels = append(labels, l)
			}
		}

		// Do nothing if there is no change.
		if len(labels) == len(story.Labels) {
			return nil
		}

		// Otherwise perform the update request.
		return &pivotal.Story{
			Labels: labels,
		}
	})
}

func StoryLabeled(story *pivotal.Story, label string) bool {
	for _, lab := range story.Labels {
		if lab.Name == label {
			return true
		}
	}
	return false
}

func ReleaseLabel(version string) string {
	return "release-" + version
}

type storyUpdateFunc func(story *pivotal.Story) (updateRequest *pivotal.Story)

type storyUpdateResult struct {
	story *pivotal.Story
	err   error
}

func updateStories(stories []*pivotal.Story, updateFunc storyUpdateFunc) ([]*pivotal.Story, *bytes.Buffer, error) {
	// Prepare the PT client.
	var (
		token     = config.PivotalTracker.ApiToken()
		projectId = config.PivotalTracker.ProjectId()
	)
	client := pivotal.NewClient(token)

	// Send all the request at once.
	retCh := make(chan *storyUpdateResult, len(stories))
	for _, story := range stories {
		go func(s *pivotal.Story) {
			// Get the update request.
			req := updateFunc(s)
			if req == nil {
				retCh <- &storyUpdateResult{s, nil}
				return
			}

			// Send the update request and collect the result.
			newS, _, err := client.Stories.Update(projectId, s.Id, req)
			if err != nil {
				retCh <- &storyUpdateResult{s, err}
			} else {
				retCh <- &storyUpdateResult{newS, nil}
			}
		}(story)
	}

	// Wait for the requests to return.
	var (
		ss     = make([]*pivotal.Story, 0, len(stories))
		stderr = new(bytes.Buffer)
		err    error
	)
	for i := 0; i < cap(retCh); i++ {
		ret := <-retCh
		ss = append(ss, ret.story)
		if ret.err != nil {
			fmt.Fprintln(stderr, ret.err)
			err = ErrApiCall
		}
	}

	return ss, stderr, err
}
