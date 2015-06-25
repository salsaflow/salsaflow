package github

import (
	// Stdlib
	"bytes"
	"errors"
	"fmt"
	"strings"

	// Internal
	"github.com/salsaflow/salsaflow/action"
	"github.com/salsaflow/salsaflow/errs"
	"github.com/salsaflow/salsaflow/git"
	ghutil "github.com/salsaflow/salsaflow/github"
	"github.com/salsaflow/salsaflow/log"
	"github.com/salsaflow/salsaflow/metastore"
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
	// Get necessary config.
	config, err := LoadConfig()
	if err != nil {
		return nil, err
	}

	owner, repo, err := git.ParseUpstreamURL()
	if err != nil {
		return nil, err
	}

	// Create the review milestone.
	_, act, err := createMilestone(config, owner, repo, v)
	return act, err
}

func (tool *codeReviewTool) FinaliseRelease(v *version.Version) (action.Action, error) {
	// Get a GitHub client.
	config, err := LoadConfig()
	if err != nil {
		return nil, err
	}
	client := ghutil.NewClient(config.Token())

	owner, repo, err := git.ParseUpstreamURL()
	if err != nil {
		return nil, err
	}

	// Get the relevant review milestone.
	releaseString := v.BaseString()
	task := fmt.Sprintf("Get code review milestone for release %v", releaseString)
	log.Run(task)
	milestone, err := milestoneForVersion(config, owner, repo, v)
	if err != nil {
		if _, ok := errs.RootCause(err).(*ErrMilestoneNotFound); ok {
			log.Warn("Weird, " + err.Error())
			return action.ActionFunc(func() error { return nil }), nil
		}
		return nil, errs.NewError(task, err)
	}

	// Close the milestone unless there are some issues open.
	task = fmt.Sprintf("Make sure review milestone for release %v can be closed", releaseString)
	if num := *milestone.OpenIssues; num != 0 {
		return nil, errs.NewError(
			task,
			fmt.Errorf(
				"review milestone for release %v cannot be closed: %v issue(s) open",
				releaseString, num))
	}

	milestoneTask := fmt.Sprintf("Close review milestone for release %v", releaseString)
	log.Run(milestoneTask)
	milestone, _, err = client.Issues.EditMilestone(owner, repo, *milestone.Number, &github.Milestone{
		State: github.String("closed"),
	})
	if err != nil {
		return nil, errs.NewError(milestoneTask, err)
	}

	// Return a rollback function.
	return action.ActionFunc(func() error {
		log.Rollback(milestoneTask)
		task := fmt.Sprintf("Reopen review milestone for release %v", releaseString)
		_, _, err := client.Issues.EditMilestone(owner, repo, *milestone.Number, &github.Milestone{
			State: github.String("open"),
		})
		if err != nil {
			return errs.NewError(task, err)
		}
		return nil
	}), nil
}

func (tool *codeReviewTool) PostReviewRequests(
	ctxs []*common.ReviewContext,
	opts map[string]interface{},
) ([]*common.ReviewContext, error) {

	// Load the GitHub config.
	config, err := LoadConfig()
	if err != nil {
		return nil, err
	}

	// Get the GitHub owner and repository from the upstream URL.
	owner, repo, err := git.ParseUpstreamURL()
	if err != nil {
		return nil, err
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

	// Post the review requests.
	updatedCtxs := make([]*common.ReviewContext, 0, len(ctxs))

	// Post the assigned commits.
	for _, ctxs := range ctxsByStoryId {
		var (
			story         = ctxs[0].Story
			reviewRequest = ctxs[0].ReviewRequest
			commits       = make([]*git.Commit, 0, len(ctxs))
		)
		for _, ctx := range ctxs {
			commits = append(commits, ctx.Commit)
		}
		metadata, ex := postAssignedReviewRequest(config, owner, repo, story, commits, reviewRequest, opts)
		if ex != nil {
			errs.Log(ex)
			err = errPostReviewRequest
			continue
		}
		for _, ctx := range ctxs {
			updatedCtxs = append(updatedCtxs, &common.ReviewContext{
				Commit:        ctx.Commit,
				ReviewRequest: metadata,
				Story:         ctx.Story,
			})
		}
	}

	// Post the unassigned commits.
	for _, commit := range unassignedCommits {
		metadata, ex := postUnassignedReviewRequest(config, owner, repo, commit, opts)
		if ex != nil {
			errs.Log(ex)
			err = errPostReviewRequest
			continue
		}
		updatedCtxs = append(updateCtxs, &common.ReviewContext{
			Commit:        commit,
			ReviewRequest: metadata,
		})
	}

	return updatedCtxs, err
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

// postAssignedReviewRequest can be used to post
// the commits associated with the given story for review.
func postAssignedReviewRequest(
	config Config,
	owner string,
	repo string,
	story common.Story,
	commits []*git.Commit,
	reviewRequest *metastore.Resource,
	opts map[string]interface{},
) (*metastore.Resource, error) {

	if reviewRequest != nil {
		return extendReviewRequest(config, owner, repo, reviewRequest, commits, opts)
	} else {
		return createAssignedReviewRequest(config, owner, repo, story, commits, opts)
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
) (*metastore.Resource, error) {

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
	milestone, err := getOrCreateMilestoneForCommit(
		config, owner, repo, commits[len(commits)-1].SHA)
	if err != nil {
		return nil, err
	}

	// Create a new review issue.
	issue, metadata, err := createIssue(
		task, config, owner, repo, issueTitle, issueBody.String(), milestone)
	if err != nil {
		return nil, err
	}

	// Open the issue if requested.
	if _, open := opts["open"]; open {
		if err := openIssue(issue); err != nil {
			return nil, err
		}
	}
	return metadata, nil
}

// postUnassignedReviewRequest can be used to post the given commit for review.
// This function is to be used to post commits that are not associated with any story.
func postUnassignedReviewRequest(
	config Config,
	owner string,
	repo string,
	commit *git.Commit,
	opts map[string]interface{},
) (*metastore.Resource, error) {

	// Extend the specified review issue in case -fixes is specified.
	flagFixes, ok := opts["fixes"]
	if ok {
		if fixes, ok := flagFixes.(uint); ok && fixes != 0 {
			return extendUnassignedReviewRequest(config, owner, repo, int(fixes), commit, opts)
		}
	}

	if reviewRequest == nil {
		// Create a new review issue.
		return createUnassignedReviewRequest(config, owner, repo, commit, opts)
	} else {
		// The review request already exists, right? We are done.
		return reviewRequest, nil
	}
}

// createUnassignedReviewRequest created a new review issue
// for the given commit that is not associated with any story.
func createUnassignedReviewRequest(
	config Config,
	owner string,
	repo string,
	reviewRequest *metastore.Resource,
	commit *git.Commit,
	opts map[string]interface{},
) (*metastore.Resource, error) {

	// Generate the issue title and body.
	var (
		task       = fmt.Sprintf("Create review issue for commit %v", commit.SHA)
		issueTitle = fmt.Sprintf("Review commit %v: %v", commit.SHA, commit.MessageTitle)
		issueBody  = fmt.Sprintf("Commits to be reviewed:\n- [ ] %v: %v",
			commit.SHA, commit.MessageTitle)
	)

	// Get the right review milestone to add the issue into.
	milestone, err := getOrCreateMilestoneForCommit(config, owner, repo, commit.SHA)
	if err != nil {
		return nil, err
	}

	// Create a new review issue.
	issue, metadata, err := createIssue(
		task, config, owner, repo, issueTitle, issueBody, milestone)
	if err != nil {
		return nil, errs.NewError(task, err)
	}

	// Open the issue if requested.
	if _, open := opts["open"]; open {
		if err := openIssue(issue); err != nil {
			return nil, err
		}
	}
	return meta, nil
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
) (*metastore.Resource, error) {

	// Fetch the issue.
	task := fmt.Sprintf("Fetch GitHub issue #%v", issueNum)
	log.Run(task)
	client := ghutil.NewClient(config.Token())
	issue, _, err := client.Issues.Get(owner, repo, issueNum)
	if err != nil {
		return nil, errs.NewError(task, err)
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
		return errs.NewError(task, err)
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
		return errs.NewError(task, err)
	}
	return nil
}

func openIssue(issue *github.Issue) error {
	task := fmt.Sprintf("Open issue #%v in the browser", *issue.Number)
	if err := webbrowser.Open(*issue.HTMLURL); err != nil {
		return errs.NewError(task, err)
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
) (*github.Issue, *metastore.Resource, error) {

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
		return nil, nil, errs.NewError(task, err)
	}

	log.Log(fmt.Sprintf("GitHub issue #%v created", *issue.Number))
	return issue, metadata(issue), nil
}

func createMilestone(
	config Config,
	owner string,
	repo string,
	v *version.Version,
) (*github.Milestone, action.Action, error) {

	// Create the review milestone.
	var (
		releaseString = v.BaseString()
		milestoneTask = fmt.Sprintf("Create code review milestone for release %v", releaseString)
		client        = ghutil.NewClient(config.Token())
	)
	log.Run(milestoneTask)
	milestone, _, err := client.Issues.CreateMilestone(owner, repo, &github.Milestone{
		Title: github.String(milestoneTitle(v)),
	})
	if err != nil {
		return nil, nil, errs.NewError(milestoneTask, err)
	}

	// Return a rollback function.
	return milestone, action.ActionFunc(func() error {
		log.Rollback(milestoneTask)
		task := fmt.Sprintf("Delete code review milestone for release %v", releaseString)
		_, err := client.Issues.DeleteMilestone(owner, repo, *milestone.Number)
		if err != nil {
			return errs.NewError(task, err)
		}
		return nil
	}), nil
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
		return nil, errs.NewError(task, err)
	}

	// Find the right one.
	task = fmt.Sprintf("Find review milestone for release %v", v)
	for _, milestone := range milestones {
		if *milestone.Title == title {
			return &milestone, nil
		}
	}

	// Milestone not found.
	return nil, errs.NewError(task, &ErrMilestoneNotFound{v})
}

func getOrCreateMilestoneForCommit(
	config Config,
	owner string,
	repo string,
	sha string,
) (*github.Milestone, error) {

	// Get the version associated with the given commit.
	v, err := version.GetByBranch(sha)
	if err != nil {
		return nil, err
	}

	// Try to get the milestone.
	milestone, err := milestoneForVersion(config, owner, repo, v)
	if err != nil {
		// In case the milestone was not found, we try to create it.
		if _, ok := errs.RootCause(err).(*ErrMilestoneNotFound); ok {
			milestone, _, err := createMilestone(config, owner, repo, v)
			return milestone, err
		}
		return nil, err
	}

	// Milestone found, return it.
	return milestone, nil
}

func milestoneTitle(v *version.Version) string {
	return fmt.Sprintf("%v-review", v.BaseString())
}

func metadata(issue *github.Issue) *metastore.Resource {
	return &metastore.Resource{
		ServiceId: "github_codereview",
		Metadata: map[string]interface{}{
			"issue_number": *issue.Number,
			"issue_url":    *issue.URL,
		},
	}
}
