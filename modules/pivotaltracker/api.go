package pivotaltracker

import (
	// Stdlib
	"bytes"
	"errors"
	"fmt"
	//"io"
	//"text/tabwriter"

	// Internal
	"github.com/salsaflow/salsaflow/errs"
	"github.com/salsaflow/salsaflow/version"

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

func searchStories(format string, v ...interface{}) ([]*pivotal.Story, error) {
	config, err := LoadConfig()
	if err != nil {
		return nil, err
	}

	client := pivotal.NewClient(config.UserToken())
	stories, _, err := client.Stories.List(config.ProjectId(), fmt.Sprintf(format, v...))
	return stories, err
}

type storyGetResult struct {
	story *pivotal.Story
	err   error
}

func listStoriesById(ids []string) ([]*pivotal.Story, error) {
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
	return searchStories(filter.String())
}

func listNextReleaseStories() ([]*pivotal.Story, error) {
	filter := fmt.Sprintf(
		"type:%v,%v state:%v -label:/%v/",
		pivotal.StoryTypeFeature,
		pivotal.StoryTypeBug,
		pivotal.StoryStateFinished,
		"release-"+version.MatcherString)
	return searchStories(filter)
}

func listStoriesByRelease(release *version.Version) ([]*pivotal.Story, error) {
	filter := fmt.Sprintf(
		"type:%v,%v state:%v label:%v",
		pivotal.StoryTypeFeature,
		pivotal.StoryTypeBug,
		pivotal.StoryStateFinished,
		releaseLabel(release),
	)
	return searchStories(filter)
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

func setStoriesState(stories []*pivotal.Story, state string) ([]*pivotal.Story, error) {
	updateRequest := &pivotal.Story{State: state}
	return updateStories(stories, func(story *pivotal.Story) *pivotal.Story {
		return updateRequest
	})
}

func addLabel(stories []*pivotal.Story, label string) ([]*pivotal.Story, error) {
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

func removeLabel(stories []*pivotal.Story, label string) ([]*pivotal.Story, error) {
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

func storyLabeled(story *pivotal.Story, label string) bool {
	for _, lab := range story.Labels {
		if lab.Name == label {
			return true
		}
	}
	return false
}

func releaseLabel(release *version.Version) string {
	return "release-" + release.String()
}

type storyUpdateFunc func(story *pivotal.Story) (updateRequest *pivotal.Story)

type storyUpdateResult struct {
	story *pivotal.Story
	err   error
}

func updateStories(stories []*pivotal.Story, updateFunc storyUpdateFunc) ([]*pivotal.Story, error) {
	config, err := LoadConfig()
	if err != nil {
		return nil, err
	}

	// Prepare the PT client.
	var (
		token     = config.UserToken()
		projectId = config.ProjectId()
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
	)
	for i := 0; i < cap(retCh); i++ {
		ret := <-retCh
		ss = append(ss, ret.story)
		if ret.err != nil {
			fmt.Fprintln(stderr, ret.err)
			err = ErrApiCall
		}
	}

	if err != nil {
		return nil, errs.NewError("Update Pivotal Tracker stories", err, stderr)
	}
	return ss, nil
}
