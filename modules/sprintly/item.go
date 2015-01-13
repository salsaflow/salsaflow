package sprintly

import (
	// Stdlib
	"fmt"
	"strconv"

	// Internal
	"github.com/salsaflow/salsaflow/errs"
	"github.com/salsaflow/salsaflow/modules/common"

	// Other
	"github.com/salsita/go-sprintly/sprintly"
)

type item struct {
	*sprintly.Item
}

func (item *item) Id() string {
	return strconv.Itoa(item.Item.Number)
}

// Sprintly items doen't have any readable id, just return the item ID.
func (item *item) ReadableId() string {
	return item.Id()
}

func (item *item) Tag() string {
	return fmt.Sprintf("%v/item/%v", item.Product.Id, item.Number)
}

func (item *item) Title() string {
	return item.Item.Title
}

func (item *item) Assignees() []common.User {
	if item.AssignedTo == nil {
		return nil
	}
	return []common.User{&user{item.AssignedTo}}
}

func (item *item) AddAssignee(user common.User) *errs.Error {
	task := "Assign the current user to the selected story"

	// Load the Sprintly config.
	config, err := LoadConfig()
	if err != nil {
		return errs.NewError(task, err, nil)
	}

	// Parse the user ID.
	userId, err := strconv.Atoi(user.Id())
	if err != nil {
		panic(err)
	}

	// Instantiate the Sprintly client.
	client := sprintly.NewClient(config.Username(), config.Token())

	// Assign the user to the selected story.
	_, _, err = client.Items.Update(config.ProductId(), item.Number, &sprintly.ItemUpdateArgs{
		AssignedTo: userId,
	})
	if err != nil {
		return errs.NewError(task, err, nil)
	}
	return nil
}

func (item *item) SetAssignees(users []common.User) *errs.Error {
	panic("Not implemented")
}

func (item *item) Start() *errs.Error {
	task := fmt.Sprintf("Mark Sprintly item %v as being in progress", item.Number)

	// Check whether we are not finished already.
	switch item.Status {
	case sprintly.ItemStatusSomeday:
	case sprintly.ItemStatusBacklog:
	case sprintly.ItemStatusInProgress:
		fallthrough
	case sprintly.ItemStatusCompleted:
		fallthrough
	case sprintly.ItemStatusAccepted:
		// Nothing to do.
		return nil
	}

	// Load the Sprintly config.
	config, err := LoadConfig()
	if err != nil {
		return errs.NewError(task, err, nil)
	}

	// Instantiate the Sprintly client.
	client := sprintly.NewClient(config.Username(), config.Token())

	// Move the item to in-progress.
	_, _, err = client.Items.Update(config.ProductId(), item.Number, &sprintly.ItemUpdateArgs{
		Status: sprintly.ItemStatusInProgress,
	})
	if err != nil {
		return errs.NewError(task, err, nil)
	}
	return nil
}

func (itm *item) LessThan(story common.Story) bool {
	return itm.CreatedAt.Before(*story.(*item).CreatedAt)
}
