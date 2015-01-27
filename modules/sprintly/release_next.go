package sprintly

import (
	// Stdlib
	"errors"
	"fmt"
	"os"
	"strconv"

	// Internal
	"github.com/salsaflow/salsaflow/action"
	"github.com/salsaflow/salsaflow/errs"
	"github.com/salsaflow/salsaflow/log"
	"github.com/salsaflow/salsaflow/prompt"
	"github.com/salsaflow/salsaflow/releases"
	"github.com/salsaflow/salsaflow/version"

	// Other
	"github.com/salsita/go-sprintly/sprintly"
)

type nextRelease struct {
	tracker *issueTracker
	client  *sprintly.Client

	trunkVersion     *version.Version
	nextTrunkVersion *version.Version

	additionalItems []sprintly.Item
}

func (release *nextRelease) PromptUserToConfirmStart() (bool, error) {
	var (
		client         = release.client
		productId      = release.tracker.config.ProductId()
		itemReleaseTag = getItemReleaseTag(release.trunkVersion)
	)

	// Collect the commits that modified trunk since the last release.
	task := "Collect the stories that modified trunk"
	log.Run(task)
	ids, err := releases.ListStoryIdsToBeAssigned(release.tracker)
	if err != nil {
		return false, errs.NewError(task, err, nil)
	}

	// Get the story ids associated with these commits.
	numbers := make([]int, 0, len(ids))
	for _, id := range ids {
		number, err := strconv.Atoi(id)
		if err != nil {
			return false, errs.NewError(task, fmt.Errorf("invalid item number: %v", id), nil)
		}
		numbers = append(numbers, number)
	}

	// Fetch the collected items from Sprintly, if necessary.
	var additional []sprintly.Item
	if len(numbers) != 0 {
		var err error
		// listItemsByNumber lists children as well, so there is no way we can miss a sub-item.
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

	// Check the additional items and collect the unrated ones.
	unrated := make([]sprintly.Item, 0)
	for _, item := range additional {
		if item.Score == sprintly.ItemScoreUnset {
			unrated = append(unrated, item)
		}
	}

	// Fetch the items that were assigned manually.
	assignedItems, err := listItemsByTag(client, productId, []string{itemReleaseTag})
	if err != nil {
		return false, errs.NewError(task, err, nil)
	}

	// Check the manually assigned items and collect the unrated ones.
	for _, item := range assignedItems {
		if item.Score == sprintly.ItemScoreUnset {
			unrated = append(unrated, item)
		}

		// Also, since the sub-items of the assigned items are returned as well,
		// they may be missing the release tag, so let's register them to be tagged.
		if !tagged(&item, itemReleaseTag) {
			additional = append(additional, item)
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
		productId      = release.tracker.config.ProductId()
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
