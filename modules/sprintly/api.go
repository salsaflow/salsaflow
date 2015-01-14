package sprintly

import (
	// Stdlib
	"bytes"
	"errors"
	"fmt"
	"io"

	// Internal
	"github.com/salsaflow/salsaflow/errs"

	// other
	"github.com/salsita/go-sprintly/sprintly"
)

// Fetch items by number concurrently ------------------------------------------

var errFailedToFetch = errors.New("failed to fetch Sprintly items")

type itemGetResult struct {
	item *sprintly.Item
	err  error
}

func listItemsByNumber(
	client *sprintly.Client,
	productId int,
	numbers []int,
) ([]sprintly.Item, error) {

	task := "Fetch Sprintly items"

	// Send out the GET requests concurrently.
	retCh := make(chan *itemGetResult, len(numbers))
	for _, number := range numbers {
		go func(number int) {
			item, _, err := client.Items.Get(productId, number)
			retCh <- &itemGetResult{item, err}
		}(number)
	}

	// Collect the results.
	var (
		items  = make([]sprintly.Item, 0, len(numbers))
		stderr = new(bytes.Buffer)
		err    error
	)
	for _ = range numbers {
		if res := <-retCh; res.err == nil {
			items = append(items, *res.item)
		} else {
			fmt.Fprintln(stderr, res.err)
			err = errFailedToFetch
		}
	}
	if err != nil {
		return nil, errs.NewError(task, err, stderr)
	}
	return items, nil
}

// Add/remove item tags concurrently -------------------------------------------

func addTagFunc(tag string) itemUpdateFunc {
	return func(item *sprintly.Item) *sprintly.ItemUpdateArgs {
		// Make sure the tag is not already there.
		tags := item.Tags
		for _, t := range tags {
			if t == tag {
				return nil
			}
		}

		// Return the update request.
		return &sprintly.ItemUpdateArgs{
			Tags: append(tags, tag),
		}
	}
}

func removeTagFunc(tag string) itemUpdateFunc {
	return func(item *sprintly.Item) *sprintly.ItemUpdateArgs {
		// Make sure the tag is there.
		index := -1
		tags := item.Tags
		for i, t := range tags {
			if t == tag {
				index = i
				break
			}
		}
		if index == -1 {
			return nil
		}

		// Return the update request.
		return &sprintly.ItemUpdateArgs{
			Tags: append(tags[:index], tags[index+1:]...),
		}
	}
}

func addTag(
	client *sprintly.Client,
	productId int,
	items []sprintly.Item,
	tag string,
) ([]sprintly.Item, error) {

	return updateItems(client, productId, items, addTagFunc(tag), removeTagFunc(tag))
}

func removeTag(
	client *sprintly.Client,
	productId int,
	items []sprintly.Item,
	tag string,
) ([]sprintly.Item, error) {

	return updateItems(client, productId, items, removeTagFunc(tag), addTagFunc(tag))
}

// Concurrent item updating ----------------------------------------------------

var errFailedToUpdate = errors.New("failed to update Sprintly items")

type itemUpdateFunc func(item *sprintly.Item) (args *sprintly.ItemUpdateArgs)

type itemUpdateResult struct {
	item *sprintly.Item
	err  error
}

func updateItems(
	client *sprintly.Client,
	productId int,
	items []sprintly.Item,
	updateFunc itemUpdateFunc,
	rollbackFunc itemUpdateFunc,
) ([]sprintly.Item, error) {

	task := "Update Sprintly items"

	// Apply the update function.
	updatedItems, updateStderr, err := applyUpdateFunc(client, productId, items, updateFunc)

	// Start generating the final stderr output.
	stderr := bytes.NewBufferString("\nUpdate Errors\n-------------\n")
	io.Copy(stderr, updateStderr)
	fmt.Fprintln(stderr)

	// In case there was no error, we are done.
	if err == nil {
		return updatedItems, nil
	}
	// In case there is no rollback function, return the error.
	if rollbackFunc == nil {
		return updatedItems, errs.NewError(task, err, stderr)
	}

	// Otherwise apply the rollback function to the updated items.
	_, rollbackStderr, err := applyUpdateFunc(client, productId, updatedItems, rollbackFunc)

	// Append to the aggregated stderr.
	if err != nil {
		fmt.Fprintln(stderr, "Rollback Errors\n---------------")
		io.Copy(stderr, rollbackStderr)
		fmt.Fprintln(stderr)
	}

	// Return the aggregated error.
	return nil, errs.NewError(task, err, stderr)
}

func applyUpdateFunc(
	client *sprintly.Client,
	productId int,
	items []sprintly.Item,
	updateFunc itemUpdateFunc,
) (newItems []sprintly.Item, stderr *bytes.Buffer, err error) {

	// Send all the request at once.
	retCh := make(chan *itemUpdateResult, len(items))
	for _, item := range items {
		go func(item sprintly.Item) {
			// Get the update arguments.
			// Returning nil means that no request is sent.
			args := updateFunc(&item)
			if args == nil {
				retCh <- &itemUpdateResult{&item, nil}
				return
			}

			// Send the update request and collect the result.
			updatedItem, _, err := client.Items.Update(productId, item.Number, args)
			if err == nil {
				// On success, return the updated item.
				retCh <- &itemUpdateResult{updatedItem, nil}
			} else {
				// On error, keep the original item, add the error.
				retCh <- &itemUpdateResult{&item, err}
			}
		}(item)
	}

	// Wait for the requests to complete.
	var (
		updatedItems = make([]sprintly.Item, 0, len(items))
		updateStderr = new(bytes.Buffer)
	)
	for _ = range items {
		if ret := <-retCh; ret.err == nil {
			updatedItems = append(updatedItems, *ret.item)
		} else {
			fmt.Fprintln(updateStderr, ret.err)
		}
	}

	if updateStderr.Len() != 0 {
		err = errFailedToUpdate
	}
	return updatedItems, updateStderr, err
}
