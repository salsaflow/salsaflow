package sprintly

import (
	"encoding/json"
	"net/http"
	"net/url"
)

const (
	LibraryVersion = "0.0.1"

	DefaultBaseURL   = "https://sprint.ly/api/"
	DefaultUserAgent = "go-sprintly/" + LibraryVersion
)

type Client struct {
	// Sprintly username to be used to authenticate API calls.
	username string

	// Sprintly access token to be used to authenticate API calls.
	token string

	// HTTP client to be used for communication with the Sprintly API.
	client *http.Client

	// Base URL of the Sprintly API that is to be used to form endpoint URLs.
	baseURL *url.URL

	// User-Agent header to use when making API calls.
	userAgent string

	// The People service.
	People *PeopleService

	// The Items service.
	Items *ItemsService

	// The Deploys service.
	Deploys *DeploysService
}

// NewClient returns a new API client instance that uses
// the given username and token to authenticate the API calls.
func NewClient(username, token string) *Client {
	baseURL, _ := url.Parse(DefaultBaseURL)
	client := &Client{
		username:  username,
		token:     token,
		client:    http.DefaultClient,
		baseURL:   baseURL,
		userAgent: DefaultUserAgent,
	}
	client.People = newPeopleService(client)
	client.Items = newItemsService(client)
	client.Deploys = newDeploysService(client)
	return client
}

// SetBaseURL can be used to overwrite the default API base URL,
// which is the Sprintly API - https://sprint.ly/api/.
func (c *Client) SetBaseURL(baseURL string) error {
	// Parse the URL.
	u, err := url.Parse(baseURL)
	if err != nil {
		return err
	}

	// Make sure the trailing slash is there.
	if u.Path != "" && u.Path[len(u.Path)-1] != '/' {
		u.Path += "/"
	}

	c.baseURL = u
	return nil
}

// SetUserAgent can be used to overwrite the default user agent string.
func (c *Client) SetUserAgent(agent string) {
	c.userAgent = agent
}

// SetHttpClient can be used to really customize the API client behaviour
// by replacing the underlying HTTP client that is being used to carry out
// all the API calls.
func (c *Client) SetHttpClient(client *http.Client) {
	c.client = client
}

// NewGetRequest returns a new GET API request for the given relative URL.
//
// In case the args object is not nil, it is encoded using github.com/google/go-querystring/query
// and the resulting string is appended to the URL.
func (c *Client) NewGetRequest(urlPath string, args interface{}) (*http.Request, error) {
	path, err := url.Parse(urlPath)
	if err != nil {
		return nil, err
	}

	u := c.baseURL.ResolveReference(path)
	if err := appendArgs(u, args); err != nil {
		return nil, err
	}

	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return nil, err
	}

	req.SetBasicAuth(c.username, c.token)
	req.Header.Set("User-Agent", c.userAgent)
	return req, nil
}

// NewPostRequest returns a new POST API request for the given relative URL and arguments.
//
// In case the args object is not nil, it is encoded using github.com/google/go-querystring/query
// and the resulting string is inserted into the request body. The content type is then set to
// application/x-www-form-urlencoded.
func (c *Client) NewPostRequest(urlPath string, args interface{}) (*http.Request, error) {
	path, err := url.Parse(urlPath)
	if err != nil {
		return nil, err
	}

	u := c.baseURL.ResolveReference(path)

	body, err := encodeArgs(args)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", u.String(), body)
	if err != nil {
		return nil, err
	}

	req.SetBasicAuth(c.username, c.token)
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return req, nil
}

// NewDeleteRequest return a new DELETE API request for the given relative URL.
func (c *Client) NewDeleteRequest(urlPath string) (*http.Request, error) {
	path, err := url.Parse(urlPath)
	if err != nil {
		return nil, err
	}

	u := c.baseURL.ResolveReference(path)

	req, err := http.NewRequest("DELETE", u.String(), nil)
	if err != nil {
		return nil, err
	}

	req.SetBasicAuth(c.username, c.token)
	req.Header.Set("User-Agent", c.userAgent)
	return req, nil
}

// Do carries out the given API request.
//
// In case the interface passed into Do is not nil, it is filled from the response body.
func (c *Client) Do(req *http.Request, v interface{}) (*http.Response, error) {
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode > 299 {
		return resp, &ErrAPI{
			Response: resp,
		}
	}

	if v != nil {
		err = json.NewDecoder(resp.Body).Decode(v)
	}

	return resp, err
}
