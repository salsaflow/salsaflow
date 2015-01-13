package sprintly

import (
	"fmt"
	"net/http"
)

// PeopleService holds all the methods for manipulating Sprintly product members.
type PeopleService struct {
	client *Client
}

func newPeopleService(client *Client) *PeopleService {
	return &PeopleService{client}
}

// User represents the Sprintly user resource.
type User struct {
	Id        int    `json:"id,omitempty"`
	Email     string `json:"email,omitempty"`
	FirstName string `json:"first_name,omitempty"`
	LastName  string `json:"last_name,omitempty"`
	Admin     bool   `json:"admin,omitempty"`
	Revoked   bool   `json:"revoked,omitempty"`
}

// List can be used to get all the users associated with the given product.
//
// https://sprintly.uservoice.com/knowledgebase/articles/98410-people
func (srv PeopleService) List(productId int) ([]User, *http.Response, error) {
	u := fmt.Sprintf("products/%v/people.json", productId)

	req, err := srv.client.NewGetRequest(u, nil)
	if err != nil {
		return nil, nil, err
	}

	var people []User
	resp, err := srv.client.Do(req, &people)
	if err != nil {
		return nil, resp, err
	}

	return people, resp, nil
}

// Get can be used to get the user associated with the given user ID.
//
// https://sprintly.uservoice.com/knowledgebase/articles/98410-people
func (srv PeopleService) Get(productId, userId int) (*User, *http.Response, error) {
	u := fmt.Sprintf("products/%v/people/%v.json", productId, userId)

	req, err := srv.client.NewGetRequest(u, nil)
	if err != nil {
		return nil, nil, err
	}

	var user User
	resp, err := srv.client.Do(req, &user)
	if err != nil {
		return nil, resp, err
	}

	return &user, resp, nil
}

// Invite will invite the specified user to the specified product.
//
// https://sprintly.uservoice.com/knowledgebase/articles/98410-people
func (srv PeopleService) Invite(productId int, invitation *Invitation) (*http.Response, error) {
	u := fmt.Sprintf("products/%v/people.json", productId)

	req, err := srv.client.NewPostRequest(u, invitation)
	if err != nil {
		return nil, err
	}

	return srv.client.Do(req, nil)
}

// Invitation represents the arguments that can be supplied
// when inviting a user to join a product.
type Invitation struct {
	Email     string `url:"email,omitempty"      schema:"email,omitempty"`
	FirstName string `url:"first_name,omitempty" schema:"first_name,omitempty"`
	LastName  string `url:"last_name,omitempty"  schema:"last_name,omitempty"`
	Admin     bool   `url:"admin,omitempty"      schema:"admin,omitempty"`
}

// Remove will remote the user from the specified product.
//
// https://sprintly.uservoice.com/knowledgebase/articles/98410-people
func (srv PeopleService) Remove(productId, userId int) (*http.Response, error) {
	u := fmt.Sprintf("products/%v/people/%v.json", productId, userId)

	req, err := srv.client.NewDeleteRequest(u)
	if err != nil {
		return nil, err
	}

	return srv.client.Do(req, nil)
}
