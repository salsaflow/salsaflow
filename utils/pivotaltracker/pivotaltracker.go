package pivotaltracker

import (
	// Stdlib
	"bytes"
	"errors"
	"fmt"

	// Internal
	"github.com/tchap/git-trunk/config"

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

func SetStoriesState(stories []*pivotal.Story, state string) (stderr *bytes.Buffer, err error) {
	// Prepare the PT client.
	var (
		token     = config.PivotalTracker.ApiToken()
		projectId = config.PivotalTracker.ProjectId()
	)
	client := pivotal.NewClient(token)

	// Send all the request at once.
	errCh := make(chan error, len(stories))
	postBody := &pivotal.Story{State: state}
	for _, story := range stories {
		go func() {
			_, _, ex := client.Stories.Update(projectId, story.Id, postBody)
			errCh <- ex
		}()
	}

	// Wait for the requests to return.
	stderr = new(bytes.Buffer)
	for i := 0; i < cap(errCh); i++ {
		if ex := <-errCh; ex != nil {
			fmt.Fprintln(stderr, ex)
			err = ErrApiCall
		}
	}

	return
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
