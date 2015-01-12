package sprintly

import (
	// Stdlib
	"fmt"

	// Internal
	"github.com/salsaflow/salsaflow/errs"
	"github.com/salsaflow/salsaflow/modules/common"
	"github.com/salsaflow/salsaflow/version"

	// Other
	// "github.com/toqueteos/webbrowser"
	"github.com/salsita/go-sprintly/sprintly"
)

type issueTracker struct {
	config Config
	client *sprintly.Client
}

func Factory() (common.IssueTracker, error) {
	config, err := LoadConfig()
	if err != nil {
		return nil, err
	}

	client := sprintly.NewClient(config.Username(), config.Token())
	return &issueTracker{config, client}, nil
}

func (tracker *issueTracker) CurrentUser() (common.User, error) {
	task := "Fetch the current user from Sprintly"

	var (
		productId = tracker.config.ProductId()
		username  = tracker.config.Username()
	)

	// Fetch all members of this product.
	users, _, err := tracker.client.People.List(productId)
	if err != nil {
		return nil, errs.NewError(task, err, nil)
	}

	// Find the user with matching username.
	for _, usr := range users {
		if usr.Email == username {
			return &user{&usr}, nil
		}
	}

	// In case there is no such user, they were not invited yet.
	return nil, errs.NewError(
		task, fmt.Errorf("user '%v' not a member of this product", username), nil)
}

func (tracker *issueTracker) StartableStories() (stories []common.Story, err error) {
	task := "Fetch the startable items from Sprintly"

	var (
		productId = tracker.config.ProductId()
		username  = tracker.config.Username()
	)

	// Fetch the items from Sprintly.
	items, _, err := tracker.client.Items.List(productId, &sprintly.ItemListArgs{
		Status: []sprintly.ItemStatus{sprintly.ItemStatusBacklog},
	})
	if err != nil {
		return nil, errs.NewError(task, err, nil)
	}

	// Drop the items that were already assigned.
	// However, keep the items that are assigned to the current user.
	for i, item := range items {
		if item.AssignedTo == nil {
			continue
		}
		if item.AssignedTo.Email == username {
			continue
		}
		items = append(items[:i], items[:i+1]...)
	}

	// Wrap the result as []common.Story
	return toCommonStories(items), nil
}

func (tracker *issueTracker) StoriesInDevelopment() (stories []common.Story, err error) {
	task := "Fetch the items that are in progress"

	// Fetch all items that are in progress.
	productId := tracker.config.ProductId()
	items, _, err := tracker.client.Items.List(productId, &sprintly.ItemListArgs{
		Status: []sprintly.ItemStatus{sprintly.ItemStatusInProgress},
	})
	if err != nil {
		return nil, errs.NewError(task, err, nil)
	}

	// Drop the items that are not assigned to the current user.
	// We could do that in the query already, but we need user ID for that,
	// which would require another remote call. However, we know the username
	// since it is saved in the configuration file, so we can filter locally
	// based on that information.
	username := tracker.config.Username()
	for i, item := range items {
		if item.AssignedTo.Email != username {
			items = append(items[:i], items[i+1:]...)
		}
	}

	// Convert the items into []common.Story
	return toCommonStories(items), nil
}

func (tracker *issueTracker) NextRelease(
	trunkVersion *version.Version,
	nextTrunkVersion *version.Version,
) (common.NextRelease, error) {

	panic("Not implemented")
}

func (tracker *issueTracker) RunningRelease(
	releaseVersion *version.Version,
) (common.RunningRelease, error) {

	panic("Not implemented")
}

func (tracker *issueTracker) OpenStory(storyId string) error {
	panic("Not implemented")
}

func (tracker *issueTracker) StoryTagToReadableStoryId(tag string) (storyId string, err error) {
	panic("Not implemented")
}
