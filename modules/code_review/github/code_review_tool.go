package github

import (
	// Stdlib
	"bytes"
	"errors"
	"fmt"

	// Internal
	"github.com/salsaflow/salsaflow/action"
	"github.com/salsaflow/salsaflow/errs"
	"github.com/salsaflow/salsaflow/git"
	ghutil "github.com/salsaflow/salsaflow/github"
	ghissues "github.com/salsaflow/salsaflow/github/issues"
	"github.com/salsaflow/salsaflow/log"
	"github.com/salsaflow/salsaflow/modules/common"
	"github.com/salsaflow/salsaflow/version"

	// Other
	"github.com/google/go-github/github"
	"github.com/toqueteos/webbrowser"
)

var errPostReviewRequest = errors.New("failed to post a review request")

func newCodeReviewTool() (common.CodeReviewTool, error) {
	config, err := loadConfig()
	if err != nil {
		return nil, err
	}

	return &codeReviewTool{config}, nil
}

type codeReviewTool struct {
	config *moduleConfig
}

func (tool *codeReviewTool) InitialiseRelease(v *version.Version) (action.Action, error) {
	// Get necessary config.
	owner, repo, err := ghutil.ParseUpstreamURL()
	if err != nil {
		return nil, err
	}

	// Get a GitHub client.
	client := ghutil.NewClient(tool.config.Token)

	// Check whether the review milestone exists or not.
	// People can create milestones manually, so this makes the thing more robust.
	title := milestoneTitle(v)
	task := fmt.Sprintf("Check whether GitHub review milestone exists for release %v", v.BaseString())
	log.Run(task)
	_, act, err := ghissues.GetOrCreateMilestoneForTitle(client, owner, repo, title)
	if err != nil {
		return nil, errs.NewError(task, err)
	}
	return act, nil
}

func (tool *codeReviewTool) FinaliseRelease(v *version.Version) (action.Action, error) {
	// Get a GitHub client.
	client := ghutil.NewClient(tool.config.Token)

	owner, repo, err := ghutil.ParseUpstreamURL()
	if err != nil {
		return nil, err
	}

	// Get the relevant review milestone.
	releaseString := v.BaseString()
	task := fmt.Sprintf("Get GitHub review milestone for release %v", releaseString)
	log.Run(task)
	milestone, err := milestoneForVersion(tool.config, owner, repo, v)
	if err != nil {
		return nil, errs.NewError(task, err)
	}
	if milestone == nil {
		log.Warn(fmt.Sprintf(
			"Weird, GitHub review milestone for release %v not found", releaseString))
		return nil, nil
	}

	// Close the milestone unless there are some issues open.
	task = fmt.Sprintf(
		"Make sure the review milestone for release %v can be closed", releaseString)
	if num := *milestone.OpenIssues; num != 0 {
		return nil, errs.NewError(
			task,
			fmt.Errorf(
				"review milestone for release %v cannot be closed: %v issue(s) open",
				releaseString, num))
	}

	milestoneTask := fmt.Sprintf("Close GitHub review milestone for release %v", releaseString)
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
		task := fmt.Sprintf("Reopen GitHub review milestone for release %v", releaseString)
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
) (err error) {

	// Get the GitHub owner and repository from the upstream URL.
	owner, repo, err := ghutil.ParseUpstreamURL()
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
		if ex := postAssignedReviewRequest(tool.config, owner, repo, story, commits, opts); ex != nil {
			errs.Log(ex)
			err = errPostReviewRequest
		}
	}

	// Post the unassigned commits.
	for _, commit := range unassignedCommits {
		if ex := postUnassignedReviewRequest(tool.config, owner, repo, commit, opts); ex != nil {
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

// postAssignedReviewRequest can be used to post
// the commits associated with the given story for review.
func postAssignedReviewRequest(
	config *moduleConfig,
	owner string,
	repo string,
	story common.Story,
	commits []*git.Commit,
	opts map[string]interface{},
) error {

	// Search for an existing review issue for the given story.
	task := fmt.Sprintf("Search for an existing review issue for story %v", story.ReadableId())
	log.Run(task)

	client := ghutil.NewClient(config.Token)
	issue, err := ghissues.FindReviewIssueForStory(client, owner, repo, story.ReadableId())
	if err != nil {
		return errs.NewError(task, err)
	}

	// Decide what to do next based on the search results.
	if issue == nil {
		// No review issue found for the given story, create a new issue.
		issue, err = createAssignedReviewRequest(config, owner, repo, story, commits, opts)
	} else {
		// An existing review issue found, extend it.
		err = extendReviewRequest(config, owner, repo, issue, commits, opts)
	}
	if err != nil {
		return err
	}

	// Link commits to the review issue.
	return linkCommitToReviewIssue(config, owner, repo, *issue.Number, commits)
}

// createAssignedReviewRequest can be used to create a new review issue
// for the given commits that is associated with the story passed in.
func createAssignedReviewRequest(
	config *moduleConfig,
	owner string,
	repo string,
	story common.Story,
	commits []*git.Commit,
	opts map[string]interface{},
) (*github.Issue, error) {

	task := fmt.Sprintf("Create review issue for story %v", story.ReadableId())

	// Prepare the issue object.
	issue := ghissues.NewStoryReviewIssue(
		story.ReadableId(),
		story.URL(),
		story.Title(),
		story.IssueTracker().ServiceName(),
		story.Tag())

	for _, commit := range commits {
		issue.AddCommit(false, commit.SHA, commit.MessageTitle)
	}

	// Get the right review milestone to add the issue into.
	milestone, err := getOrCreateMilestoneForCommit(
		config, owner, repo, commits[len(commits)-1].SHA)
	if err != nil {
		return nil, errs.NewError(task, err)
	}

	// Create a new review issue.
	var implemented bool
	implementedOpt, ok := opts["implemented"]
	if ok {
		implemented = implementedOpt.(bool)
	}

	issueResource, err := createIssue(
		task, config, owner, repo,
		issue.FormatTitle(), issue.FormatBody(),
		optValueString(opts["reviewer"]), milestone, implemented)
	if err != nil {
		return nil, errs.NewError(task, err)
	}

	// Open the issue if requested.
	if _, open := opts["open"]; open {
		if err := openIssue(issueResource); err != nil {
			return nil, err
		}
	}

	return issueResource, nil
}

// postUnassignedReviewRequest can be used to post the given commit for review.
// This function is to be used to post commits that are not associated with any story.
func postUnassignedReviewRequest(
	config *moduleConfig,
	owner string,
	repo string,
	commit *git.Commit,
	opts map[string]interface{},
) error {

	// Extend the specified review issue in case -fixes is specified.
	flagFixes, ok := opts["fixes"]
	if ok {
		if fixes, ok := flagFixes.(uint); ok && fixes != 0 {
			issueNum := int(fixes)

			// Extend the specified review issue.
			err := extendUnassignedReviewRequest(config, owner, repo, issueNum, commit, opts)
			if err != nil {
				return err
			}

			// Link the commit to the review issue.
			return linkCommitToReviewIssue(config, owner, repo, issueNum, []*git.Commit{commit})
		}
	}

	// Search for an existing issue.
	task := fmt.Sprintf("Search for an existing review issue for commit %v", commit.SHA)
	log.Run(task)

	client := ghutil.NewClient(config.Token)
	issue, err := ghissues.FindReviewIssueForCommit(client, owner, repo, commit.SHA)
	if err != nil {
		return errs.NewError(task, err)
	}

	// Decide what to do next based on the search results.
	if issue == nil {
		// Create a new unassigned review request.
		issue, err = createUnassignedReviewRequest(config, owner, repo, commit, opts)
		if err != nil {
			return err
		}

		// Link the commit to the review issue.
		return linkCommitToReviewIssue(config, owner, repo, *issue.Number, []*git.Commit{commit})
	} else {
		// The issues already exists, return an error.
		issueNum := *issue.Number
		err := fmt.Errorf("existing review issue found for commit %v: %v", commit.SHA, issueNum)
		return errs.NewError("Make sure the review issue can be created", err)
	}
}

// createUnassignedReviewRequest created a new review issue
// for the given commit that is not associated with any story.
func createUnassignedReviewRequest(
	config *moduleConfig,
	owner string,
	repo string,
	commit *git.Commit,
	opts map[string]interface{},
) (*github.Issue, error) {

	task := fmt.Sprintf("Create review issue for commit %v", commit.SHA)

	// Prepare the issue object.
	issue := ghissues.NewCommitReviewIssue(commit.SHA, commit.MessageTitle)

	// Get the right review milestone to add the issue into.
	milestone, err := getOrCreateMilestoneForCommit(config, owner, repo, commit.SHA)
	if err != nil {
		return nil, errs.NewError(task, err)
	}

	// Create a new review issue.
	issueResource, err := createIssue(
		task, config, owner, repo,
		issue.FormatTitle(), issue.FormatBody(),
		optValueString(opts["reviewer"]), milestone, true)
	if err != nil {
		return nil, errs.NewError(task, err)
	}

	// Open the issue if requested.
	if _, open := opts["open"]; open {
		if err := openIssue(issueResource); err != nil {
			return nil, err
		}
	}

	return issueResource, nil
}

// extendUnassignedReviewRequest can be used to upload fixes for
// the specified unassigned review issue.
func extendUnassignedReviewRequest(
	config *moduleConfig,
	owner string,
	repo string,
	issueNum int,
	commit *git.Commit,
	opts map[string]interface{},
) error {

	// Fetch the issue.
	task := fmt.Sprintf("Fetch GitHub issue #%v", issueNum)
	log.Run(task)
	client := ghutil.NewClient(config.Token)
	issue, _, err := client.Issues.Get(owner, repo, issueNum)
	if err != nil {
		return errs.NewError(task, err)
	}

	// Extend the given review issue.
	return extendReviewRequest(config, owner, repo, issue, []*git.Commit{commit}, opts)
}

// extendReviewRequest is a general function that can be used to extend
// the given review issue with the given list of commits.
func extendReviewRequest(
	config *moduleConfig,
	owner string,
	repo string,
	issue *github.Issue,
	commits []*git.Commit,
	opts map[string]interface{},
) error {

	issueNum := *issue.Number

	// Parse the issue.
	task := fmt.Sprintf("Parse review issue #%v", issueNum)
	reviewIssue, err := ghissues.ParseReviewIssue(issue)
	if err != nil {
		return errs.NewError(task, err)
	}

	// Add the commits.
	newCommits := make([]*git.Commit, 0, len(commits))
	for _, commit := range commits {
		if reviewIssue.AddCommit(false, commit.SHA, commit.MessageTitle) {
			newCommits = append(newCommits, commit)
		}
	}
	if len(newCommits) == 0 {
		log.Log(fmt.Sprintf("All commits already listed in issue #%v", issueNum))
		return nil
	}

	// Add the implemented label if necessary.
	var (
		implemented      bool
		implementedLabel = config.StoryImplementedLabel
		labelsPtr        *[]string
	)
	implementedOpt, ok := opts["implemented"]
	if ok {
		implemented = implementedOpt.(bool)
	}
	if implemented {
		labels := make([]string, 0, len(issue.Labels)+1)
		labelsPtr = &labels

		for _, label := range issue.Labels {
			if *label.Name == implementedLabel {
				// The label is already there, for some reason.
				// Set the pointer to nil so that we don't update labels.
				labelsPtr = nil
				break
			}
			labels = append(labels, *label.Name)
		}
		if labelsPtr != nil {
			labels = append(labels, implementedLabel)
		}
	}

	// Edit the issue.
	task = fmt.Sprintf("Update GitHub issue #%v", issueNum)
	log.Run(task)

	client := ghutil.NewClient(config.Token)
	newIssue, _, err := client.Issues.Edit(owner, repo, issueNum, &github.IssueRequest{
		Body:   github.String(reviewIssue.FormatBody()),
		State:  github.String("open"),
		Labels: labelsPtr,
	})
	if err != nil {
		return errs.NewError(task, err)
	}

	// Add the review comment.
	if err := addReviewComment(config, owner, repo, issueNum, newCommits); err != nil {
		return err
	}

	// Open the issue if requested.
	if _, open := opts["open"]; open {
		return openIssue(newIssue)
	}
	return nil
}

func addReviewComment(
	config *moduleConfig,
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
	client := ghutil.NewClient(config.Token)
	_, _, err := client.Issues.CreateComment(owner, repo, issueNum, &github.IssueComment{
		Body: github.String(buffer.String()),
	})
	if err != nil {
		return errs.NewError(task, err)
	}
	return nil
}

func linkCommitToReviewIssue(
	config *moduleConfig,
	owner string,
	repo string,
	issueNum int,
	commits []*git.Commit,
) error {

	// Instantiate an API client.
	client := ghutil.NewClient(config.Token)

	// Loop over the commits and post a commit comment for each of them.
	for _, commit := range commits {
		task := fmt.Sprintf("Link commit %v to the associated review issue", commit.SHA)
		log.Run(task)

		body := fmt.Sprintf(
			"This commit is being reviewed as a part of review issue #%v.", issueNum)
		comment := &github.RepositoryComment{
			Body: &body,
		}
		_, _, err := client.Repositories.CreateComment(owner, repo, commit.SHA, comment)
		if err != nil {
			// Just print the error to the console.
			errs.LogError(task, err)
		}
	}

	// We actually never return an error, but let's keep this open.
	// It's much harder to switch from returning nothing to returning an error.
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
	config *moduleConfig,
	owner string,
	repo string,
	issueTitle string,
	issueBody string,
	assignee string,
	milestone *github.Milestone,
	implemented bool,
) (issue *github.Issue, err error) {

	log.Run(task)
	client := ghutil.NewClient(config.Token)

	var labels []string
	if implemented {
		labels = []string{config.ReviewLabel, config.StoryImplementedLabel}
	} else {
		labels = []string{config.ReviewLabel}
	}

	var assigneePtr *string
	if assignee != "" {
		assigneePtr = &assignee
	}

	issue, _, err = client.Issues.Create(owner, repo, &github.IssueRequest{
		Title:     github.String(issueTitle),
		Body:      github.String(issueBody),
		Labels:    &labels,
		Assignee:  assigneePtr,
		Milestone: milestone.Number,
	})
	if err != nil {
		return nil, errs.NewError(task, err)
	}

	log.Log(fmt.Sprintf("GitHub issue #%v created", *issue.Number))
	return issue, nil
}

func createMilestone(
	config *moduleConfig,
	owner string,
	repo string,
	v *version.Version,
) (*github.Milestone, action.Action, error) {

	// Create the review milestone for the given version.
	var (
		client = ghutil.NewClient(config.Token)
		title  = milestoneTitle(v)
	)
	return ghissues.CreateMilestone(client, owner, repo, title)
}

func milestoneForVersion(
	config *moduleConfig,
	owner string,
	repo string,
	v *version.Version,
) (*github.Milestone, error) {

	// Find the milestone matching the version.
	var (
		client = ghutil.NewClient(config.Token)
		title  = milestoneTitle(v)
	)
	return ghissues.FindMilestoneByTitle(client, owner, repo, title)
}

func getOrCreateMilestoneForCommit(
	config *moduleConfig,
	owner string,
	repo string,
	sha string,
) (*github.Milestone, error) {

	// Get the version associated with the given commit.
	v, err := version.GetByBranch(sha)
	if err != nil {
		return nil, err
	}

	// Get or create the milestone for the given title.
	var (
		client = ghutil.NewClient(config.Token)
		title  = milestoneTitle(v)
	)
	milestone, _, err := ghissues.GetOrCreateMilestoneForTitle(client, owner, repo, title)
	return milestone, err
}

func milestoneTitle(v *version.Version) string {
	return fmt.Sprintf("%v-review", v.BaseString())
}

func optValueString(value interface{}) string {
	if value == nil {
		return ""
	}
	return value.(string)
}