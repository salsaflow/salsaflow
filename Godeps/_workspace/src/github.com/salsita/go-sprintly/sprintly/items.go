package sprintly

import (
	"fmt"
	"net/http"
	"time"
)

type ItemType string

const (
	ItemTypeStory  ItemType = "story"
	ItemTypeTask   ItemType = "task"
	ItemTypeDefect ItemType = "defect"
	ItemTypeTest   ItemType = "test"
)

type ItemStatus string

const (
	ItemStatusSomeday    ItemStatus = "someday"
	ItemStatusBacklog    ItemStatus = "backlog"
	ItemStatusInProgress ItemStatus = "in-progress"
	ItemStatusCompleted  ItemStatus = "completed"
	ItemStatusAccepted   ItemStatus = "accepted"
)

type ItemScore string

const (
	ItemScoreUnset     ItemScore = "~"
	ItemScoreSmall     ItemScore = "S"
	ItemScoreMedium    ItemScore = "M"
	ItemScoreLarge     ItemScore = "L"
	ItemScoreVeryLarge ItemScore = "XL"
)

type ItemOrdering string

const (
	ItemOrderingOldest    ItemOrdering = "oldest"
	ItemOrderingNewest                 = "newest"
	ItemOrderingPriority               = "priority"
	ItemOrderingRecent                 = "recent"
	ItemOrderingStale                  = "stale"
	ItemOrderingActive                 = "active"
	ItemOrderingAbandoned              = "abandoned"
)

// ItemsService holds all the methods for manipulating Sprintly items.
type ItemsService struct {
	client *Client
}

func newItemsService(client *Client) *ItemsService {
	return &ItemsService{client}
}

// Item represents a Sprintly item.
type Item struct {
	// Common fields.
	Number       int           `json:"number,omitempty"`
	Type         string        `json:"type,omitempty"`
	Title        string        `json:"title,omitempty"`
	Description  string        `json:"description,omitempty"`
	Score        ItemScore     `json:"score,omitempty"`
	Status       ItemStatus    `json:"status,omitempty"`
	Tags         []string      `json:"tags,omitempty"`
	ShortURL     string        `json:"short_url,omitempty"`
	Product      *Product      `json:"product,omitempty"`
	Progress     *ItemProgress `json:"progress,omitempty"`
	CreatedAt    *time.Time    `json:"created_at,omitempty"`
	CreatedBy    *User         `json:"created_by,omitempty"`
	AssignedTo   *User         `json:"assigned_to,omitempty"`
	Counts       *ItemCounts   `json:"counts,omitempty"`
	Email        *ItemEmail    `json:"email,omitempty"`
	LastModified *time.Time    `json:"last_modified,omitempty"`
	Archived     bool          `json:"archived,omitempty"`

	// Stories only.
	Who  string `json:"who,omitempty"`
	What string `json:"what,omitempty"`
	Why  string `json:"why,omitempty"`
}

// Progress represents a Sprintly item progress.
type ItemProgress struct {
	TriagedAt  *time.Time `json:"triaged_at,omitempty"`
	StartedAt  *time.Time `json:"started_at,omitempty"`
	AcceptedAt *time.Time `json:"accepted_at,omitempty"`
	ClosedAt   *time.Time `json:"closed_at,omitempty"`
}

type ItemCounts struct {
	Blockers  int `json:"blockers,omitempty"`
	Blocking  int `json:"blocking,omitempty"`
	Comments  int `json:"comments,omitempty"`
	Favorites int `json:"favorites,omitempty"`
}

type ItemEmail struct {
	Discussion string `json:"discussion,omitempty"`
	Files      string `json:"files,omitempty"`
}

// ItemCreateArgs represent the arguments that can be passed into Items.Create.
type ItemCreateArgs struct {
	Type        string     `url:"type,omitempty"        schema:"type,omitempty"`
	Title       string     `url:"title,omitempty"       schema:"title,omitempty"`
	Who         string     `url:"who,omitempty"         schema:"who,omitempty"`
	What        string     `url:"what,omitempty"        schema:"what,omitempty"`
	Why         string     `url:"why,omitempty"         schema:"why,omitempty"`
	Description string     `url:"description,omitempty" schema:"description,omitempty"`
	Score       ItemScore  `url:"score,omitempty"       schema:"score,omitempty"`
	Status      ItemStatus `url:"status,omitempty"      schema:"status,omitempty"`
	AssignedTo  int        `url:"assigned_to,omitempty" schema:"assigned_to,omitempty"`
	Tags        []string   `url:"tags,comma,omitempty"  schema:"tags,omitempty"`
}

// ItemUpdateArgs represent the arguments that can be passed into Items.Update.
//
// This struct is the same as ItemCreateArgs, just the Parent field is extra.
type ItemUpdateArgs struct {
	Type        string     `url:"type,omitempty"        schema:"type,omitempty"`
	Title       string     `url:"title,omitempty"       schema:"title,omitempty"`
	Who         string     `url:"who,omitempty"         schema:"who,omitempty"`
	What        string     `url:"what,omitempty"        schema:"what,omitempty"`
	Why         string     `url:"why,omitempty"         schema:"why,omitempty"`
	Description string     `url:"description,omitempty" schema:"description,omitempty"`
	Score       ItemScore  `url:"score,omitempty"       schema:"score,omitempty"`
	Status      ItemStatus `url:"status,omitempty"      schema:"status,omitempty"`
	AssignedTo  int        `url:"assigned_to,omitempty" schema:"assigned_to,omitempty"`
	Tags        []string   `url:"tags,comma,omitempty"  schema:"tags,omitempty"`
	Parent      int        `url:"parent,omitempty"      schema:"parent,omitempty"`
}

// ItemListArgs represents the arguments for the List method.
type ItemListArgs struct {
	Type       []ItemType   `url:"type,comma,omitempty"   schema:"type,omitempty"`
	Status     []ItemStatus `url:"status,comma,omitempty" schema:"status,omitempty"`
	Offset     int          `url:"offset,omitempty"       schema:"offset,omitempty"`
	Limit      int          `url:"limit,omitempty"        schema:"limit,omitempty"`
	OrderBy    ItemOrdering `url:"order_by,omitempty"     schema:"order_by,omitempty"`
	AssignedTo int          `url:"assigned_to,omitempty"  schema:"assigned_to,omitempty"`
	CreatedBy  int          `url:"created_by,omitempty"   schema:"created_by,omitempty"`
	Tags       []string     `url:"tags,comma,omitempty"   schema:"tags,omitempty"`
	Children   bool         `url:"children,omitempty"     schema:"children,omitempty"`
}

// Create can be used to create new items.
//
// See https://sprintly.uservoice.com/knowledgebase/articles/98412-items
func (srv ItemsService) Create(productId int, args *ItemCreateArgs) (*Item, *http.Response, error) {
	u := fmt.Sprintf("products/%v/items.json", productId)

	req, err := srv.client.NewPostRequest(u, args)
	if err != nil {
		return nil, nil, err
	}

	var item Item
	resp, err := srv.client.Do(req, &item)
	if err != nil {
		switch resp.StatusCode {
		case 400:
			return nil, nil, &ErrItems400{err.(*ErrAPI)}
		case 404:
			return nil, nil, &ErrItems404{err.(*ErrAPI)}
		default:
			return nil, resp, err
		}
	}

	return &item, resp, nil
}

// List can be used to list items for the given product according to the given arguments.
//
// See https://sprintly.uservoice.com/knowledgebase/articles/98412-items
func (srv ItemsService) List(productId int, args *ItemListArgs) ([]Item, *http.Response, error) {
	u := fmt.Sprintf("products/%v/items.json", productId)

	req, err := srv.client.NewGetRequest(u, args)
	if err != nil {
		return nil, nil, err
	}

	var items []Item
	resp, err := srv.client.Do(req, &items)
	if err != nil {
		switch resp.StatusCode {
		case 400:
			return nil, nil, &ErrItems400{err.(*ErrAPI)}
		case 404:
			return nil, nil, &ErrItems404{err.(*ErrAPI)}
		default:
			return nil, resp, err
		}
	}

	return items, resp, nil
}

// Get can be used to get the item identified by the given item number.
//
// See https://sprintly.uservoice.com/knowledgebase/articles/98412-items
func (srv ItemsService) Get(productId, itemNumber int) (*Item, *http.Response, error) {
	u := fmt.Sprintf("products/%v/items/%v.json", productId, itemNumber)

	req, err := srv.client.NewGetRequest(u, nil)
	if err != nil {
		return nil, nil, err
	}

	var item Item
	resp, err := srv.client.Do(req, &item)
	if err != nil {
		switch resp.StatusCode {
		case 400:
			return nil, nil, &ErrItems400{err.(*ErrAPI)}
		case 404:
			return nil, nil, &ErrItems404{err.(*ErrAPI)}
		default:
			return nil, resp, err
		}
	}

	return &item, resp, nil
}

// Update can be used to update the item identified by the given item number.
//
// See https://sprintly.uservoice.com/knowledgebase/articles/98412-items
func (srv ItemsService) Update(
	productId int,
	itemNumber int,
	args *ItemUpdateArgs,
) (*Item, *http.Response, error) {

	u := fmt.Sprintf("products/%v/items/%v.json", productId, itemNumber)

	req, err := srv.client.NewPostRequest(u, args)
	if err != nil {
		return nil, nil, err
	}

	var item Item
	resp, err := srv.client.Do(req, &item)
	if err != nil {
		switch resp.StatusCode {
		case 400:
			return nil, nil, &ErrItems400{err.(*ErrAPI)}
		case 404:
			return nil, nil, &ErrItems404{err.(*ErrAPI)}
		default:
			return nil, resp, err
		}
	}

	return &item, resp, nil
}

// ListChildren can be used to list children of the given item.
//
// See https://sprintly.uservoice.com/knowledgebase/articles/98412-items
func (srv ItemsService) ListChildren(productId, itemNumber int) ([]Item, *http.Response, error) {
	u := fmt.Sprintf("products/%v/items/%v/children.json", productId, itemNumber)

	req, err := srv.client.NewGetRequest(u, nil)
	if err != nil {
		return nil, nil, err
	}

	var items []Item
	resp, err := srv.client.Do(req, &items)
	if err != nil {
		return nil, resp, err
	}

	return items, resp, nil
}
