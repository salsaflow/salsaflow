package jira

import (
	// Stdlib
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	// Internal
	"github.com/salsita/salsaflow/errs"
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

func newClient(config Config) *client.Client {
	relativeURL, _ := url.Parse("rest/api/2/")
	baseURL := config.BaseURL().ResolveReference(relativeURL)
	return client.New(baseURL, &http.Client{
		Transport: &BasicAuthRoundTripper{
			username: config.Username(),
			password: config.Password(),
			next:     http.DefaultTransport},
	})
}

// Issue operations in parallel ------------------------------------------------

// issueUpdateFunc represents a function that takes an existing story and
// changes it somehow using an API call. It then returns any error encountered.
type issueUpdateFunc func(*client.Client, *client.Issue) error

// issueUpdateResult represents what was returned by an issueUpdateFunc.
// It contains the original issue object and the error returned by the update function.
type issueUpdateResult struct {
	issue *client.Issue
	err   error
}

// updateIssues calls updateFunc on every issue in the list, concurrently.
// It then collects all the results and returns the cumulative result.
func updateIssues(
	api *client.Client,
	issues []*client.Issue,
	updateFunc issueUpdateFunc,
	rollbackFunc issueUpdateFunc,
) error {
	// Send all the request at once.
	retCh := make(chan *issueUpdateResult, len(issues))
	for _, issue := range issues {
		go func(is *client.Issue) {
			// Call the update function.
			err := updateFunc(api, is)
			retCh <- &issueUpdateResult{is, err}
		}(issue)
	}

	// Wait for the requests to complete.
	var (
		stderr         = bytes.NewBufferString("\nUpdate Errors\n-------------\n")
		rollbackStderr = bytes.NewBufferString("\nRollback Errors\n---------------\n")
		rollbackRetCh  = make(chan *issueUpdateResult)
		numThreads     int
		err            error
	)
	for i := 0; i < cap(retCh); i++ {
		if ret := <-retCh; ret.err != nil {
			fmt.Fprintln(stderr, ret.err)
			err = errors.New("failed to update JIRA issues")
			// If the rollback function is available, spawn it now.
			if rollbackFunc != nil {
				numThreads++
				go func(is *client.Issue) {
					err := rollbackFunc(api, is)
					rollbackRetCh <- &issueUpdateResult{is, err}
				}(ret.issue)
			}
		}
	}

	if err != nil {
		// Collect the rollback results.
		if rollbackFunc != nil {
			for i := 0; i < numThreads; i++ {
				if ret := <-rollbackRetCh; ret.err != nil {
					fmt.Fprintln(rollbackStderr, ret.err)
				}
			}
			// Append the rollback error output to the update error output.
			if _, err := io.Copy(stderr, rollbackStderr); err != nil {
				panic(err)
			}
		}
		// Return the aggregate error.
		return errs.NewError("Update JIRA issues", err, stderr)
	}
	return nil
}

// Versions --------------------------------------------------------------------

func assignIssuesToVersion(api *client.Client, issues []*client.Issue, versionId string) error {
	// The payload is the same for all the issue updates.
	addRequest := client.M{
		"update": client.M{
			"fixVersions": client.L{
				client.M{
					"add": &client.Version{
						Id: versionId,
					},
				},
			},
		},
	}

	// Rollback request is used when we want to delete the version again.
	removeRequest := client.M{
		"update": client.M{
			"fixVersions": client.L{
				client.M{
					"remove": &client.Version{
						Id: versionId,
					},
				},
			},
		},
	}

	// Update all the issues concurrently and return the result.
	return updateIssues(api, issues,
		func(api *client.Client, issue *client.Issue) error {
			_, err := api.Issues.Update(issue.Id, addRequest)
			return err
		},
		func(api *client.Client, issue *client.Issue) error {
			_, err := api.Issues.Update(issue.Id, removeRequest)
			return err
		})
}

// Transitions -----------------------------------------------------------------

func performBulkTransition(
	api *client.Client,
	issues []*client.Issue,
	transitionId string,
	rollbackTransitionId string,
) error {
	var rollbackFunc issueUpdateFunc
	if rollbackTransitionId != "" {
		rollbackFunc = func(api *client.Client, issue *client.Issue) error {
			_, err := api.Issues.PerformTransition(issue.Id, rollbackTransitionId)
			return err
		}
	}

	return updateIssues(api, issues,
		func(api *client.Client, issue *client.Issue) error {
			_, err := api.Issues.PerformTransition(issue.Id, transitionId)
			return err
		},
		rollbackFunc)
}

// Various userful helper functions --------------------------------------------

func search(api *client.Client, query string) ([]*client.Issue, error) {
	issues, _, err := api.Issues.Search(&client.SearchOptions{
		JQL: query,
	})
	return issues, err
}

func listStoriesById(api *client.Client, ids []string) ([]*client.Issue, error) {
	var query bytes.Buffer
	for _, id := range ids {
		if id == "" {
			panic("bug(id is an empty string)")
		}
		if query.Len() != 0 {
			if _, err := query.WriteString(" OR "); err != nil {
				return nil, err
			}
		}
		if _, err := query.WriteString("id="); err != nil {
			return nil, err
		}
		if _, err := query.WriteString(id); err != nil {
			return nil, err
		}
	}

	return search(api, query.String())
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
