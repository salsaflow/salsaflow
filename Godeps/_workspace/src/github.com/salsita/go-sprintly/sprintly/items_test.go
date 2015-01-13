package sprintly

import (
	"fmt"
	"net/http"
	"testing"
	"time"
)

var testingUser = User{
	Id:        1,
	Email:     "joe@joestump.net",
	FirstName: "Joe",
	LastName:  "Stump",
}

var testingUserJson = `
{
	"id": 1,
	"email": "joe@joestump.net",
	"first_name": "Joe",
	"last_name": "Stump"
}`

var testingUserSlice = []User{testingUser}

var testingUserSliceJson = fmt.Sprintf("[%v]", testingUserJson)

var testingProduct = Product{
	Id:       1,
	Name:     "sprint.ly",
	Archived: false,
}

var testingTask = Item{
	Number:      188,
	Title:       "Don't let un-scored items out of the backlog.",
	Description: "Require people to estimate the score of an item before they can start working on it.",
	Score:       "M",
	Status:      ItemStatusBacklog,
	Tags:        []string{"scoring"},
	Product:     &testingProduct,
	CreatedBy:   &testingUser,
	AssignedTo:  &testingUser,
	Archived:    false,
	Type:        "task",
}

var testingTaskSlice []Item

func init() {
	layout := "2006-01-02T15:04:05-07:00"
	acceptedAt, err := time.Parse(layout, "2013-06-14T22:52:07+00:00")
	if err != nil {
		panic(err)
	}
	closedAt, err := time.Parse(layout, "2013-06-14T21:53:43+00:00")
	if err != nil {
		panic(err)
	}
	startedAt, err := time.Parse(layout, "2013-06-14T21:50:36+00:00")
	if err != nil {
		panic(err)
	}

	testingTask.Progress = &ItemProgress{
		StartedAt:  &startedAt,
		AcceptedAt: &acceptedAt,
		ClosedAt:   &closedAt,
	}

	testingTaskSlice = []Item{testingTask}
}

var testingTaskString = `
{
	"status": "backlog",
	"product": {
		"archived": false,
		"id": 1,
		"name": "sprint.ly"
	},
	"progress": {
		"accepted_at": "2013-06-14T22:52:07+00:00",
		"closed_at": "2013-06-14T21:53:43+00:00",
		"started_at": "2013-06-14T21:50:36+00:00"
	},
	"description": "Require people to estimate the score of an item before they can start working on it.",
	"tags": [
		"scoring"
	],
	"number": 188,
	"archived": false,
	"title": "Don't let un-scored items out of the backlog.",
	"created_by": {
		"first_name": "Joe",
		"last_name": "Stump",
		"id": 1,
		"email": "joe@joestump.net"
	},
	"score": "M",
	"assigned_to": {
		"first_name": "Joe",
		"last_name": "Stump",
		"id": 1,
		"email": "joe@joestump.net"
	},
	"type": "task"
}`

var testingTaskSliceString = fmt.Sprintf("[%v]", testingTaskString)

func TestItems_Create(t *testing.T) {
	client, server, mux := setup()
	defer server.Close()

	args := ItemCreateArgs{
		Type:        testingTask.Type,
		Title:       testingTask.Title,
		Who:         "user",
		What:        "not to be able to move un-scored items out of the backlog",
		Why:         "it does not make any sense",
		Description: testingTask.Description,
		Score:       testingTask.Score,
		Status:      testingTask.Status,
		AssignedTo:  testingUser.Id,
		Tags:        testingTask.Tags,
	}

	mux.HandleFunc("/products/1/items.json", func(w http.ResponseWriter, r *http.Request) {
		ensureMethod(t, r, "POST")

		var got ItemCreateArgs
		if err := decodeArgs(&got, r); err != nil {
			t.Error(err)
			return
		}

		ensureEqual(t, &got, &args)
		fmt.Fprint(w, testingTaskString)
	})

	item, _, err := client.Items.Create(1, &args)
	if err != nil {
		t.Errorf("Items.Create failed: %v", err)
		return
	}

	ensureEqual(t, item, &testingTask)
}

func TestItems_List(t *testing.T) {
	client, server, mux := setup()
	defer server.Close()

	mux.HandleFunc("/products/1/items.json", func(w http.ResponseWriter, r *http.Request) {
		ensureMethod(t, r, "GET")
		fmt.Fprint(w, testingTaskSliceString)
	})

	items, _, err := client.Items.List(1, nil)
	if err != nil {
		t.Errorf("Items.List failed: %v", err)
		return
	}

	ensureEqual(t, items, testingTaskSlice)
}

func TestItems_Get(t *testing.T) {
	client, server, mux := setup()
	defer server.Close()

	mux.HandleFunc("/products/1/items/188.json", func(w http.ResponseWriter, r *http.Request) {
		ensureMethod(t, r, "GET")
		fmt.Fprint(w, testingTaskString)
	})

	item, _, err := client.Items.Get(1, 188)
	if err != nil {
		t.Errorf("Items.Get failed: %v", err)
		return
	}

	ensureEqual(t, item, &testingTask)
}

func TestItems_Update(t *testing.T) {
	client, server, mux := setup()
	defer server.Close()

	args := ItemUpdateArgs{
		Type:        testingTask.Type,
		Title:       testingTask.Title,
		Who:         "user",
		What:        "not to be able to move un-scored items out of the backlog",
		Why:         "it does not make any sense",
		Description: testingTask.Description,
		Score:       testingTask.Score,
		Status:      testingTask.Status,
		AssignedTo:  testingUser.Id,
		Tags:        testingTask.Tags,
		Parent:      99,
	}

	mux.HandleFunc("/products/1/items/188.json", func(w http.ResponseWriter, r *http.Request) {
		ensureMethod(t, r, "POST")

		var got ItemUpdateArgs
		if err := decodeArgs(&got, r); err != nil {
			t.Error(err)
			return
		}

		ensureEqual(t, &got, &args)
		fmt.Fprint(w, testingTaskString)
	})

	item, _, err := client.Items.Update(1, 188, &args)
	if err != nil {
		t.Errorf("Items.Update failed: %v", err)
		return
	}

	ensureEqual(t, item, &testingTask)
}

func TestItems_ListChildren(t *testing.T) {
	client, server, mux := setup()
	defer server.Close()

	mux.HandleFunc("/products/1/items/188/children.json", func(w http.ResponseWriter, r *http.Request) {
		ensureMethod(t, r, "GET")
		fmt.Fprint(w, testingTaskSliceString)
	})

	items, _, err := client.Items.ListChildren(1, 188)
	if err != nil {
		t.Errorf("Items.ListChildren failed: %v", err)
		return
	}

	ensureEqual(t, items, testingTaskSlice)
}
