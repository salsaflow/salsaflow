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
	"github.com/salsaflow/salsaflow/errs"
	"github.com/salsaflow/salsaflow/git"
	ghutil "github.com/salsaflow/salsaflow/github"
	"github.com/salsaflow/salsaflow/log"
	"github.com/salsaflow/salsaflow/modules/common"

	// Other
	"github.com/google/go-github/github"
	"github.com/toqueteos/webbrowser"
)

const Id = "github"

type codeReviewTool struct{}

func Factory() (common.CodeReviewTool, error) {
	return &codeReviewTool{}, nil
}

func (tool *codeReviewTool) PostReviewRequestForCommit(
	ctx *common.CommitReviewContext,
	opts map[string]interface{},
) error {

	return tool.PostReviewRequestForBranch("", []*common.CommitReviewContext{ctx}, opts)
}

func (tool *codeReviewTool) PostReviewRequestForBranch(
	branch string,
	ctxs []*common.CommitReviewContext,
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

	// Initialise a GitHub API client.
	client := ghutil.NewClient(config.Token())

	// Group commits by story ID.
	ctxsByStoryId := make(map[string][]*common.CommitReviewContext, 1)
	ctxsUnassigned := make([]*common.CommitReviewContext, 0, 1)
	for _, ctx := range ctxs {
		story := ctx.Story
		if story != nil && story.Tag() != git.StoryIdUnassignedTagValue {
			sid := story.ReadableId()
			cts, ok := ctxsByStoryId[sid]
			if ok {
				cts = append(cts, ctx)
			} else {
				cts = []*common.CommitReviewContext{ctx}
			}
			ctxsByStoryId[sid] = cts
		} else {
			ctxsUnassigned = append(ctxsUnassigned, ctx)
		}
	}

	// Collect the issue numbers updated to post comments at the end.
	var (
		issueNumsAffected = make(map[int]struct{}, 1)
		issuesEdited      = make(map[int][]*common.CommitReviewContext, 1)
	)

	// Go through the commits and post the review requests.
	// Try to post all commits and print errors as they come.
	for _, ctxs := range ctxsByStoryId {
		// Post a new review request for the given commit.
		issue, edited, ex := postReviewRequest(config, owner, repo, ctxs)
		if ex != nil {
			errs.Log(ex)
			err = errors.New("failed to post a review request")
			continue
		}

		// Register the issue as created or edited.
		issueNum := *issue.Number
		issueNumsAffected[issueNum] = struct{}{}
		if edited {
			issuesEdited[issueNum] = ctxs
		}
	}

	fixes, _ := opts["fixes"].(uint)
	for _, ctx := range ctxsUnassigned {
		issue, ex := postUnassignedReviewRequest(config, owner, repo, ctx.Commit, int(fixes))
		if ex != nil {
			errs.Log(ex)
			err = errors.New("failed to post a review request")
			continue
		}

		issueNumsAffected[*issue.Number] = struct{}{}
	}

	// Post review comments for the issue that were edited.
	for num, ctxs := range issuesEdited {
		// Generate the comment body.
		var buffer bytes.Buffer
		fmt.Fprintf(&buffer, "Added commit %v: %v", ctxs[0].Commit.SHA, ctxs[0].Commit.MessageTitle)
		for _, ctx := range ctxs[1:] {
			fmt.Fprintf(&buffer, "\nAdded commit %v: %v", ctx.Commit.SHA, ctx.Commit.MessageTitle)
		}

		// Call GitHub API.
		task := fmt.Sprintf("Add review comment for issue #%v", num)
		log.Run(task)
		_, _, ex := client.Issues.CreateComment(owner, repo, num, &github.IssueComment{
			Body: github.String(buffer.String()),
		})
		if ex != nil {
			errs.LogError(task, ex, nil)
			if err == nil {
				err = errors.New("failed to create a GitHub issue comment")
			}
		}
	}

	// Open the issues in the browser if requested.
	if _, open := opts["open"]; open {
		for num := range issueNumsAffected {
			u := fmt.Sprintf("https://github.com/%v/%v/issues/%v", owner, repo, num)
			if ex := webbrowser.Open(u); ex != nil {
				errs.LogError(fmt.Sprintf("Open issue #%v in the browser", num), ex, nil)
				if err == nil {
					err = errors.New("failed to open a GitHub issue in the browser")
				}
			}
		}
	}

	return
}

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

func postUnassignedReviewRequest(
	config Config,
	owner string,
	repo string,
	commit *git.Commit,
	fixes int,
) (*github.Issue, error) {

	// Assert that certain field are set.
	switch {
	case commit.SHA == "":
		panic("SHA not set for the commit being posted")
	}

	// Handle differently in case -fixes is specified.
	if fixes != 0 {
		return postFixForUnassignedReviewRequest(config, owner, repo, fixes, commit)
	}

	// Search for an existing issue.
	task := fmt.Sprintf("Search for an existing review issue for commit %v", commit.SHA)
	log.Run(task)

	query := fmt.Sprintf(
		"\"Review commit %v\" repo:%v/%v label:%v type:issue in:title",
		commit.SHA, owner, repo, config.ReviewLabel())

	client := ghutil.NewClient(config.Token())
	res, _, err := client.Search.Issues(query, &github.SearchOptions{})
	if err != nil {
		return nil, errs.NewError(task, err, nil)
	}

	// Decide what to do next based on the search results.
	switch len(res.Issues) {
	case 0:
		// Just create a new issue.
		var (
			task       = fmt.Sprintf("Create review issue for commit %v", commit.SHA)
			issueTitle = fmt.Sprintf("Review commit %v: %v", commit.SHA, commit.MessageTitle)
			issueBody  = fmt.Sprintf("Commits to be reviewed:\n- [ ] %v: %v",
				commit.SHA, commit.MessageTitle)
		)
		return createIssue(task, config, owner, repo, issueTitle, issueBody)

	case 1:
		// Nothing to be done.
		task := "Make sure the review issue can be created"
		issueNum := *res.Issues[0].Number
		err := fmt.Errorf("existing review issue found for commit %v: #%v", commit.SHA, issueNum)
		return nil, errs.NewError(task, err, nil)

	default:
		// Inconsistency detected: multiple review issues found.
		task := "Make sure the review issue can be created"
		err = fmt.Errorf("multiple commit review issues found for commit %v", commit.SHA)
		return nil, errs.NewError(task, err, nil)
	}
}

func postFixForUnassignedReviewRequest(
	config Config,
	owner string,
	repo string,
	issueNum int,
	commit *git.Commit,
) (*github.Issue, error) {

	// Fetch the issue.
	task := fmt.Sprintf("Fetch GitHub issue #%v", issueNum)
	log.Run(task)
	client := ghutil.NewClient(config.Token())
	issue, _, err := client.Issues.Get(owner, repo, issueNum)
	if err != nil {
		return nil, errs.NewError(task, err, nil)
	}

	// Extend the body.
	task = fmt.Sprintf("Edit GitHub issue #%v to include the new commit", issueNum)
	log.Run(task)
	issueBody := fmt.Sprintf("%v\n- [ ] %v: %v", *issue.Body, commit.SHA, commit.MessageTitle)
	issue, _, err = client.Issues.Edit(owner, repo, issueNum, &github.IssueRequest{
		Body:  github.String(issueBody),
		State: github.String("open"),
	})
	if err != nil {
		return nil, errs.NewError(task, err, nil)
	}

	// Comment on the issue.
	task = fmt.Sprintf("Add review comment for GitHub issue #%v", issueNum)
	log.Run(task)
	_, _, err = client.Issues.CreateComment(owner, repo, issueNum, &github.IssueComment{
		Body: github.String(fmt.Sprintf("Added commit %v: %v", commit.SHA, commit.MessageTitle)),
	})
	if err != nil {
		return nil, errs.NewError(task, err, nil)
	}

	// Success!
	return issue, nil
}

func postReviewRequest(
	config Config,
	owner string,
	repo string,
	ctxs []*common.CommitReviewContext,
) (issue *github.Issue, edited bool, err error) {

	client := ghutil.NewClient(config.Token())

	// Try to find an existing issue to update.
	story := ctxs[0].Story
	searchTask := fmt.Sprintf("Search for an existing review issue for story %v", story.ReadableId())
	log.Run(searchTask)

	query := fmt.Sprintf(
		"\"Review story %v\" repo:%v/%v label:%v type:issue in:title",
		story.ReadableId(), owner, repo, config.ReviewLabel())

	res, _, err := client.Search.Issues(query, &github.SearchOptions{})
	if err != nil {
		return nil, false, errs.NewError(searchTask, err, nil)
	}

	// Decide what to do next based on the search results.
	switch len(res.Issues) {
	case 0:
		// No issue found, create a new story review issue.
		var (
			task       = fmt.Sprintf("Create review issue for story %v", story.ReadableId())
			issueTitle = fmt.Sprintf("Review story %v: %v", story.ReadableId(), story.Title())
		)

		// Generate the issue body.
		var issueBody bytes.Buffer
		fmt.Fprintf(&issueBody, "Story being reviewed: [%v](%v)\n\n", story.ReadableId(), story.URL())
		fmt.Fprintf(&issueBody, "SF-Issue-Tracker: %v\n", story.IssueTrackerName())
		fmt.Fprintf(&issueBody, "SF-Story-Key: %v\n\n", story.Tag())
		fmt.Fprintf(&issueBody, "The associated commits are following:")
		for _, ctx := range ctxs {
			commit := ctx.Commit
			if commit.SHA == "" {
				panic("SHA not set for the commit being posted")
			}
			fmt.Fprintf(&issueBody, "\n- [ ] %v: %v", commit.SHA, commit.MessageTitle)
		}

		isu, err := createIssue(task, config, owner, repo, issueTitle, issueBody.String())
		if err != nil {
			return nil, false, err
		}
		return isu, false, nil

	case 1:
		// A story review issue found, amend it to include this commit.
		var (
			isu         = res.Issues[0]
			issueNum    = *isu.Number
			issueBody   = *isu.Body
			bodyBuffer  = bytes.NewBufferString(issueBody)
			bodyChanged bool
		)

		for _, ctx := range ctxs {
			commit := ctx.Commit

			// Make sure the commit is not added yet.
			line := fmt.Sprintf("] %v: %v", commit.SHA, commit.MessageTitle)
			if strings.Contains(issueBody, line) {
				log.Log(fmt.Sprintf("Commit %v already listed in issue #%v", commit.SHA, issueNum))
				continue
			}

			// Extend the issue body.
			bodyChanged = true
			fmt.Fprintf(bodyBuffer, "\n- [ ] %v: %v", commit.SHA, commit.MessageTitle)
		}

		// Edit the issue.
		if !bodyChanged {
			log.Log("No need to modify any GitHub issue")
			return &isu, false, nil
		}

		task := fmt.Sprintf("Update GitHub review issue #%v", issueNum)
		log.Run(task)

		newIssue, _, err := client.Issues.Edit(owner, repo, issueNum, &github.IssueRequest{
			Body:  github.String(bodyBuffer.String()),
			State: github.String("open"),
		})
		if err != nil {
			return nil, false, errs.NewError(task, err, nil)
		}
		return newIssue, true, nil

	default:
		err := errors.New("inconsistency detected: multiple story review issues found")
		return nil, false, errs.NewError(searchTask, err, nil)
	}
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

func createIssue(
	task string,
	config Config,
	owner string,
	repo string,
	issueTitle string,
	issueBody string,
) (issue *github.Issue, err error) {

	log.Run(task)
	client := ghutil.NewClient(config.Token())
	issue, _, err = client.Issues.Create(owner, repo, &github.IssueRequest{
		Title:  github.String(issueTitle),
		Body:   github.String(issueBody),
		Labels: []string{config.ReviewLabel()},
	})
	if err != nil {
		return nil, errs.NewError(task, err, nil)
	}
	return issue, nil
}
