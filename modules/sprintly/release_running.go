package sprintly

import (
	// Stdlib
	"bytes"
	"errors"
	"fmt"
	"io"
	"text/tabwriter"

	// Internal
	"github.com/salsaflow/salsaflow/action"
	"github.com/salsaflow/salsaflow/errs"
	"github.com/salsaflow/salsaflow/log"
	"github.com/salsaflow/salsaflow/modules/common"
	"github.com/salsaflow/salsaflow/version"

	// Other
	"github.com/salsita/go-sprintly/sprintly"
)

var errNotStageable = errors.New("release cannot be staged")

type runningRelease struct {
	client *sprintly.Client
	config Config

	version *version.Version

	cachedItems []sprintly.Item
}

func (release *runningRelease) Version() *version.Version {
	return release.version
}

func (release *runningRelease) Stories() ([]common.Story, error) {
	items, err := release.items()
	if err != nil {
		return nil, err
	}
	return toCommonStories(items), nil
}

func (release *runningRelease) EnsureStageable() error {
	task := "Make sure the items can be staged"
	log.Run(task)

	// Get relevant items.
	items, err := release.items()
	if err != nil {
		return err
	}

	// Make sure the items can be staged.
	var details bytes.Buffer
	tw := tabwriter.NewWriter(&details, 0, 8, 4, '\t', 0)
	io.WriteString(tw, "\n")
	io.WriteString(tw, "Item URL\tError\n")
	io.WriteString(tw, "========\t=====\n")

	// An item is considered stageable when its status is Completed.
	// That by definition means that it has been reviewed and verified.
	for _, item := range items {
		// Completed and Accepted states are considered stageable.
		switch item.Status {
		case sprintly.ItemStatusCompleted:
			continue
		case sprintly.ItemStatusAccepted:
			continue
		}

		// Other states are considered not stageable.
		fmt.Fprintf(tw, "%v\titem status not stageable: %v\n", item.ShortURL, item.Status)
		err = errNotStageable
	}
	if err != nil {
		io.WriteString(tw, "\n")
		tw.Flush()
		return errs.NewError(task, err, &details)
	}
	return nil
}

// Stage can be used to stage the items associated with this release.
//
// The rollback function is a NOOP in this case since there is no way
// how to delete a deployment once created in Sprintly.
func (release *runningRelease) Stage() (action.Action, error) {
	task := "Ping Sprintly to register the deployment"
	log.Run(task)

	// Create the Sprintly deployment.
	if err := release.deploy(release.config.StagingEnvironment()); err != nil {
		return nil, errs.NewError(task, err)
	}

	// Return the rollback function, which is empty in this case.
	return action.ActionFunc(func() error {
		log.Rollback(task)
		log.Warn("It is not possible to delete a Sprintly deployment, skipping ...")
		return nil
	}), nil

}

func (release *runningRelease) Releasable() (bool, error) {
	// Get the items associated with this release.
	items, err := release.items()
	if err != nil {
		return false, err
	}

	// Make sure that all items are Accepted.
	for _, item := range items {
		if item.Status != sprintly.ItemStatusAccepted {
			return false, nil
		}
	}
	return true, nil
}

func (release *runningRelease) Release() error {
	task := "Ping Sprintly to register the deployment"
	log.Run(task)

	// Create the Sprintly deployment.
	return release.deploy(release.config.ProductionEnvironment())
}

func (release *runningRelease) items() ([]sprintly.Item, error) {
	// Fetch the items unless cached.
	if release.cachedItems == nil {
		var (
			client         = release.client
			productId      = release.config.ProductId()
			itemReleaseTag = getItemReleaseTag(release.version)
		)
		// Need to list all possible item statuses since by default
		// only the items from Backlog are returned.
		items, err := listItemsByTag(client, productId, []string{itemReleaseTag})
		if err != nil {
			return nil, err
		}

		// Make sure that all items are tagged for the release.
		// In can happen that someone adds a sub-item and forgets to tag it.
		missingTag := make([]sprintly.Item, 0, len(items))
		for _, item := range items {
			if !tagged(&item, itemReleaseTag) {
				missingTag = append(missingTag, item)
			}
		}
		if len(missingTag) != 0 {
			task := "Automatically add sub-items into the release"
			log.Run(task)

			// Update the items in Sprintly.
			_, err := addTag(client, productId, missingTag, itemReleaseTag)
			if err != nil {
				return nil, errs.NewError(task, err)
			}

			// Update the local objects.
			for _, item := range items {
				if !tagged(&item, itemReleaseTag) {
					item.Tags = append(item.Tags, itemReleaseTag)
				}
			}
		}

		release.cachedItems = items
	}

	// Return the cached items.
	return release.cachedItems, nil
}

func (release *runningRelease) deploy(environment string) error {
	// Get the items associated with this release.
	items, err := release.items()
	if err != nil {
		return err
	}

	// Collect the item numbers that are being deployed.
	numbers := make([]int, 0, len(items))
	for _, item := range items {
		numbers = append(numbers, item.Number)
	}

	// Create the deployment.
	var (
		client    = release.client
		productId = release.config.ProductId()
	)
	_, _, err = client.Deploys.Create(productId, &sprintly.DeployCreateArgs{
		Environment: environment,
		ItemNumbers: numbers,
	})
	return err
}
