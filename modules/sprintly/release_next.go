package sprintly

import (
	// Stdlib
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	// Internal
	"github.com/salsaflow/salsaflow/action"
	"github.com/salsaflow/salsaflow/errs"
	"github.com/salsaflow/salsaflow/git"
	"github.com/salsaflow/salsaflow/log"
	"github.com/salsaflow/salsaflow/prompt"
	"github.com/salsaflow/salsaflow/releases"
	"github.com/salsaflow/salsaflow/version"

	// Other
	"github.com/salsita/go-sprintly/sprintly"
)

type nextRelease struct {
	client *sprintly.Client
	config Config

	trunkVersion     *version.Version
	nextTrunkVersion *version.Version

	additionalItems []sprintly.Item
}

func newNextRelease(
	client *sprintly.Client,
	config Config,
	trunkVersion *version.Version,
	nextTrunkVersion *version.Version,
) (*nextRelease, error) {

	return &nextRelease{
		client:           client,
		config:           config,
		trunkVersion:     trunkVersion,
		nextTrunkVersion: nextTrunkVersion,
	}, nil
}

func (release *nextRelease) PromptUserToConfirmStart() (bool, error) {
	var (
		client         = release.client
		productId      = release.config.ProductId()
		itemReleaseTag = getItemReleaseTag(release.trunkVersion)
	)

	// Collect the commits that modified trunk since the last release.
	task := "Collect the stories that modified trunk"
	log.Run(task)
	commits, err := releases.ListNewTrunkCommits()
	if err != nil {
		return false, errs.NewError(task, err, nil)
	}

	// Get the story ids associated with these commits.
	tags := git.StoryIdTags(commits)
	numbers := make([]int, 0, len(tags))
	for _, tag := range tags {
		// Story-Id tag for Sprintly is "<product-id>/item/<item-number>".
		parts := strings.Split(tag, "/")
		if len(parts) != 3 {
			return false, errs.NewError(task, fmt.Errorf("invalid Story-Id tag: %v", tag), nil)
		}
		number, err := strconv.Atoi(parts[2])
		if err != nil {
			return false, errs.NewError(task, fmt.Errorf("invalid Story-Id tag: %v", tag), nil)
		}
		numbers = append(numbers, number)
	}

	// Fetch the collected items from Sprintly, if necessary.
	var additional []sprintly.Item
	if len(numbers) != 0 {
		task = "Fetch the collected items from Sprintly"
		log.Run(task)

		var err error
		additional, err = listItemsByNumber(client, productId, numbers)
		if err != nil {
			return false, err
		}

		// Drop the issues that are already assigned to the right release.
		unassignedItems := make([]sprintly.Item, 0, len(additional))
		for _, item := range additional {
			if tagged(&item, itemReleaseTag) {
				continue
			}
			unassignedItems = append(unassignedItems, item)
		}
		additional = unassignedItems
	}

	// Make sure there are no unrated items.
	task = "Make sure there are no unrated items"
	log.Run(task)

	// Fetch the items that were assigned manually.
	assignedItems, _, err := client.Items.List(productId, &sprintly.ItemListArgs{
		Tags: []string{itemReleaseTag},
	})
	if err != nil {
		return false, errs.NewError(task, err, nil)
	}
	// Keep only the items that are unrated (score not set).
	unrated := make([]sprintly.Item, 0, len(assignedItems))
	for _, item := range assignedItems {
		if item.Score == sprintly.ItemScoreUnset {
			unrated = append(unrated, item)
		}
	}
	// Also add these that are to be added but are unrated.
	for _, item := range additional {
		if item.Score == sprintly.ItemScoreUnset {
			unrated = append(unrated, item)
		}
	}
	// In case there are some unrated items, abort the release process.
	if len(unrated) != 0 {
		fmt.Println("\nThe following items have not been rated yet:\n")
		err := prompt.ListStories(toCommonStories(unrated), os.Stdout)
		if err != nil {
			return false, err
		}
		fmt.Println()
		return false, errs.NewError(task, errors.New("unrated items detected"), nil)
	}

	// Print the items to be added to the release.
	if len(additional) != 0 {
		fmt.Println("\nThe following items are going to be added to the release:\n")
		err := prompt.ListStories(toCommonStories(additional), os.Stdout)
		if err != nil {
			return false, err
		}
	}

	// Ask the user to confirm.
	ok, err := prompt.Confirm(
		fmt.Sprintf("\nAre you sure you want to start release %v?", release.trunkVersion))
	if err == nil {
		release.additionalItems = additional
	}
	return ok, err
}

func (release *nextRelease) Start() (action.Action, error) {
	var (
		client         = release.client
		productId      = release.config.ProductId()
		itemReleaseTag = getItemReleaseTag(release.trunkVersion)
	)

	// Add the release tag to the relevant Sprintly items.
	task := "Tag relevant items with the release tag"
	log.Run(task)
	items, err := addTag(client, productId, release.additionalItems, itemReleaseTag)
	if err != nil {
		return nil, errs.NewError(task, err, nil)
	}
	release.additionalItems = nil

	// Return the rollback action, which removes the release tags that were added.
	return action.ActionFunc(func() error {
		log.Rollback(task)
		_, err := removeTag(client, productId, items, itemReleaseTag)
		if err != nil {
			return errs.NewError("Remove the release tag from relevant items", err, nil)
		}
		return nil
	}), nil
}
