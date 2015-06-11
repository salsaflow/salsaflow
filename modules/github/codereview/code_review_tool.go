package github

import (
	// Stdlib
	"bytes"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"

	// Internal
	"github.com/salsaflow/salsaflow/action"
	"github.com/salsaflow/salsaflow/errs"
	"github.com/salsaflow/salsaflow/git"
	ghutil "github.com/salsaflow/salsaflow/github"
	"github.com/salsaflow/salsaflow/log"
	"github.com/salsaflow/salsaflow/modules/common"
	"github.com/salsaflow/salsaflow/version"

	// Other
	"github.com/google/go-github/github"
	"github.com/toqueteos/webbrowser"
)

const Id = "github"

var errPostReviewRequest = errors.New("failed to post a review request")

type codeReviewTool struct{}

func Factory() (common.CodeReviewTool, error) {
	return &codeReviewTool{}, nil
}

func (tool *codeReviewTool) InitialiseRelease(v *version.Version) (action.Action, error) {
	// Get a GitHub client.
	config, err := LoadConfig()
	if err != nil {
		return nil, err
	}
	client := ghutil.NewClient(config.Token())

	owner, repo, err := parseUpstreamURL()
	if err != nil {
		return nil, err
	}

	// Create the review milestone.
	milestoneTask := fmt.Sprintf("Create code review milestone for release %v", v)
	log.Run(milestoneTask)
	milestone, _, err := client.Issues.CreateMilestone(owner, repo, &github.Milestone{
		Title: github.String(milestoneTitle(v)),
	})
	if err != nil {
		return nil, errs.NewError(milestoneTask, err, nil)
	}

	// Return a rollback function.
	return action.ActionFunc(func() error {
		log.Rollback(milestoneTask)
		task := fmt.Sprintf("Delete code review milestone '%v'", *milestone.Title)
		_, err := client.Issues.DeleteMilestone(owner, repo, *milestone.Number)
		if err != nil {
			return errs.NewError(task, err, nil)
		}
		return nil
	}), nil
}

func (tool *codeReviewTool) FinaliseRelease(v *version.Version) (action.Action, error) {
	// Get a GitHub client.
	config, err := LoadConfig()
	if err != nil {
		return nil, err
	}
	client := ghutil.NewClient(config.Token())

	owner, repo, err := parseUpstreamURL()
	if err != nil {
		return nil, err
	}

	// Get the relevant review milestone.
	task := fmt.Sprintf("Get code review milestone for release %v", v)
	log.Run(task)
	milestone, err := milestoneForVersion(config, owner, repo, v)
	if err != nil {
		if _, ok := err.(*ErrMilestoneNotFound); ok {
			log.Warn("Weird, " + err.Error())
			return action.ActionFunc(func() error { return nil }), nil
		}
		return nil, errs.NewError(task, err, nil)
	}

	// Close the milestone unless there are some issues open.
	task = fmt.Sprintf("Make sure review milestone for release %v can be closed", v)
	if num := *milestone.OpenIssues; num != 0 {
		return nil, errs.NewError(task,
			fmt.Errorf("review milestone for release %v cannot be closed: %v issue(s) open", v, num), nil)
	}

	milestoneTask := fmt.Sprintf("Close review milestone for release %v", v)
	log.Run(milestoneTask)
	milestone, _, err = client.Issues.EditMilestone(owner, repo, *milestone.Number, &github.Milestone{
		State: github.String("closed"),
	})
	if err != nil {
		return nil, errs.NewError(milestoneTask, err, nil)
	}

	// Return a rollback function.
	return action.ActionFunc(func() error {
		log.Rollback(milestoneTask)
		task := fmt.Sprintf("Reopen review milestone for release %v", v)
		_, _, err := client.Issues.EditMilestone(owner, repo, *milestone.Number, &github.Milestone{
			State: github.String("open"),
		})
		if err != nil {
			return errs.NewError(task, err, nil)
		}
		return nil
	}), nil
}

func (tool *codeReviewTool) PostReviewRequests(
	ctxs []*common.ReviewContext,
	opts map[string]interface{},
) (err error) {

	// Load the GitHub config.
	config, err := LoadConfig()
	if err != nil {
		return err
	}

	// Get the GitHub owner and repository from the upstream URL.
	owner, repo, err := parseUpstreamURL()
	if err != nil {
		return err
	}

	// Group commits by story ID.
	//
	// In case the commit is associated with a story, we add it to the relevant story group.
	// Otherwise the commit is marked as unassigned and added to the relevant list.
	var (
		ctxsByStoryId     = make(map[string][]*common.ReviewContext, 1)
		unassignedCommits = make([]*git.Commit, 0, 1)
	)
	for _, ctx := range ctxs {
		story := ctx.Story
		if story != nil {
			sid := story.ReadableId()
			rcs, ok := ctxsByStoryId[sid]
			if ok {
				rcs = append(rcs, ctx)
			} else {
				rcs = []*common.ReviewContext{ctx}
			}
			ctxsByStoryId[sid] = rcs
		} else {
			unassignedCommits = append(unassignedCommits, ctx.Commit)
		}
	}

	// Post the assigned commits.
	for _, ctxs := range ctxsByStoryId {
		var (
			story   = ctxs[0].Story
			commits = make([]*git.Commit, 0, len(ctxs))
		)
		for _, ctx := range ctxs {
			commits = append(commits, ctx.Commit)
		}
		if ex := postAssignedReviewRequest(config, owner, repo, story, commits, opts); ex != nil {
			errs.Log(ex)
			err = errPostReviewRequest
		}
	}

	// Post the unassigned commits.
	for _, commit := range unassignedCommits {
		if ex := postUnassignedReviewRequest(config, owner, repo, commit, opts); ex != nil {
			errs.Log(ex)
			err = errPostReviewRequest
		}
	}

	return
}

func (tool *codeReviewTool) PostReviewFollowupMessage() string {
	return `
GitHub review issues successfully created.

Please visit the issues that have been created and assign a reviewer.
Annotate and explain the changes to make the reviewer's job easier.

In case there are any review issues raised for a story review issue,
just keep adding commits to that review issue. That will happen
automatically when the same Story-Id tag is used.

In case there are any review issues raised for an unassigned commit
review issue, use

    $ salsaflow review post -fixes=ISSUE_NUMBER

to create a new GitHub review issue that references ISSUE_NUMBER.
`
}

// parseUpstreamURL parses the URL of the git upstream being used by SalsaFlow
// and returns the given GitHub owner and repository.
func parseUpstreamURL() (owner, repo string, err error) {
	// Load the Git config.
	gitConfig, err := git.LoadConfig()
	if err != nil {
		return "", "", err
	}
	remoteName := gitConfig.RemoteName()

	// Get the upstream URL.
	task := fmt.Sprintf("Get URL for git remote '%v'", remoteName)
	remoteURL, err := git.GetConfigString(fmt.Sprintf("remote.%v.url", remoteName))
	if err != nil {
		return "", "", errs.NewError(task, err, nil)
	}

	// Parse the upstream URL to get the owner and repo name.
	task = "Parse the upstream repository URL"
	u, err := url.Parse(remoteURL)
	if err != nil {
		return "", "", errs.NewError(task, err, nil)
	}

	var match []string
	if u.Scheme == "https" {
		// Handle HTTPS.
		re := regexp.MustCompile("/([^/]+)/(.+)")
		match = re.FindStringSubmatch(u.Path)
	} else {
		// Handle SSH.
		re := regexp.MustCompile("git@github.com:([^/]+)/(.+)[.]git")
		match = re.FindStringSubmatch(remoteURL)
	}
	if len(match) != 3 {
		err := fmt.Errorf("failed to parse git remote URL: %v", remoteURL)
		return "", "", errs.NewError(task, err, nil)
	}
	return match[1], match[2], nil
}

// postAssignedReviewRequest can be used to post
// the commits associated with the given story for review.
func postAssignedReviewRequest(
	config Config,
	owner string,
	repo string,
	story common.Story,
	commits []*git.Commit,
	opts map[string]interface{},
) error {

	// Search for an existing review issue for the given story.
	task := fmt.Sprintf("Search for an existing review issue for story %v", story.ReadableId())
	log.Run(task)

	query := fmt.Sprintf(
		"\"Review story %v\" repo:%v/%v label:%v type:issue in:title",
		story.ReadableId(), owner, repo, config.ReviewLabel())

	client := ghutil.NewClient(config.Token())
	result, _, err := client.Search.Issues(query, &github.SearchOptions{})
	if err != nil {
		return errs.NewError(task, err, nil)
	}

	// Decide what to do next based on the search results.
	switch len(result.Issues) {
	case 0:
		// No review issue found for the given story, create a new issue.
		return createAssignedReviewRequest(config, owner, repo, story, commits, opts)
	case 1:
		// An existing review issue found, extend it.
		return extendReviewRequest(config, owner, repo, &result.Issues[0], commits, opts)
	default:
		// Multiple review issue found for the given story, that is clearly wrong
		// since there is always just a single review issue for every story.
		err := errors.New("inconsistency detected: multiple story review issues found")
		return errs.NewError("Make sure the review issue can be created", err, nil)
	}
}

// createAssignedReviewRequest can be used to create a new review issue
// for the given commits that is associated with the story passed in.
func createAssignedReviewRequest(
	config Config,
	owner string,
	repo string,
	story common.Story,
	commits []*git.Commit,
	opts map[string]interface{},
) error {

	var (
		task       = fmt.Sprintf("Create review issue for story %v", story.ReadableId())
		issueTitle = fmt.Sprintf("Review story %v: %v", story.ReadableId(), story.Title())
	)

	// Generate the issue body.
	var issueBody bytes.Buffer
	fmt.Fprintf(&issueBody, "Story being reviewed: [%v](%v)\n\n", story.ReadableId(), story.URL())
	fmt.Fprintf(&issueBody, "SF-Issue-Tracker: %v\n", story.IssueTracker().ServiceName())
	fmt.Fprintf(&issueBody, "SF-Story-Key: %v\n\n", story.Tag())
	fmt.Fprintf(&issueBody, "The associated commits are following:")
	for _, commit := range commits {
		fmt.Fprintf(&issueBody, "\n- [ ] %v: %v", commit.SHA, commit.MessageTitle)
	}

	// Get the right review milestone to add the issue into.
	milestone, err := milestoneForCommit(config, owner, repo, commits[len(commits)-1].SHA)
	if err != nil {
		return err
	}

	// Create a new review issue.
	issue, err := createIssue(task, config, owner, repo, issueTitle, issueBody.String(), milestone)
	if err != nil {
		return err
	}

	// Open the issue if requested.
	if _, open := opts["open"]; open {
		return openIssue(issue)
	}
	return nil
}

// postUnassignedReviewRequest can be used to post the given commit for review.
// This function is to be used to post commits that are not associated with any story.
func postUnassignedReviewRequest(
	config Config,
	owner string,
	repo string,
	commit *git.Commit,
	opts map[string]interface{},
) error {

	// Extend the specified review issue in case -fixes is specified.
	flagFixes, ok := opts["fixes"]
	if ok {
		if fixes, ok := flagFixes.(uint); ok && fixes != 0 {
			return extendUnassignedReviewRequest(config, owner, repo, int(fixes), commit, opts)
		}
	}

	// Search for an existing issue.
	task := fmt.Sprintf("Search for an existing review issue for commit %v", commit.SHA)
	log.Run(task)

	query := fmt.Sprintf(
		"\"Review commit %v\" repo:%v/%v label:%v type:issue in:title",
		commit.SHA, owner, repo, config.ReviewLabel())

	client := ghutil.NewClient(config.Token())
	result, _, err := client.Search.Issues(query, &github.SearchOptions{})
	if err != nil {
		return errs.NewError(task, err, nil)
	}

	// Decide what to do next based on the search results.
	switch len(result.Issues) {
	case 0:
		// Create a new unassigned review request.
		return createUnassignedReviewRequest(config, owner, repo, commit, opts)
	case 1:
		// The issues already exists, return an error.
		issueNum := *result.Issues[0].Number
		err := fmt.Errorf("existing review issue found for commit %v: %v", commit.SHA, issueNum)
		return errs.NewError("Make sure the review issue can be created", err, nil)
	default:
		// Inconsistency detected: multiple review issues found.
		err := fmt.Errorf(
			"inconsistency detected: multiple review issue found for commit %v", commit.SHA)
		return errs.NewError("Make sure the review issue can be created", err, nil)
	}
}

// createUnassignedReviewRequest created a new review issue
// for the given commit that is not associated with any story.
func createUnassignedReviewRequest(
	config Config,
	owner string,
	repo string,
	commit *git.Commit,
	opts map[string]interface{},
) error {

	// Generate the issue title and body.
	var (
		task       = fmt.Sprintf("Create review issue for commit %v", commit.SHA)
		issueTitle = fmt.Sprintf("Review commit %v: %v", commit.SHA, commit.MessageTitle)
		issueBody  = fmt.Sprintf("Commits to be reviewed:\n- [ ] %v: %v",
			commit.SHA, commit.MessageTitle)
	)

	// Get the right review milestone to add the issue into.
	milestone, err := milestoneForCommit(config, owner, repo, commit.SHA)
	if err != nil {
		return err
	}

	// Create a new review issue.
	issue, err := createIssue(task, config, owner, repo, issueTitle, issueBody, milestone)
	if err != nil {
		return errs.NewError(task, err, nil)
	}

	// Open the issue if requested.
	if _, open := opts["open"]; open {
		return openIssue(issue)
	}
	return nil
}

// extendUnassignedReviewRequest can be used to upload fixes for
// the specified unassigned review issue.
func extendUnassignedReviewRequest(
	config Config,
	owner string,
	repo string,
	issueNum int,
	commit *git.Commit,
	opts map[string]interface{},
) error {

	// Fetch the issue.
	task := fmt.Sprintf("Fetch GitHub issue #%v", issueNum)
	log.Run(task)
	client := ghutil.NewClient(config.Token())
	issue, _, err := client.Issues.Get(owner, repo, issueNum)
	if err != nil {
		return errs.NewError(task, err, nil)
	}

	// Extend the given review issue.
	return extendReviewRequest(config, owner, repo, issue, []*git.Commit{commit}, opts)
}

// extendReviewRequest is a general function that can be used to extend
// the given review issue with the given list of commits.
func extendReviewRequest(
	config Config,
	owner string,
	repo string,
	issue *github.Issue,
	commits []*git.Commit,
	opts map[string]interface{},
) error {

	var (
		issueNum     = *issue.Number
		issueBody    = *issue.Body
		bodyBuffer   = bytes.NewBufferString(issueBody)
		addedCommits = make([]*git.Commit, 0, len(commits))
	)

	for _, commit := range commits {
		// Make sure the commit is not added yet.
		commitString := fmt.Sprintf("] %v: %v", commit.SHA, commit.MessageTitle)
		if strings.Contains(issueBody, commitString) {
			log.Log(fmt.Sprintf("Commit %v already listed in issue #%v", commit.SHA, issueNum))
			continue
		}

		// Extend the issue body.
		addedCommits = append(addedCommits, commit)
		fmt.Fprintf(bodyBuffer, "\n- [ ] %v: %v", commit.SHA, commit.MessageTitle)
	}

	if len(addedCommits) == 0 {
		log.Log(fmt.Sprintf("All commits already listed in issue #%v", issueNum))
		return nil
	}

	// Edit the issue.
	task := fmt.Sprintf("Update GitHub issue #%v", issueNum)
	log.Run(task)

	client := ghutil.NewClient(config.Token())
	newIssue, _, err := client.Issues.Edit(owner, repo, issueNum, &github.IssueRequest{
		Body:  github.String(bodyBuffer.String()),
		State: github.String("open"),
	})
	if err != nil {
		return errs.NewError(task, err, nil)
	}

	// Add the review comment.
	if err := addReviewComment(config, owner, repo, issueNum, addedCommits); err != nil {
		return err
	}

	// Open the issue if requested.
	if _, open := opts["open"]; open {
		return openIssue(newIssue)
	}
	return nil
}

func addReviewComment(
	config Config,
	owner string,
	repo string,
	issueNum int,
	commits []*git.Commit,
) error {

	// Generate the comment body.
	buffer := bytes.NewBufferString("The following commits were added to this issue:")
	for _, commit := range commits {
		fmt.Fprintf(buffer, "\n* %v: %v", commit.SHA, commit.MessageTitle)
	}

	// Call GitHub API.
	task := fmt.Sprintf("Add review comment for issue #%v", issueNum)
	client := ghutil.NewClient(config.Token())
	_, _, err := client.Issues.CreateComment(owner, repo, issueNum, &github.IssueComment{
		Body: github.String(buffer.String()),
	})
	if err != nil {
		return errs.NewError(task, err, nil)
	}
	return nil
}

func openIssue(issue *github.Issue) error {
	task := fmt.Sprintf("Open issue #%v in the browser", *issue.Number)
	if err := webbrowser.Open(*issue.HTMLURL); err != nil {
		return errs.NewError(task, err, nil)
	}
	return nil
}

func createIssue(
	task string,
	config Config,
	owner string,
	repo string,
	issueTitle string,
	issueBody string,
	milestone *github.Milestone,
) (issue *github.Issue, err error) {

	log.Run(task)
	client := ghutil.NewClient(config.Token())
	labels := []string{config.ReviewLabel()}
	issue, _, err = client.Issues.Create(owner, repo, &github.IssueRequest{
		Title:     github.String(issueTitle),
		Body:      github.String(issueBody),
		Labels:    &labels,
		Milestone: milestone.Number,
	})
	if err != nil {
		return nil, errs.NewError(task, err, nil)
	}

	log.Log(fmt.Sprintf("GitHub issue #%v created", *issue.Number))
	return issue, nil
}

func milestoneForVersion(
	config Config,
	owner string,
	repo string,
	v *version.Version,
) (*github.Milestone, error) {

	// Fetch milestones for the given repository.
	var (
		task   = fmt.Sprintf("Fetch GitHub milestones for %v/%v", owner, repo)
		client = ghutil.NewClient(config.Token())
		title  = milestoneTitle(v)
	)
	milestones, _, err := client.Issues.ListMilestones(owner, repo, nil)
	if err != nil {
		return nil, errs.NewError(task, err, nil)
	}

	// Find the right one.
	task = fmt.Sprintf("Find review milestone for release %v", v)
	for _, milestone := range milestones {
		if *milestone.Title == title {
			return &milestone, nil
		}
	}
	return nil, &ErrMilestoneNotFound{v}
}

func milestoneForCommit(config Config, owner, repo, sha string) (*github.Milestone, error) {
	// Get the version associated with the given commit.
	v, err := version.GetByBranch(sha)
	if err != nil {
		return nil, err
	}

	// Return the associated milestone.
	return milestoneForVersion(config, owner, repo, v)
}

func milestoneTitle(v *version.Version) string {
	return fmt.Sprintf("%v-review", v.BaseString())
}
