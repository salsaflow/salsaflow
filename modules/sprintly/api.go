package sprintly

import (
	// Stdlib
	"bytes"
	"errors"
	"fmt"
	"io"
	"sort"
	"text/tabwriter"

	// Internal
	"github.com/salsaflow/salsaflow/errs"
	"github.com/salsaflow/salsaflow/log"

	// other
	"github.com/salsita/go-sprintly/sprintly"
)

// Item fetching ---------------------------------------------------------------

const maxItemsPerPage = 500

func listItems(
	client *sprintly.Client,
	productId int,
	args *sprintly.ItemListArgs,
) ([]sprintly.Item, error) {

	task := "Fetch Sprintly items"

	// Loop until we get all requested items.
	items := make([]sprintly.Item, 0)

	args.Offset = 0
	args.Limit = maxItemsPerPage
	args.Children = true

	for {
		// Fetch another page of items.
		itms, _, err := client.Items.List(productId, args)
		if err != nil {
			return nil, errs.NewError(task, err)
		}

		// Append the items to the final set of items.
		items = append(items, itms...)

		// Break the loop in case this is the last page.
		if len(itms) != maxItemsPerPage {
			break
		}

		// Increment the offset to move to the next page.
		args.Offset += maxItemsPerPage
	}

	return items, nil
}

// listItemsByNumber returns the items specified by the numbers passed in.
//
// This function returns the sub-items as well so that the caller can inter-link
// the items when necessary.
//
// This function only consults Backlog, Current and Complete for the items
// specified, but the sub-items can be in any state available.
func listItemsByNumber(
	client *sprintly.Client,
	productId int,
	numbers []int,
) ([]sprintly.Item, error) {

	task := "Fetch Sprintly items by number"
	log.Run(task)

	// Fetch the items.
	itms, err := listItems(client, productId, &sprintly.ItemListArgs{
		Status: []sprintly.ItemStatus{
			sprintly.ItemStatusBacklog,
			sprintly.ItemStatusInProgress,
			sprintly.ItemStatusCompleted,
		},
	})
	if err != nil {
		return nil, errs.NewError(task, err)
	}

	// Build necessary data structures.
	var (
		numberIndex    = make(map[int]struct{}, len(numbers))
		remainingIndex = make(map[int]struct{}, len(numbers))
	)
	for _, number := range numbers {
		numberIndex[number] = struct{}{}
		remainingIndex[number] = struct{}{}
	}

	// Collect the items we are interested in.
	items := make([]sprintly.Item, 0, len(numbers))

	for _, item := range itms {
		// Add the item in case it is requested.
		if _, ok := numberIndex[item.Number]; ok {
			items = append(items, item)
			delete(remainingIndex, item.Number)
		}
		// Add the item in case it is a sub-item of an item that is requested.
		number, err := item.ParentNumber()
		if err != nil {
			return nil, errs.NewError(task, err)
		}
		if number != 0 {
			if _, ok := numberIndex[number]; ok {
				items = append(items, item)
			}
		}
	}

	// Make sure we got all the items that were requested.
	if len(remainingIndex) != 0 {
		// Sort the remaining numbers.
		numbers := make([]int, 0, len(remainingIndex))
		for number := range remainingIndex {
			numbers = append(numbers, number)
		}
		sort.Sort(sort.IntSlice(numbers))

		// Generate the error details.
		details := new(bytes.Buffer)
		tw := tabwriter.NewWriter(details, 0, 8, 4, '\t', 0)
		fmt.Fprintln(tw)
		fmt.Fprintln(tw, "Item Number\tError")
		fmt.Fprintln(tw, "===========\t=====")
		for _, number := range numbers {
			fmt.Fprintf(tw, "%v\tnot found in Backlog, Current or Complete\n", number)
		}
		fmt.Fprintln(tw)
		tw.Flush()

		return nil, errs.NewError(task, errors.New("some items not found"), details)
	}

	return items, nil
}

func listItemsByTag(
	client *sprintly.Client,
	productId int,
	tags []string,
) ([]sprintly.Item, error) {

	task := "Fetch Sprintly items by tag"
	log.Run(task)

	// Since Sprintly API is not exactly powerful, we need to get all the items.
	// Then we need to locally pair stories with their sub-items.
	itms, err := listItems(client, productId, &sprintly.ItemListArgs{
		Status: []sprintly.ItemStatus{
			sprintly.ItemStatusSomeday,
			sprintly.ItemStatusBacklog,
			sprintly.ItemStatusInProgress,
			sprintly.ItemStatusCompleted,
		},
	})

	var (
		items     []sprintly.Item
		itemIndex = make(map[int]struct{}, 0)
	)
	for _, item := range itms {
		// In case the tag matches, add the item to the list.
		// Also add the item to the index of potential parent items.
		for _, tag := range tags {
			if tagged(&item, tag) {
				items = append(items, item)
				itemIndex[item.Number] = struct{}{}
			}
		}
	}
	for _, item := range itms {
		// In case the parent is not empty and it matches an item
		// that is already in the list, add the current item to the list as well,
		// but only if the item is not there yet.
		number, err := item.ParentNumber()
		if err != nil {
			return nil, errs.NewError(task, err)
		}
		if number != 0 {
			if _, ok := itemIndex[number]; ok {
				if _, ok := itemIndex[item.Number]; !ok {
					items = append(items, item)
				}
			}
		}
	}

	if err != nil {
		return nil, errs.NewError(task, err)
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
	return nil, errs.NewError(task, errFailedToUpdate, stderr)
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
