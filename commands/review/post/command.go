package postCmd

import (
	// Stdlib
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"text/tabwriter"

	// Internal
	"github.com/salsaflow/salsaflow/app"
	"github.com/salsaflow/salsaflow/asciiart"
	"github.com/salsaflow/salsaflow/errs"
	"github.com/salsaflow/salsaflow/git"
	"github.com/salsaflow/salsaflow/log"
	"github.com/salsaflow/salsaflow/modules"
	"github.com/salsaflow/salsaflow/modules/common"
	"github.com/salsaflow/salsaflow/prompt"

	// Other
	"gopkg.in/tchap/gocli.v2"
)

var Command = &gocli.Command{
	UsageLine: `
  post [-update=RRID] [-fixes=RRID] [-open] [REVISION]

  post [-fixes=RRID] [-no_fetch] [-no_rebase]
       [-ask_once] [-open] -parent=BRANCH`,
	Short: "post code review requests",
	Long: `
  Post a code review request for each commit specified.

  In case REVISION is specified, the selected revision is posted for review.
  Make sure the Story-Id tag is in the commit message, salsaflow will not try
  to rewrite the commit message for you in case it is not there.

  In case the parent branch BRANCH is specified, all the commits between
  BRANCH and HEAD are selected to be posted for code review. Using git revision
  ranges, these are the commits matching BRANCH..HEAD, or BRANCH.. for short.
  The selected commits are rebased onto the parent branch before posting.
  To prevent rebasing, use -no_rebase. To be asked to pick up the missing
  story ID only once and use it for all commits, set -ask_once.

  When no parent branch nor the revision is specified, the last commit
  on the current branch is selected and posted alone into the code review tool.
  `,
	Action: run,
}

var (
	flagAskOnce  bool
	flagFixes    uint
	flagNoFetch  bool
	flagNoRebase bool
	flagOpen     bool
	flagParent   string
	flagUpdate   uint
)

func init() {
	Command.Flags.BoolVar(&flagAskOnce, "ask_once", flagAskOnce,
		"ask once and reuse the story ID for all commits")
	Command.Flags.UintVar(&flagFixes, "fixes", flagFixes,
		"mark the commits as fixing issues in the given review request")
	Command.Flags.BoolVar(&flagNoFetch, "no_fetch", flagNoFetch,
		"do not fetch the upstream repository")
	Command.Flags.BoolVar(&flagNoRebase, "no_rebase", flagNoRebase,
		"do not rebase onto the parent branch")
	Command.Flags.BoolVar(&flagOpen, "open", flagOpen,
		"open the review requests in the browser")
	Command.Flags.StringVar(&flagParent, "parent", flagParent,
		"branch to be used in computing the revision range")
	Command.Flags.UintVar(&flagUpdate, "update", flagUpdate,
		"update an existing review request with REVISION")
}

var ErrNoCommits = errors.New("no commits selected for code review")

func run(cmd *gocli.Command, args []string) {
	if len(args) > 1 {
		cmd.Usage()
		os.Exit(2)
	}

	app.InitOrDie()

	// Exit cleanly when the panic is actually ErrCanceled.
	defer func() {
		if r := recover(); r != nil {
			if r == prompt.ErrCanceled {
				log.Println("\nOperation canceled. You are welcome to come back any time!")
			} else {
				panic(r)
			}
		}
	}()

	var err error
	switch {
	case len(args) == 1:
		err = postRevision(args[0])
	case flagParent != "":
		if flagUpdate != 0 {
			log.Fatalln("\nError: cannot use -update together with -parent")
		}
		err = postBranch(flagParent)
	default:
		err = postRevision("HEAD")
	}
	if err != nil {
		errs.Fatal(err)
	}
}

func postRevision(revision string) error {
	// Get the commit to be posted
	task := "Get the commit to be posted for code review"
	commits, err := git.ShowCommits(revision)
	if err != nil {
		return errs.NewError(task, err, nil)
	}

	return postReviewRequests(commits, revision == "HEAD")
}

func postBranch(parentBranch string) error {
	// Get the current branch name.
	currentBranch, err := git.CurrentBranch()
	if err != nil {
		return err
	}

	if !flagNoFetch {
		// Fetch the remote repository.
		task := "Fetch the remote repository"
		log.Run(task)

		gitConfig, err := git.LoadConfig()
		if err != nil {
			return errs.NewError(task, err, nil)
		}

		var remoteName = gitConfig.RemoteName()
		if err := git.UpdateRemotes(remoteName); err != nil {
			return errs.NewError(task, err, nil)
		}

		// Make sure the parent branch is up to date.
		task = fmt.Sprintf("Make sure reference '%v' is up to date", parentBranch)
		log.Run(task)
		if err := git.EnsureBranchSynchronized(parentBranch, remoteName); err != nil {
			return errs.NewError(task, err, nil)
		}

		// Make sure the current branch is up to date.
		task = fmt.Sprintf("Make sure branch '%v' is up to date", currentBranch)
		log.Run(task)
		if err = git.EnsureBranchSynchronized(currentBranch, remoteName); err != nil {
			return errs.NewError(task, err, nil)
		}
	}

	// Rebase the current branch on top the parent branch.
	if !flagNoRebase {
		task := fmt.Sprintf("Rebase branch '%v' onto '%v'", currentBranch, parentBranch)
		log.Run(task)
		if err := git.Rebase(parentBranch); err != nil {
			ex := errs.Log(errs.NewError(task, err, nil))
			asciiart.PrintGrimReaper("GIT REBASE FAILED")
			fmt.Printf(`Git failed to rebase your branch onto '%v'.

The repository might have been left in the middle of the rebase process.
In case you do not know how to handle this, just execute

  $ git rebase --abort

to make your repository clean again.

In any case, you have to rebase your current branch onto '%v'
if you want to continue and post a review request. In the edge cases
you can as well use -no_rebase to skip this step, but try not to do it.
`, parentBranch)
			return ex
		}
	}

	// Get the commits to be posted
	task := "Get the commits to be posted for code review"
	commits, err := git.ShowCommitRange(parentBranch + "..")
	if err != nil {
		return errs.NewError(task, err, nil)
	}

	// Post the review requests.
	return postReviewRequests(commits, true)
}

func postReviewRequests(commits []*git.Commit, canAmend bool) error {
	// Make sure there are actually some commits to be posted.
	task := "Make sure there are actually some commits to be posted"
	if len(commits) == 0 {
		return errs.NewError(task, ErrNoCommits, nil)
	}

	// Tell the user what is going to happen.
	fmt.Print(`
You are about to post review requests for the following commits:

`)
	mustListCommits(os.Stdout, commits, "  ")

	// Ask the user for confirmation.
	task = "Prompt the user for confirmation"
	confirmed, err := prompt.Confirm("\nYou cool with that?")
	if err != nil {
		return errs.NewError(task, err, nil)
	}
	if !confirmed {
		fmt.Println("\nFair enough, have a nice day!")
		return nil
	}
	fmt.Println()

	// Check the commits.
	task = "Make sure the commits comply with the rules"
	commits, err = rewriteCommits(commits, canAmend)
	if err != nil {
		return errs.NewError(task, err, nil)
	}

	// Print Snoopy.
	asciiart.PrintSnoopy()

	// Post the review requests.
	task = "Post the review requests"
	if err := sendReviewRequests(commits); err != nil {
		return errs.NewError(task, err, nil)
	}

	// In case there is no error, tell the user what to do next.
	printFollowup()

	return nil
}

func rewriteCommits(commits []*git.Commit, canAmend bool) ([]*git.Commit, error) {
	// Make sure we are not posting any merge commits.
	// Also check whether we need to fetch the stories or not.
	var (
		noMergeCommits []*git.Commit
		storyIdMissing bool
	)
	for _, commit := range commits {
		if commit.StoryId == "" {
			storyIdMissing = true
		}
		if commit.Merge == "" {
			noMergeCommits = append(noMergeCommits, commit)
		}
	}

	// Again, make sure there are actually some commits to be posted.
	if len(commits) == 0 {
		return commits, ErrNoCommits
	}

	// In case there is no Story-Id tag missing, we are done.
	if !storyIdMissing {
		log.Log("Commit check passed")
		return commits, nil
	}

	// In case we cannot add Story-Id tag, we have to return an error.
	if !canAmend {
		hint := `
The commit specified does not contain the Story-Id tag.
We are, however, unable to amend the commit message when
a revision is specified explicitly, because the revision
can be anywhere in the git commit graph.

Only the current branch tip (HEAD) or the current branch
as a whole can be amended. That is what the other review post
modes can do for you.

TL;DR: Please amend the commit manually to add the Story-Id tag,
or use some other mode of review post. To see what modes are
available, execute

  $ salsaflow review post -h

and read the DESCRIPTION section.

`
		return commits, errs.NewError(
			"Make sure the commits can be amended",
			errors.New("Story-Id tag missing"),
			bytes.NewBufferString(hint))
	}

	// Fetch the stories in progress from the issue tracker.
	storiesTask := "Missing Story-Id detected, fetch stories from the issue tracker"
	log.Run(storiesTask)

	tracker, err := modules.GetIssueTracker()
	if err != nil {
		return commits, errs.NewError(storiesTask, err, nil)
	}

	task := "Fetch the user record from the issue tracker"
	me, err := tracker.CurrentUser()
	if err != nil {
		return commits, errs.NewError(task, err, nil)
	}

	stories, err := tracker.StoriesInDevelopment()
	if err != nil {
		return commits, errs.NewError(storiesTask, err, nil)
	}

	// Show only the stories owned by the current user.
	var myStories []common.Story
StoryLoop:
	for _, story := range stories {
		for _, assignee := range story.Assignees() {
			if assignee.Id() == me.Id() {
				myStories = append(myStories, story)
				continue StoryLoop
			}
		}
	}
	stories = myStories

	// Tell the user what is happening.
	log.Run("Prepare a temporary branch to rewrite commit messages")

	// Get the current branch name.
	currentBranch, err := git.CurrentBranch()
	if err != nil {
		return commits, err
	}

	// Get the parent of the first commit in the chain.
	task = "Get the parent commit of the commit chain to be posted"
	stdout, err := git.Log("--pretty=%P", "-n", "1", commits[0].SHA)
	if err != nil {
		return commits, errs.NewError(task, err, nil)
	}
	parentSHA := strings.TrimSpace(stdout.String())

	// Prepare a temporary branch that will be used to amend commit messages.
	task = "Create a temporary branch to rewrite commit messages"
	tempBranch := "salsaflow/temp-review-post"
	if err := git.Branch("-f", tempBranch, parentSHA); err != nil {
		return commits, errs.NewError(task, err, nil)
	}
	defer func() {
		// Delete the temporary branch on exit.
		task := "Delete the temporary branch"
		if err := git.Branch("-D", tempBranch); err != nil {
			errs.LogError(task, err, nil)
		}
	}()

	// Checkout the temporary branch.
	task = "Checkout the temporary branch"
	if err := git.Checkout(tempBranch); err != nil {
		return commits, errs.NewError(task, err, nil)
	}
	defer func() {
		// Checkout the original branch on exit.
		task := fmt.Sprintf("Checkout branch '%v'", currentBranch)
		if err := git.Checkout(currentBranch); err != nil {
			errs.LogError(task, err, nil)
		}
	}()

	// Loop and rewrite the commit messages.
	var story common.Story
	if flagAskOnce {
		header := `
Some of the commits listed above are not assigned to any story.
Please pick up the story that these commits will be assigned to:`
		selectedStory, err := prompt.PromptStory(header, stories)
		if err != nil {
			if err == prompt.ErrCanceled {
				panic(err)
			}
			return commits, err
		}
		story = selectedStory
	}

	for _, commit := range commits {
		// Cherry-pick the commit.
		task := fmt.Sprintf("Move commit %v onto the temporary branch", commit.SHA)
		if err := git.CherryPick(commit.SHA); err != nil {
			return commits, errs.NewError(task, err, nil)
		}

		if commit.StoryId == "" {
			if !flagAskOnce {
				// Ask for the story ID for the current commit.
				header := fmt.Sprintf(`
The following commit is not assigned to any story:

  commit hash:  %v
  commit title: %v

Please pick up the story to assign the commit to:`, commit.SHA, commit.MessageTitle)
				selectedStory, err := prompt.PromptStory(header, stories)
				if err != nil {
					if err == prompt.ErrCanceled {
						panic(err)
					}
					return commits, err
				}
				story = selectedStory
			}

			// Extend the commit message to include Story-Id.
			commitMessage := fmt.Sprintf("%v\nStory-Id: %v\n", commit.Message, story.Tag())

			// Amend the cherry-picked commit to include the new commit message.
			task = "Amend the commit message for " + commit.SHA
			stderr := new(bytes.Buffer)
			cmd := exec.Command("git", "commit", "--amend", "-F", "-")
			cmd.Stdin = bytes.NewBufferString(commitMessage)
			cmd.Stderr = stderr
			if err := cmd.Run(); err != nil {
				return commits, errs.NewError(task, err, stderr)
			}
		}
	}

	// Reset the current branch to point to the new branch.
	task = "Reset the current branch to point to the temporary branch"
	if err := git.ResetKeep(currentBranch, tempBranch); err != nil {
		return commits, errs.NewError(task, err, nil)
	}

	// Parse the commits again since the commit hashes have changed.
	newCommits, err := git.ShowCommitRange(parentSHA + "..")
	if err != nil {
		return commits, err
	}

	log.NewLine("")
	log.Log("Commit messages amended successfully")

	// And we are done!
	return newCommits, nil
}

func mustListCommits(writer io.Writer, commits []*git.Commit, prefix string) {
	must := func(n int, err error) error {
		if err != nil {
			panic(err)
		}
		return err
	}

	tw := tabwriter.NewWriter(writer, 0, 8, 4, '\t', 0)

	must(fmt.Fprintf(tw, "%vCommit SHA\tCommit Title\n", prefix))
	must(fmt.Fprintf(tw, "%v==========\t============\n", prefix))
	for _, commit := range commits {
		must(fmt.Fprintf(tw, "%v%v\t%v\n", prefix, commit.SHA, commit.MessageTitle))
	}

	must(0, tw.Flush())
}

func sendReviewRequests(commits []*git.Commit) error {
	var (
		topErr    error
		topErrMux sync.Mutex
	)

	tool, err := modules.GetCodeReviewTool()
	if err != nil {
		return err
	}

	var postOpts = make(map[string]interface{}, 2)
	if flagFixes != 0 {
		postOpts["fixes"] = flagFixes
	}
	if flagUpdate != 0 {
		postOpts["update"] = flagUpdate
	}
	if flagOpen {
		postOpts["open"] = true
	}

	var wg sync.WaitGroup
	wg.Add(len(commits))
	for _, commit := range commits {
		go func(commit *git.Commit) {
			defer wg.Done()
			task := "Post the review request for commit " + commit.SHA
			log.Go(task)
			if err := tool.PostReviewRequest(commit, postOpts); err != nil {
				errs.LogError(task, err, nil)
				topErrMux.Lock()
				topErr = errors.New("failed to post a code review request")
				topErrMux.Unlock()
			}
		}(commit)
	}
	wg.Wait()

	return topErr
}

func printFollowup() {
	log.Println(`
----------

Now, please, take some time to go through all the review requests
to check and annotate them for the reviewers to make their part easier.

If you find any issues you want to fix (even before publishing),
do so now, and if you haven't pushed into any shared branch yet,
amend the relevant commit and use

  $ salsaflow review post -update REVIEW_REQUEST_ID [REVISION]

to update (replace) the associated review request. Do this for every review
request you want to overwrite.

In case you cannot amend the relevant commit any more, make sure the affected
review request is published, and use the process for fixing review issues:

  $ salsaflow review post -fixes REVIEW_REQUEST_ID [REVISION]

This will create a new review request that is linked to the one being fixed.

  ###########################################################
  # IMPORTANT: Your code has not been merged and/or pushed. #
  ###########################################################
`)
}
