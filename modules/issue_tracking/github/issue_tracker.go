package github

import (
	// Stdlib
	"fmt"
	"regexp"
	"strconv"

	// Internal
	"github.com/salsaflow/salsaflow/action"
	"github.com/salsaflow/salsaflow/errs"
	ghutil "github.com/salsaflow/salsaflow/github"
	ghissues "github.com/salsaflow/salsaflow/github/issues"
	"github.com/salsaflow/salsaflow/log"
	"github.com/salsaflow/salsaflow/modules/common"
	"github.com/salsaflow/salsaflow/version"

	// Other
	"github.com/google/go-github/github"
	"github.com/toqueteos/webbrowser"
)

const ServiceName = "GitHub Issues"

type issueTracker struct {
	config *moduleConfig
}

func newIssueTracker() (common.IssueTracker, error) {
	config, err := loadConfig()
	if err != nil {
		return nil, err
	}

	return &issueTracker{config}, nil
}

// ServiceName is a part of common.IssueTracker interface.
func (tracker *issueTracker) ServiceName() string {
	return ServiceName
}

// CurrentUser is a part of common.IssueTracker interface.
func (tracker *issueTracker) CurrentUser() (common.User, error) {
	task := "Get the GitHub user record for the authenticated user"
	client := tracker.newClient()

	var (
		me  *github.User
		err error
	)
	withRequestAllocated(func() {
		me, _, err = client.Users.Get("")
	})
	if err != nil {
		return nil, errs.NewError(task, err)
	}

	return &user{me}, nil
}

// StartableStories is a part of common.IssueTracker interface.
func (tracker *issueTracker) StartableStories() ([]common.Story, error) {
	return tracker.searchIssuesAndWrap(`state:open label:"%v"`, tracker.config.ApprovedLabel)
}

// ReviewableStories is a part of common.IssueTracker interface.
func (tracker *issueTracker) ReviewableStories() ([]common.Story, error) {
	// Prepare the accumulator for GitHub issues.
	var issues []*github.Issue

	// Send the search requests concurrently.
	ch := make(chan *searchResult, 2)
	tracker.goSearchIssues(ch, `state:open label:"%v"`, tracker.config.BeingImplementedLabel)
	tracker.goSearchIssues(ch, `state:open label:"%v"`, tracker.config.ImplementedLabel)

	// Collect the results.
	for i := 0; i < cap(ch); i++ {
		res := <-ch
		// In case there is an error, return immediately,
		// even though there might be some requests still in progress.
		if err := res.err; err != nil {
			return nil, err
		}
		// Append the received issues to the accumulator.
		issues = append(issues, res.issues...)
	}

	// Return the results.
	return toCommonStories(issues, tracker), nil
}

// ReviewedStories is a part of common.IssueTracker interface.
func (tracker *issueTracker) ReviewedStories() (stories []common.Story, err error) {
	return tracker.searchIssuesAndWrap(`state:open label:"%v"`, tracker.config.ReviewedLabel)
}

// ListStoriesByTag is a part of common.IssueTracker interface.
func (tracker *issueTracker) ListStoriesByTag(tags []string) ([]common.Story, error) {
	// Convert tags to ids.
	ids := make([]int, 0, len(tags))
	for _, tag := range tags {
		readableId, err := tracker.StoryTagToReadableStoryId(tag)
		if err != nil {
			return nil, err
		}
		// A readable ID is actually #issueNum, so we split '#' away.
		id, err := strconv.Atoi(readableId[1:])
		if err != nil {
			panic(err)
		}
		ids = append(ids, id)
	}

	// Fetch the relevant stories.
	issues, err := tracker.issuesByNumberOrdered(ids)
	if err != nil {
		return nil, err
	}

	// Convert to []common.Story and return.
	return toCommonStories(issues, tracker), nil
}

// ListStoriesByRelease is a part of common.IssueTracker interface.
func (tracker *issueTracker) ListStoriesByRelease(v *version.Version) ([]common.Story, error) {
	issues, err := tracker.issuesByRelease(v)
	if err != nil {
		return nil, err
	}
	return toCommonStories(issues, tracker), nil
}

// NextRelease is a part of common.IssueTracker interface.
func (tracker *issueTracker) NextRelease(
	trunkVersion *version.Version,
	nextTrunkVersion *version.Version,
) common.NextRelease {

	return newNextRelease(tracker, trunkVersion, nextTrunkVersion)
}

// RunningRelease is a part of common.IssueTracker interface.
func (tracker *issueTracker) RunningRelease(
	releaseVersion *version.Version,
) common.RunningRelease {

	return newRunningRelease(tracker, releaseVersion)
}

// OpenStory is a part of common.IssueTracker interface.
func (tracker *issueTracker) OpenStory(storyId string) error {
	u := fmt.Sprintf("https://github.com/%v/%v/issues/%v",
		tracker.config.GitHubOwner, tracker.config.GitHubRepository, storyId)

	return webbrowser.Open(u)
}

// StoryIdToReadableStoryId is a part of common.IssueTracker interface.
func (tracker *issueTracker) StoryTagToReadableStoryId(tag string) (storyId string, err error) {
	// The format is owner/repo#issueNum
	re := regexp.MustCompile("^[^/]+/[^#]+(#[0-9]+)$")
	match := re.FindStringSubmatch(tag)
	if len(match) == 0 {
		return "", fmt.Errorf("invalid GitHub issue tag: %v", tag)
	}
	return match[1], nil
}

// Utility methods used internally ---------------------------------------------

func (tracker *issueTracker) newClient() *github.Client {
	return ghutil.NewClient(tracker.config.UserToken)
}

type searchResult struct {
	issues []*github.Issue
	err    error
}

// searchIssues can be used to query GitHub for issues matching the given filter.
// It handles pagination internally, it fetches all matching issues automatically.
func (tracker *issueTracker) searchIssues(
	queryFormat string,
	v ...interface{},
) ([]*github.Issue, error) {

	logger := log.V(log.Debug)

	// Format the query.
	query := fmt.Sprintf(queryFormat, v...)

	// Since GH API does not allow OR queries, we need to send a concurrent request
	// for every item in tracker.config.StoryLabels label list.
	ch := make(chan *searchResult, len(tracker.config.StoryLabels))
	for _, label := range tracker.config.StoryLabels {
		go func(label string) {
			// We are only interested in issues for the given repository.
			innerQuery := fmt.Sprintf(`%v type:issue repo:%v/%v label:"%v"`,
				query, tracker.config.GitHubOwner, tracker.config.GitHubRepository, label)

			task := "Search GitHub: " + innerQuery

			if logger {
				logger.Go(task)
			}

			searchOpts := &github.SearchOptions{}
			searchOpts.Page = 1
			searchOpts.PerPage = 50

			var (
				acc      []*github.Issue
				searched int
			)

			client := tracker.newClient()

			for {
				// Fetch another page.
				var (
					result *github.IssuesSearchResult
					err    error
				)
				withRequestAllocated(func() {
					result, _, err = client.Search.Issues(innerQuery, searchOpts)
				})
				if err != nil {
					ch <- &searchResult{nil, errs.NewError(task, err)}
					return
				}

				// Check the issues for exact string match.
				for i := range result.Issues {
					acc = append(acc, &result.Issues[i])
				}

				// Check whether we have reached the end or not.
				searched += len(result.Issues)
				if searched == *result.Total {
					ch <- &searchResult{acc, nil}
					return
				}

				// Check the next page in the next iteration.
				searchOpts.Page += 1
			}
		}(label)
	}

	// Collect the results.
	var issues []*github.Issue
	for i := 0; i < cap(ch); i++ {
		res := <-ch
		if err := res.err; err != nil {
			return nil, err
		}
		issues = append(issues, res.issues...)
	}

	// Make sure there are no duplicates in the list.
	return dedupeIssues(issues), nil
}

// goSearchIssues simply executes searchIssues, but in a background goroutine,
// returning the results on the given channel when available.
func (tracker *issueTracker) goSearchIssues(
	ch chan<- *searchResult,
	queryFormat string,
	v ...interface{},
) {
	go func() {
		issues, err := tracker.searchIssues(queryFormat, v...)
		ch <- &searchResult{issues, err}
	}()
}

// searchIssuesAndWrap combines searchIssues and toCommonStories, that's it.
func (tracker *issueTracker) searchIssuesAndWrap(
	queryFormat string,
	v ...interface{},
) ([]common.Story, error) {

	issues, err := tracker.searchIssues(queryFormat, v...)
	if err != nil {
		return nil, err
	}

	return toCommonStories(issues, tracker), nil
}

type getIssueResult struct {
	issue *github.Issue
	err   error
}

// issuesByNumber fetches the issues matching the given issue numbers.
//
// The issues are fetched concurrently.
func (tracker *issueTracker) issuesByNumber(issueNums []int) ([]*github.Issue, error) {
	task := "Get GitHub issues for the given issue numbers"

	// Prepare an accumulator.
	issues := make([]*github.Issue, 0, len(issueNums))

	// Send the requests concurrently.
	var (
		client = tracker.newClient()
		owner  = tracker.config.GitHubOwner
		repo   = tracker.config.GitHubRepository
	)
	ch := make(chan *getIssueResult, len(issueNums))
	for _, num := range issueNums {
		go func(issueNum int) {
			var (
				issue *github.Issue
				err   error
			)
			withRequestAllocated(func() {
				issue, _, err = client.Issues.Get(owner, repo, issueNum)
			})
			ch <- &getIssueResult{issue, err}
		}(num)
	}

	// Wait for the results to arrive.
	for i := 0; i < cap(ch); i++ {
		res := <-ch
		if err := res.err; err != nil {
			return nil, errs.NewError(task, err)
		}
		issues = append(issues, res.issue)
	}

	// Return the result.
	return issues, nil
}

// issuesByNumberOrdered is almost the same as issuesByNumber except that
// the resulting issue slice is sorted according to the given number list, i.e.
// issues[i] corresponds to issueNums[i] for every index.
func (tracker *issueTracker) issuesByNumberOrdered(issueNums []int) ([]*github.Issue, error) {
	// Fetch the issues, unordered.
	unordered, err := tracker.issuesByNumber(issueNums)
	if err != nil {
		return nil, err
	}

	// Generate a mapping issue number -> issue object.
	m := make(map[int]*github.Issue, len(issueNums))
	for _, issue := range unordered {
		m[*issue.Number] = issue
	}

	// Generate a list of issues where ordered[i] corresponds to issueNums[i].
	ordered := make([]*github.Issue, 0, len(issueNums))
	for _, num := range issueNums {
		issue, ok := m[num]
		if ok {
			ordered = append(ordered, issue)
		} else {
			ordered = append(ordered, nil)
		}
	}

	// Return the ordered list.
	return ordered, nil
}

// issuesByRelease returns the issues assigned to the relevant milestone.
func (tracker *issueTracker) issuesByRelease(v *version.Version) ([]*github.Issue, error) {
	return tracker.searchIssues(`milestone:"%v"`, v.BaseString())
}

// updateIssues just calls updateIssues utility function
// using the client and config as contained in this issueTracker.
func (tracker *issueTracker) updateIssues(
	issues []*github.Issue,
	updateFunc issueUpdateFunc,
	rollbackFunc issueUpdateFunc,
) ([]*github.Issue, action.Action, error) {

	var (
		client = tracker.newClient()
		owner  = tracker.config.GitHubOwner
		repo   = tracker.config.GitHubRepository
	)
	return updateIssues(client, owner, repo, issues, updateFunc, rollbackFunc)
}

// getOrCreateMilestone just calls ghissues.GetOrCreateMilestoneForTitle
// using the client and config as contained in this issueTracker.
func (tracker *issueTracker) getOrCreateMilestone(
	v *version.Version,
) (*github.Milestone, action.Action, error) {

	var (
		client = tracker.newClient()
		owner  = tracker.config.GitHubOwner
		repo   = tracker.config.GitHubRepository
		title  = v.BaseString()

		milestone *github.Milestone
		act       action.Action
		err       error
	)
	withRequestAllocated(func() {
		milestone, act, err = ghissues.GetOrCreateMilestoneForTitle(client, owner, repo, title)
	})
	return milestone, act, err
}

func (tracker *issueTracker) closeMilestone(
	v *version.Version,
) (m *github.Milestone, act action.Action, err error) {

	// Use a chain to group the actions.
	chain := action.NewActionChain()
	defer chain.RollbackOnError(&err)

	// Get the associated milestone.
	milestone, act, err := tracker.getOrCreateMilestone(v)
	if err != nil {
		return nil, nil, err
	}
	chain.Push(act)

	// Mark it as closed.
	var (
		client = tracker.newClient()
		owner  = tracker.config.GitHubOwner
		repo   = tracker.config.GitHubRepository
	)
	withRequestAllocated(func() {
		milestone, act, err = ghissues.CloseMilestone(client, owner, repo, milestone)
	})
	if err != nil {
		return nil, nil, err
	}
	chain.Push(act)

	// Return the results.
	return milestone, chain, nil
}
