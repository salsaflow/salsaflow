package jira

import (
	// Stdlib
	"bytes"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	// Internal
	"github.com/salsita/salsaflow/modules/jira/client"
)

// API client instantiation ----------------------------------------------------

type BasicAuthRoundTripper struct {
	username string
	password string
	next     http.RoundTripper
}

func (rt *BasicAuthRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	req.SetBasicAuth(rt.username, rt.password)
	return rt.next.RoundTrip(req)
}

func newClient(tracker *issueTracker) *client.Client {
	relativeURL, _ := url.Parse("rest/api/2/")
	baseURL := tracker.config.BaseURL().ResolveReference(relativeURL)
	return client.New(baseURL, &http.Client{
		Transport: &BasicAuthRoundTripper{
			username: tracker.config.Username(),
			password: tracker.config.Password(),
			next:     http.DefaultTransport},
	})
}

// Various userful helper functions --------------------------------------------

func listStoriesById(tracker *issueTracker, ids []string) ([]*client.Issue, error) {
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

	stories, _, err := newClient(tracker).Issues.Search(&client.SearchOptions{
		JQL: jql.String(),
	})
	return stories, err
}

// formatInRange takes the arguments and creates a JQL IN query for them, i.e.
//
//    formatInRange("status", "1", "2", "3")
//
// will return
//
//    "(status in (1,2,3))"
func formatInRange(ident string, ids ...string) string {
	if len(ids) == 0 {
		return ""
	}
	return fmt.Sprintf("(%s in (%s))", ident, strings.Join(ids, ","))
}
