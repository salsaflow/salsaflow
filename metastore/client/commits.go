package client

import (
	"fmt"
	"net/http"
	"regexp"
)

type CommitData struct {
	SHA           string                 `json:"commit_sha"`
	Story         map[string]interface{} `json:"story,omitempty"`
	ReviewRequest map[string]interface{} `json:"review_request,omitempty"`
}

type CommitService struct {
	client *Client
}

func newCommitService(c *Client) *CommitService {
	return &CommitService{c}
}

func (srv *CommitService) Get(sha string) (*CommitData, *http.Response, error) {
	// Check the SHA.
	if err := checkSHA(sha); err != nil {
		return nil, nil, err
	}

	// Prepare the HTTP request.
	u := fmt.Sprintf("commits/%v", sha)
	req, err := srv.client.NewRequest("GET", u, nil)
	if err != nil {
		return nil, nil, err
	}

	// Send the HTTP request.
	var data CommitData
	resp, err := srv.client.Do(req, &data)
	if err != nil {
		return nil, nil, err
	}
	return &data, resp, nil
}

func (srv *CommitService) Post(data []*CommitData) (*http.Response, error) {
	// Check commit SHAs.
	for _, commit := range data {
		if err := checkSHA(commit.SHA); err != nil {
			return nil, err
		}
	}

	// Prepare the HTTP request.
	req, err := srv.client.NewRequest("POST", "commits", data)
	if err != nil {
		return nil, err
	}

	// Send the HTTP request.
	resp, err := srv.client.Do(req, nil)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func checkSHA(sha string) error {
	if !regexp.MustCompile("^[0-9a-f]{40}$").MatchString(sha) {
		return fmt.Errorf("invalid commit hash: %v", sha)
	}
	return nil
}
