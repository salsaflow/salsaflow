package jira

import (
	// Stdlib
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	// Internal
	"github.com/salsaflow/salsaflow/errs"

	// Vendor
	"github.com/salsita/go-jira/v2/jira"
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

func newClient(config *moduleConfig) *jira.Client {
	relativeURL, _ := url.Parse("rest/api/2/")
	serverURL := config.ServerURL.ResolveReference(relativeURL)
	return jira.NewClient(serverURL, &http.Client{
		Transport: &BasicAuthRoundTripper{
			username: config.Username,
			password: config.Password,
			next:     http.DefaultTransport},
	})
}

// Issue operations in parallel ------------------------------------------------

// issueUpdateFunc represents a function that takes an existing story and
// changes it somehow using an API call. It then returns any error encountered.
type issueUpdateFunc func(*jira.Client, *jira.Issue) error

// issueUpdateResult represents what was returned by an issueUpdateFunc.
// It contains the original issue object and the error returned by the update function.
type issueUpdateResult struct {
	issue *jira.Issue
	err   error
}

// updateIssues calls updateFunc on every issue in the list, concurrently.
// It then collects all the results and returns the cumulative result.
func updateIssues(
	api *jira.Client,
	issues []*jira.Issue,
	updateFunc issueUpdateFunc,
	rollbackFunc issueUpdateFunc,
) error {
	// Send all the requests at once.
	retCh := make(chan *issueUpdateResult, len(issues))
	for _, issue := range issues {
		go func(is *jira.Issue) {
			err := updateFunc(api, is)
			retCh <- &issueUpdateResult{is, err}
		}(issue)
	}

	// Wait for the requests to complete.
	var (
		stderr        = bytes.NewBufferString("\nUpdate Errors\n-------------\n")
		updatedIssues = make([]*jira.Issue, 0, len(issues))
		err           error
	)
	for i := 0; i < cap(retCh); i++ {
		if ret := <-retCh; ret.err != nil {
			fmt.Fprintln(stderr, ret.err)
			err = errors.New("failed to update JIRA issues")
		} else {
			updatedIssues = append(updatedIssues, ret.issue)
		}
	}
	fmt.Fprintln(stderr)

	if err != nil {
		if rollbackFunc != nil {
			// Spawn the rollback goroutines.
			retCh := make(chan *issueUpdateResult)
			for _, issue := range updatedIssues {
				go func(is *jira.Issue) {
					err := rollbackFunc(api, is)
					retCh <- &issueUpdateResult{is, err}
				}(issue)
			}

			// Collect the rollback results.
			rollbackStderr := bytes.NewBufferString("Rollback Errors\n---------------\n")
			for _ = range updatedIssues {
				if ret := <-retCh; ret.err != nil {
					fmt.Fprintln(rollbackStderr, ret.err)
				}
			}
			fmt.Fprintln(stderr)

			// Append the rollback error output to the update error output.
			if _, err := io.Copy(stderr, rollbackStderr); err != nil {
				panic(err)
			}
		}
		// Return the aggregate error.
		return errs.NewErrorWithHint("Update JIRA issues", err, stderr.String())
	}
	return nil
}

// Labels ----------------------------------------------------------------------

func newAddLabelFunc(label string) issueUpdateFunc {
	addRequest := jira.M{
		"update": jira.M{
			"labels": jira.L{
				jira.M{
					"add": label,
				},
			},
		},
	}

	return func(api *jira.Client, issue *jira.Issue) error {
		_, err := api.Issues.Update(issue.Id, addRequest)
		return err
	}
}

func newRemoveLabelFunc(label string) issueUpdateFunc {
	removeRequest := jira.M{
		"update": jira.M{
			"labels": jira.L{
				jira.M{
					"remove": label,
				},
			},
		},
	}

	return func(api *jira.Client, issue *jira.Issue) error {
		_, err := api.Issues.Update(issue.Id, removeRequest)
		return err
	}
}

func addLabel(api *jira.Client, issues []*jira.Issue, label string) error {
	return updateIssues(api, issues, newAddLabelFunc(label), newRemoveLabelFunc(label))
}

func removeLabel(api *jira.Client, issues []*jira.Issue, label string) error {
	return updateIssues(api, issues, newRemoveLabelFunc(label), newAddLabelFunc(label))
}

func issuesByLabel(api *jira.Client, projectKey, label string) ([]*jira.Issue, error) {
	query := fmt.Sprintf("project = %v AND labels = \"%v\"", projectKey, label)
	return search(api, query)
}

// Transitions -----------------------------------------------------------------

func performBulkTransition(
	api *jira.Client,
	issues []*jira.Issue,
	transitionId string,
	rollbackTransitionId string,
) error {
	var rollbackFunc issueUpdateFunc
	if rollbackTransitionId != "" {
		rollbackFunc = func(api *jira.Client, issue *jira.Issue) error {
			_, err := api.Issues.PerformTransition(issue.Id, jira.M{
				"transition": jira.M{
					"id": rollbackTransitionId,
				},
			})
			return err
		}
	}

	return updateIssues(api, issues,
		func(api *jira.Client, issue *jira.Issue) error {
			_, err := api.Issues.PerformTransition(issue.Id, jira.M{
				"transition": jira.M{
					"id": transitionId,
				},
			})
			return err
		},
		rollbackFunc)
}

// Various userful helper functions --------------------------------------------

func search(api *jira.Client, query string) ([]*jira.Issue, error) {
	issues, _, err := api.Issues.Search(&jira.SearchOptions{
		JQL: query,
	})
	return issues, err
}

func listStoriesById(api *jira.Client, ids []string) ([]*jira.Issue, error) {
	// In case the list of IDs is empty, just return an empty slice.
	if len(ids) == 0 {
		return nil, nil
	}

	// Fetch the issues from JIRA, return immediately on success.
	issues, err := search(api, issuesQuery(ids))
	if err == nil {
		return issues, err
	}

	// JIRA returns 400 Bad Request when some of the issues are not found.
	// To handle this error, we parse the error messages (not too robust)
	// and we try to send the request again without the IDs that were not found.
	if err, ok := err.(*jira.ErrAPI); ok {
		invalidIdRegexp := regexp.MustCompile(
			"A value with ID '([^']+)' does not exist for the field 'id'.")

		var retry bool
		for _, msg := range err.Err.ErrorMessages {
			groups := invalidIdRegexp.FindStringSubmatch(msg)
			if len(groups) == 2 {
				for i, id := range ids {
					if id == groups[1] {
						ids = append(ids[:i], ids[i+1:]...)
						retry = true
						break
					}
				}
			}
		}

		// Just take a shortcut in case there are no issues left.
		if len(ids) == 0 {
			return nil, err
		}

		if retry {
			issues, ex := search(api, issuesQuery(ids))
			if ex != nil {
				// In case there is an error on retry, return that error.
				return nil, ex
			} else {
				// In case there is no error, return the original error together with
				// the issues that were successfully fetched on retry.
				return issues, err
			}
		}
		return nil, err
	}

	return nil, err
}

func listStoriesByIdOrdered(api *jira.Client, ids []string) ([]*jira.Issue, error) {
	// Fetch the issues.
	issues, err := listStoriesById(api, ids)
	if err != nil {
		return nil, err
	}

	// Order them.
	idMap := make(map[string]*jira.Issue, len(ids))
	keyMap := make(map[string]*jira.Issue, len(ids))
	for _, issue := range issues {
		idMap[issue.Id] = issue
		keyMap[issue.Key] = issue
	}

	ordered := make([]*jira.Issue, 0, len(ids))
	for _, id := range ids {
		if issue, ok := idMap[id]; ok {
			ordered = append(ordered, issue)
			continue
		}

		if issue, ok := keyMap[id]; ok {
			ordered = append(ordered, issue)
			continue
		}

		panic("unreachable code reached")
	}

	return ordered, nil
}

func issuesQuery(ids []string) (queryString string) {
	var query bytes.Buffer
	for _, id := range ids {
		if id == "" {
			panic("bug(id is an empty string)")
		}
		if query.Len() != 0 {
			if _, err := query.WriteString(" OR "); err != nil {
				panic(err)
			}
		}
		if _, err := query.WriteString("id="); err != nil {
			panic(err)
		}
		if _, err := query.WriteString(id); err != nil {
			panic(err)
		}
	}
	return query.String()
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
