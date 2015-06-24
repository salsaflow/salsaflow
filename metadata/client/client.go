package client

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"

	"github.com/salsaflow/salsaflow/app/metadata"
)

const (
	defaultUserAgent = "salsaflow/" + metadata.Version
)

type Client struct {
	// Access token.
	token string

	// HTTP client to be used to accessing salsaflow-metadata.
	client *http.Client

	// Base URL for salsaflow-metadata.
	baseURL *url.URL

	// User-Agent header to use when accessing salsaflow-metadata.
	userAgent string

	// Commit service.
	Commits *CommitService
}

func New(baseURL, token string) (*Client, error) {
	client := &Client{
		token:     token,
		client:    http.DefaultClient,
		userAgent: defaultUserAgent,
	}
	if err := client.SetBaseURL(baseURL); err != nil {
		return nil, err
	}

	client.Commits = newCommitService(client)
	return client, nil
}

func (c *Client) SetBaseURL(baseURL string) error {
	u, err := url.Parse(baseURL)
	if err != nil {
		return err
	}

	if u.Path != "" && u.Path[len(u.Path)-1] != '/' {
		return ErrNoTrailingSlash
	}

	c.baseURL = u
	return nil
}

func (c *Client) SetUserAgent(agent string) {
	c.userAgent = agent
}

func (c *Client) NewRequest(method, urlPath string, body interface{}) (*http.Request, error) {
	path, err := url.Parse(urlPath)
	if err != nil {
		return nil, err
	}

	u := c.baseURL.ResolveReference(path)

	var bodyBuffer bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&bodyBuffer).Encode(body); err != nil {
			return nil, err
		}
	}

	req, err := http.NewRequest(method, u.String(), &bodyBuffer)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.token)
	return req, nil
}

func (c *Client) Do(req *http.Request, v interface{}) (*http.Response, error) {
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode > 299 {
		return nil, &ErrAPI{
			Response: resp,
		}
	}

	if v != nil {
		if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
			return nil, err
		}
	}

	return resp, nil
}
