package sprintly

import (
	"fmt"
	"net/http"
	"testing"
)

var testingInvitation = Invitation{
	Email:     "joe@joestump.net",
	FirstName: "Joe",
	LastName:  "Stump",
	Admin:     true,
}

func TestPeople_List(t *testing.T) {
	client, server, mux := setup()
	defer server.Close()

	mux.HandleFunc("/products/1/people.json", func(w http.ResponseWriter, r *http.Request) {
		ensureMethod(t, r, "GET")
		fmt.Fprint(w, testingUserSliceJson)
	})

	users, _, err := client.People.List(1)
	if err != nil {
		t.Errorf("People.List failed: %v", err)
		return
	}

	ensureEqual(t, users, testingUserSlice)
}

func TestPeople_Get(t *testing.T) {
	client, server, mux := setup()
	defer server.Close()

	mux.HandleFunc("/products/1/people/1.json", func(w http.ResponseWriter, r *http.Request) {
		ensureMethod(t, r, "GET")
		fmt.Fprint(w, testingUserJson)
	})

	user, _, err := client.People.Get(1, 1)
	if err != nil {
		t.Errorf("People.Get failed: %v", err)
		return
	}

	ensureEqual(t, user, &testingUser)
}

func TestPeople_Invite(t *testing.T) {
	client, server, mux := setup()
	defer server.Close()

	mux.HandleFunc("/products/1/people.json", func(w http.ResponseWriter, r *http.Request) {
		ensureMethod(t, r, "POST")

		var received Invitation
		if err := decodeArgs(&received, r); err != nil {
			t.Error(err)
			return
		}

		ensureEqual(t, &received, &testingInvitation)
	})

	_, err := client.People.Invite(1, &testingInvitation)
	if err != nil {
		t.Errorf("People.Invite failed: %v", err)
		return
	}
}

func TestPeople_Remove(t *testing.T) {
	client, server, mux := setup()
	defer server.Close()

	mux.HandleFunc("/products/1/people/1.json", func(w http.ResponseWriter, r *http.Request) {
		ensureMethod(t, r, "DELETE")
	})

	_, err := client.People.Remove(1, 1)
	if err != nil {
		t.Errorf("People.Invite failed: %v", err)
		return
	}
}
