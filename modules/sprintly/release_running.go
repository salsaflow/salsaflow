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

const DeploymentEnvironmentStaging = "staging"

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
	task := "Create a new deployment for the stories being staged"
	log.Run(task)

	// Get the items associated with this release.
	items, err := release.items()
	if err != nil {
		return nil, errs.NewError(task, err, nil)
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
		Environment: DeploymentEnvironmentStaging,
		ItemNumbers: numbers,
	})
	if err != nil {
		return nil, errs.NewError(task, err, nil)
	}

	// Return the rollback function, which is empty in this case.
	return action.ActionFunc(func() error {
		log.Rollback(task)
		log.Warn("It is not possible to delete a Sprintly deployment, skipping ...")
		return nil
	}), nil
}

func (release *runningRelease) Releasable() (bool, error) {
	panic("Not implemented")
}

func (release *runningRelease) Release() error {
	panic("Not implemented")
}

func (release *runningRelease) items() ([]sprintly.Item, error) {
	// Fetch the items unless cached.
	if release.cachedItems == nil {
		task := "Fetch items from Sprintly"
		log.Run(task)
		var (
			client         = release.client
			productId      = release.config.ProductId()
			itemReleaseTag = getItemReleaseTag(release.version)
		)
		// Need to list all possible item statuses since by default
		// only the items from Backlog are returned.
		items, _, err := client.Items.List(productId, &sprintly.ItemListArgs{
			Status: []sprintly.ItemStatus{
				sprintly.ItemStatusSomeday,
				sprintly.ItemStatusBacklog,
				sprintly.ItemStatusInProgress,
				sprintly.ItemStatusCompleted,
				sprintly.ItemStatusAccepted,
			},
			Tags: []string{itemReleaseTag},
		})
		if err != nil {
			return nil, errs.NewError(task, err, nil)
		}
		release.cachedItems = items
	}

	// Return the cached items.
	return release.cachedItems, nil
}
