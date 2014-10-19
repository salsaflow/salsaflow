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
	"github.com/salsita/salsaflow/app"
	"github.com/salsita/salsaflow/asciiart"
	"github.com/salsita/salsaflow/config"
	"github.com/salsita/salsaflow/errs"
	"github.com/salsita/salsaflow/git"
	"github.com/salsita/salsaflow/log"
	"github.com/salsita/salsaflow/modules"
	"github.com/salsita/salsaflow/modules/common"
	"github.com/salsita/salsaflow/prompt"

	// Other
	"gopkg.in/tchap/gocli.v1"
)

var Command = &gocli.Command{
	UsageLine: "post [-parent=BRANCH] [-no_fetch] [-no_rebase] [REVISION]",
	Short:     "post code review requests",
	Long: `
  Post a code review request for each commit specified.

  In case REVISION is specified, the selected revision is posted for review.
  Make sure the Story-Id tag is in the commit message, salsaflow will not try
  to rewrite the commit message for you in case it is not there.

  In case the parent branch BRANCH is specified, all the commits between
  BRANCH and HEAD are selected to be posted for code review. Using git revision
  ranges, these are the commits matching BRANCH..HEAD, or BRANCH.. for short.
  The selected sommits are rebased onto the parent branch before posting.
  To prevent rebasing, use -no_rebase.

  When no parent branch nor the revision is specified, the last commit
  on the current branch is selected and posted alone into the code review tool.
  `,
	Action: run,
}

var (
	flagParent   string
	flagNoFetch  bool
	flagNoRebase bool
)

func init() {
	Command.Flags.StringVar(&flagParent, "parent", flagParent,
		"branch to be used in computing the revision range")
	Command.Flags.BoolVar(&flagNoFetch, "no_fetch", flagNoFetch,
		"do not fetch the upstream repository")
	Command.Flags.BoolVar(&flagNoRebase, "no_rebase", flagNoRebase,
		"do not rebase onto the parent branch")
}

func run(cmd *gocli.Command, args []string) {
	if len(args) > 1 {
		cmd.Usage()
		os.Exit(2)
	}

	app.MustInit()

	var err error
	switch {
	case len(args) == 1:
		err = postRevision(args[0])
	case flagParent != "":
		err = postBranch(flagParent)
	default:
		err = postRevision("HEAD")
	}
	if err != nil {
		log.Fatalln("\nError: " + err.Error())
	}
}

func handleError(task string, err error, stderr *bytes.Buffer) error {
	errs.NewError(task, stderr, err).Log(log.V(log.Info))
	return err
}

func postRevision(revision string) error {
	// Get the commit to be posted
	msg := "Get the commit to be posted for code review"
	commits, stderr, err := git.ShowCommits(revision)
	if err != nil {
		return handleError(msg, err, stderr)
	}

	return postReviewRequests(commits, revision == "HEAD")
}

func postBranch(parentBranch string) error {
	if !flagNoFetch {
		// Fetch the remote repository.
		msg := "Fetch the remote repository"
		log.Run(msg)
		if stderr, err := git.UpdateRemotes(config.OriginName); err != nil {
			return handleError(msg, err, stderr)
		}

		// Make sure the parent branch is up to date.
		msg = fmt.Sprintf("Make sure branch '%v' is up to date", parentBranch)
		log.Run(msg)
		exists, stderr, err := git.RemoteBranchExists(parentBranch, config.OriginName)
		if err != nil {
			return handleError(msg, err, stderr)
		}
		if exists {
			stderr, err = git.EnsureBranchSynchronized(config.TrunkBranch, config.OriginName)
			if err != nil {
				return handleError(msg, err, stderr)
			}
		}
	}

	// Rebase the current branch on top the parent branch.
	if !flagNoRebase {
		msg := fmt.Sprintf("Rebase the current branch onto '%v'", parentBranch)
		log.Run(msg)
		_, stderr, err := git.Git("rebase", parentBranch)
		if err != nil {
			handleError(msg, err, stderr)
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
			return err
		}
	}

	// Get the commits to be posted
	msg := "Get the commits to be posted for code review"
	commits, stderr, err := git.ShowCommitRange(parentBranch + "..")
	if err != nil {
		return handleError(msg, err, stderr)
	}

	// Post the review requests.
	return postReviewRequests(commits, true)
}

func postReviewRequests(commits []*git.Commit, canAmend bool) error {
	// Tell the user what is going to happen.
	fmt.Print(`
You are about to post review requests for the following commits:

`)
	mustBumpCommits(os.Stdout, commits, "  ")

	// Ask the user for confirmation.
	msg := "Prompt the user for confirmation"
	confirmed, err := prompt.Confirm("\nYou cool with that?")
	if err != nil {
		return handleError(msg, err, nil)
	}
	if !confirmed {
		fmt.Println("\nFair enough, have a nice day!")
		return nil
	}
	fmt.Println()

	// Check the commits.
	msg = "Make sure the commits comply with the rules"
	commits, err = rewriteCommits(commits, canAmend)
	if err != nil {
		return handleError(msg, err, nil)
	}

	// Print Snoopy.
	asciiart.PrintSnoopy()

	// Post the review requests.
	msg = "Post the review request"
	if err := sendReviewRequests(commits); err != nil {
		return handleError(msg, err, nil)
	}

	// Tell the user what to do next.
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

	// Make sure there are actually some commits to be posted.
	if len(commits) == 0 {
		return commits, errors.New("no commits selected for code review")
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
			bytes.NewBufferString(hint),
			errors.New("Story-Id tag missing"))
	}

	log.Log("Story-Id tags missing, rewriting the commits")

	// Fetch the stories in progress from the issue tracker.
	storiesMsg := "Fetch stories from the issue tracker"
	log.Run(storiesMsg)

	tracker := modules.GetIssueTracker()

	msg := "Fetch the user record from the issue tracker"
	me, err := tracker.CurrentUser()
	if err != nil {
		return commits, errs.NewError(msg, nil, err)
	}

	stories, err := tracker.StoriesInProgress()
	if err != nil {
		return commits, errs.NewError(storiesMsg, nil, err)
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
	msg = "Get the current branch name"
	currentBranch, stderr, err := git.CurrentBranch()
	if err != nil {
		return commits, errs.NewError(msg, stderr, err)
	}

	// Get the parent of the first commit in the chain.
	msg = "Get the parent commit of the commit chain to be posted"
	stdout, stderr, err := git.Log("--pretty=%P", "-n", "1", commits[0].SHA)
	if err != nil {
		return commits, errs.NewError(msg, stderr, err)
	}
	parentSHA := strings.TrimSpace(stdout.String())

	// Prepare a temporary branch that will be used to amend commit messages.
	msg = "Create a temporary branch to rewrite commit messages"
	tempBranch := "salsaflow/temp-review-post"
	if stderr, err = git.Branch("-f", tempBranch, parentSHA); err != nil {
		return commits, errs.NewError(msg, stderr, err)
	}
	defer func() {
		// Delete the temporary branch on exit.
		msg := "Delete the temporary branch"
		if stderr, err := git.Branch("-D", tempBranch); err != nil {
			errs.NewError(msg, stderr, err).Log(log.V(log.Info))
		}
	}()

	// Checkout the temporary branch.
	msg = "Checkout the temporary branch"
	if stderr, err := git.Checkout(tempBranch); err != nil {
		return commits, errs.NewError(msg, stderr, err)
	}
	defer func() {
		// Checkout the original branch on exit.
		msg := "Checkout the original branch"
		if stderr, err := git.Checkout(currentBranch); err != nil {
			errs.NewError(msg, stderr, err).Log(log.V(log.Info))
		}
	}()

	// Loop and rewrite the commit messages.
	for _, commit := range commits {
		// Cherry-pick the commit.
		msg := "Move the next commit onto the temporary branch"
		if _, stderr, err := git.CherryPick(commit.SHA); err != nil {
			return commits, errs.NewError(msg, stderr, err)
		}

		if commit.StoryId == "" {
			// Tell the user what is happening.
			fmt.Printf(`
The following commit is not assigned to any story:

  commit hash:  %v
  commit title: %v

The commit can be assigned to one of the following stories.
Please pick up the story to assign the commit to.

`, commit.SHA, commit.MessageTitle)

			tw := tabwriter.NewWriter(os.Stdout, 0, 8, 4, '\t', 0)
			io.WriteString(tw, "  Index\tStory ID\tStory Title\n")
			io.WriteString(tw, "  =====\t========\t===========\n")
			for i, story := range stories {
				fmt.Fprintf(tw, "  %v\t%v\t%v\n", i, story.ReadableId(), story.Title())
			}
			io.WriteString(tw, "\n")
			tw.Flush()

			// Prompt the user to select a story to assign the commit with.
			msg = "Prompt the user to select a story"
			index, err := prompt.PromptIndex("Choose a story by inserting its index: ", 0, len(stories)-1)
			if err != nil {
				return commits, errs.NewError(msg, nil, err)
			}
			story := stories[index]

			// Amend the commit message to include Story-Id.
			commit.Message += fmt.Sprintf("\nStory-Id: %v\n", story.ReadableId())
			commit.StoryId = story.ReadableId()

			// Amend the cherry-picked commit to include the new commit message.
			msg = "Amend the commit message for " + commit.SHA
			stderr = new(bytes.Buffer)
			cmd := exec.Command("git", "commit", "--amend", "-F", "-")
			cmd.Stdin = bytes.NewBufferString(commit.Message)
			cmd.Stderr = stderr
			if err := cmd.Run(); err != nil {
				return commits, errs.NewError(msg, stderr, err)
			}
		}
	}

	// Reset the current branch to point to the new branch.
	msg = "Reset the current branch to point to the temporary branch"
	if stderr, err := git.ResetKeep(currentBranch, tempBranch); err != nil {
		return commits, errs.NewError(msg, stderr, err)
	}

	// Parse the commits again since the commit hashes have changed.
	newCommits, stderr, err := git.ShowCommitRange(parentSHA + "..")
	if err != nil {
		return commits, handleError(msg, err, stderr)
	}

	// And we are done!
	return newCommits, nil
}

func mustBumpCommits(writer io.Writer, commits []*git.Commit, prefix string) {
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
		tool      = modules.GetCodeReviewTool()
		topErr    error
		topErrMux sync.Mutex
	)

	var wg sync.WaitGroup
	wg.Add(len(commits))
	for _, commit := range commits {
		go func(commit *git.Commit) {
			defer wg.Done()
			msg := "Post the review request for commit " + commit.SHA
			log.Go(msg)
			if err := tool.PostReviewRequest(commit); err != nil {
				errs.LogFail(msg, err)
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
	fmt.Println("\n----------")
	modules.GetCodeReviewTool().PrintPostReviewRequestFollowup()
	fmt.Print(`  ###########################################################
  # IMPORTANT: Your code has not been merged and/or pushed. #
  ###########################################################

`)
}
