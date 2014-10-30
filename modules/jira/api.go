package jira

import (
	// Stdlib
	"bytes"
	"net/http"
	"net/url"

	// Internal
	"github.com/salsita/salsaflow/modules/jira/client"
)

// API client instantiation ----------------------------------------------------

type BasicAuthRoundTripper struct {
	next http.RoundTripper
}

func (rt *BasicAuthRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	req.SetBasicAuth(config.Username(), config.Password())
	return rt.next.RoundTrip(req)
}

func newClient() *client.Client {
	relativeURL, _ := url.Parse("rest/api/2/")
	baseURL := config.BaseURL().ResolveReference(relativeURL)
	return client.New(baseURL, &http.Client{
		Transport: &BasicAuthRoundTripper{http.DefaultTransport},
	})
}

// Various userful helper functions --------------------------------------------

func listStoriesById(ids []string) ([]*client.Issue, error) {
	var jql bytes.Buffer
	for _, id := range ids {
		if jql.Len() != 0 {
			if _, err := jql.WriteString("OR "); err != nil {
				return nil, err
			}
		}
		if _, err := jql.WriteString("id="); err != nil {
			return nil, err
		}
		if _, err := jql.WriteString(id); err != nil {
			return nil, err
		}
	}

	stories, _, err := newClient().Issues.Search(&client.SearchOptions{
		JQL: jql.String(),
	})
	return stories, err
}
