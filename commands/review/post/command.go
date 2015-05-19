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
	"text/tabwriter"

	// Internal
	"github.com/salsaflow/salsaflow/app"
	"github.com/salsaflow/salsaflow/asciiart"
	"github.com/salsaflow/salsaflow/commands/review/post/constants"
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

  post [-fixes=RRID] [-no_fetch] [-no_rebase] [-ask_once]
       [-no_seq] [-open] [-no_dialog] -parent=BRANCH`,
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
	flagNoDialog bool
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
	Command.Flags.BoolVar(&flagNoDialog, "no_dialog", flagNoDialog,
		"skip the followup dialog in case -parent is being used")
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

	defer prompt.RecoverCancel()

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
	// "HEAD" is used to post the tip of the current branch.
	headMode := revision == "HEAD"

	// Get the commit to be posted
	task := "Get the commit to be posted for code review"
	commits, err := git.ShowCommits(revision)
	if err != nil {
		return errs.NewError(task, err, nil)
	}

	// Assert that things are consistent.
	if numCommits := len(commits); numCommits != 1 {
		panic(fmt.Sprintf("len(commits): expected 1, got %v", numCommits))
	}

	// Post the review requests, in this case it will be only one.
	if err := postReviewRequests(commits, headMode); err != nil {
		return err
	}

	// In case there is no error, tell the user what to do next.
	return printFollowup()
}

func postBranch(parentBranch string) error {
	// Load the git-related config.
	gitConfig, err := git.LoadConfig()
	if err != nil {
		return err
	}
	var (
		remoteName  = gitConfig.RemoteName()
		trunkBranch = gitConfig.TrunkBranchName()
	)

	// Get the current branch name.
	currentBranch, err := git.CurrentBranch()
	if err != nil {
		return err
	}

	if !flagNoFetch {
		// Fetch the remote repository.
		task := "Fetch the remote repository"
		log.Run(task)

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
	if err := postReviewRequests(commits, true); err != nil {
		return err
	}

	// Just print the regular followup in case the dialog is disabled.
	if flagNoDialog {
		return printFollowup()
	}

	// Ask the user what to do next.
	return parentFollowupDialog(currentBranch, remoteName, trunkBranch)
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
		prompt.PanicCancel()
	}
	fmt.Println()

	// Check the commits.
	task = "Make sure the commits comply with the rules"
	if isStoryIdMissing(commits) {
		commits, err = rewriteCommits(commits, canAmend)
		if err != nil {
			return errs.NewError(task, err, nil)
		}

		// Push the branch in case we are on a branch tip.
		// We are on a branch tip when canAmend is true.
		// Use force in case we are not on any SF core branch.
		if canAmend {
			current, err := git.CurrentBranch()
			if err != nil {
				return err
			}

			args := make([]string, 0, 1)
			msg := "Current branch rewritten, pushing to synchronize"
			isCore, err := git.IsCoreBranch(current)
			if err != nil {
				return err
			}
			if !isCore {
				args = append(args, "-f")
				msg += " (using force)"
			}
			log.Log(msg)
			if _, err = git.RunCommand("push", args...); err != nil {
				return errs.NewError("Push the current branch", err, nil)
			}
		}
	} else {
		log.Log("Commit check passed")
	}

	// Print Snoopy.
	asciiart.PrintSnoopy()

	// Turn Commits into ReviewContexts.
	task = "Fetch stories for the commits to be posted for review"
	log.Run(task)
	ctxs, err := commitsToReviewContexts(commits)
	if err != nil {
		return errs.NewError(task, err, nil)
	}

	// Post the review requests.
	task = "Post the review requests"
	if err := sendReviewRequests(ctxs); err != nil {
		return errs.NewError(task, err, nil)
	}

	return nil
}

func isStoryIdMissing(commits []*git.Commit) bool {
	for _, commit := range commits {
		if commit.Merge != "" {
			continue
		}
		if commit.StoryIdTag != "" {
			continue
		}
		return true
	}
	return false
}

func rewriteCommits(commits []*git.Commit, canAmend bool) ([]*git.Commit, error) {
	// Make sure we are not posting any merge commits.
	var noMergeCommits []*git.Commit
	for _, commit := range commits {
		if commit.Merge == "" {
			noMergeCommits = append(noMergeCommits, commit)
		}
	}

	// Again, make sure there are actually some commits to be posted.
	if len(noMergeCommits) == 0 {
		return nil, ErrNoCommits
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
		return nil, errs.NewError(
			"Make sure the commits can be amended",
			errors.New("Story-Id tag missing"),
			bytes.NewBufferString(hint))
	}

	// Fetch the stories in progress from the issue tracker.
	storiesTask := "Missing Story-Id detected, fetch stories from the issue tracker"
	log.Run(storiesTask)

	tracker, err := modules.GetIssueTracker()
	if err != nil {
		return nil, errs.NewError(storiesTask, err, nil)
	}

	task := "Fetch the user record from the issue tracker"
	me, err := tracker.CurrentUser()
	if err != nil {
		return nil, errs.NewError(task, err, nil)
	}

	stories, err := tracker.StoriesInDevelopment()
	if err != nil {
		return nil, errs.NewError(storiesTask, err, nil)
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
		return nil, err
	}

	// Get the parent of the first commit in the chain.
	task = "Get the parent commit of the commit chain to be posted"
	stdout, err := git.Log("--pretty=%P", "-n", "1", commits[0].SHA)
	if err != nil {
		return nil, errs.NewError(task, err, nil)
	}
	parentSHA := strings.Fields(stdout.String())[0]

	// Prepare a temporary branch that will be used to amend commit messages.
	task = "Create a temporary branch to rewrite commit messages"
	if err := git.Branch("-f", constants.TempBranchName, parentSHA); err != nil {
		return nil, errs.NewError(task, err, nil)
	}
	defer func() {
		// Delete the temporary branch on exit.
		task := "Delete the temporary branch"
		if err := git.Branch("-D", constants.TempBranchName); err != nil {
			errs.LogError(task, err, nil)
		}
	}()

	// Checkout the temporary branch.
	task = "Checkout the temporary branch"
	if err := git.Checkout(constants.TempBranchName); err != nil {
		return nil, errs.NewError(task, err, nil)
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
Please pick up the story that these commits will be assigned to.
You can also insert '0' to mark the commits as unassigned:`
		selectedStory, err := prompt.PromptStoryAllowNone(header, stories)
		if err != nil {
			switch err {
			case prompt.ErrNoStories:
				hint := `
There are no stories that the unassigned commits can be assigned to.
In other words, there are no stories in the right state for that.

`
				return nil, errs.NewError(task, err, bytes.NewBufferString(hint))
			case prompt.ErrCanceled:
				prompt.PanicCancel()
			default:
				return nil, err
			}
		}
		story = selectedStory
	}

	for _, commit := range commits {
		// Cherry-pick the commit.
		task := fmt.Sprintf("Move commit %v onto the temporary branch", commit.SHA)
		if err := git.CherryPick(commit.SHA); err != nil {
			return nil, errs.NewError(task, err, nil)
		}

		if commit.StoryIdTag == "" {
			if !flagAskOnce {
				commitMessageTitle := prompt.ShortenCommitTitle(commit.MessageTitle)

				// Ask for the story ID for the current commit.
				header := fmt.Sprintf(`
The following commit is not assigned to any story:

  commit hash:  %v
  commit title: %v

Please pick up the story to assign the commit to.
Inserting '0' will mark the commit as unassigned:`, commit.SHA, commitMessageTitle)
				selectedStory, err := prompt.PromptStoryAllowNone(header, stories)
				if err != nil {
					if err == prompt.ErrCanceled {
						panic(err)
					}
					return nil, err
				}
				story = selectedStory
			}

			// Use the unassigned tag value in case no story is selected.
			storyTag := git.StoryIdUnassignedTagValue
			if story != nil {
				storyTag = story.Tag()
			}

			// Extend the commit message to include Story-Id.
			commitMessage := fmt.Sprintf("%v\nStory-Id: %v\n", commit.Message, storyTag)

			// Amend the cherry-picked commit to include the new commit message.
			task = "Amend the commit message for " + commit.SHA
			stderr := new(bytes.Buffer)
			cmd := exec.Command("git", "commit", "--amend", "-F", "-")
			cmd.Stdin = bytes.NewBufferString(commitMessage)
			cmd.Stderr = stderr
			if err := cmd.Run(); err != nil {
				return nil, errs.NewError(task, err, stderr)
			}
		}
	}

	// Reset the current branch to point to the new branch.
	task = "Reset the current branch to point to the temporary branch"
	if err := git.ResetKeep(currentBranch, constants.TempBranchName); err != nil {
		return nil, errs.NewError(task, err, nil)
	}

	// Parse the commits again since the commit hashes have changed.
	newCommits, err := git.ShowCommitRange(parentSHA + "..")
	if err != nil {
		return nil, err
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
		commitMessageTitle := prompt.ShortenCommitTitle(commit.MessageTitle)
		must(fmt.Fprintf(tw, "%v%v\t%v\n", prefix, commit.SHA, commitMessageTitle))
	}

	must(0, tw.Flush())
}

func commitsToReviewContexts(commits []*git.Commit) ([]*common.ReviewContext, error) {
	tracker, err := modules.GetIssueTracker()
	if err != nil {
		return nil, err
	}

	// Collect the Story-Id tags.
	var (
		storyTagSet  = make(map[string]struct{}, 1)
		storyTagList = make([]string, 0, 1)
	)
	for _, commit := range commits {
		tag := commit.StoryIdTag

		// Skip empty tags.
		if tag == "" {
			continue
		}

		// Skip unassigned stories.
		if _, err := tracker.StoryTagToReadableStoryId(tag); err != nil {
			continue
		}

		// Otherwise register the tag, unless already registered.
		if _, ok := storyTagSet[tag]; ok {
			continue
		}
		storyTagSet[tag] = struct{}{}
		storyTagList = append(storyTagList, tag)
	}

	// Fetch the stories from the issue tracker.
	stories, err := tracker.ListStoriesByTag(storyTagList)
	if err != nil {
		return nil, err
	}

	// Build the story map.
	storiesByTag := make(map[string]common.Story, 1)
	for _, story := range stories {
		storiesByTag[story.Tag()] = story
	}

	// Build the final list of review contexts.
	ctxs := make([]*common.ReviewContext, 0, len(commits))
	for _, commit := range commits {
		// Story can be set to nil here in case the story is unassigned.
		// In that case there will be, obviously, no story object in the map.
		ctxs = append(ctxs, &common.ReviewContext{
			Commit: commit,
			Story:  storiesByTag[commit.StoryIdTag],
		})
	}

	// Return the commit review contexts.
	return ctxs, nil
}

func sendReviewRequests(ctxs []*common.ReviewContext) error {
	// Instantiate the code review module.
	tool, err := modules.GetCodeReviewTool()
	if err != nil {
		return err
	}

	// Collect the command line flags into a map.
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

	// Only post a single commit in case -parent is not being used.
	// By definition it must be only a single commit anyway.
	if flagParent == "" {
		if len(ctxs) != 1 {
			panic(fmt.Sprintf("len(ctxs): expected 1, got %v", len(ctxs)))
		}

		task := "Post review request for commit " + ctxs[0].Commit.SHA
		log.Run(task)
		if err := tool.PostReviewRequests(ctxs, postOpts); err != nil {
			return errs.NewError(task, err, nil)
		}
		return nil
	}

	// Post the review for the whole branch.
	currentBranch, err := git.CurrentBranch()
	if err != nil {
		return err
	}

	task := fmt.Sprintf("Post review request for branch '%v'", currentBranch)
	if err := tool.PostReviewRequests(ctxs, postOpts); err != nil {
		return errs.NewError(task, err, nil)
	}

	return nil
}

func printFollowup() error {
	task := "Print the followup message"
	tool, err := modules.GetCodeReviewTool()
	if err != nil {
		return errs.NewError(task, err, nil)
	}

	log.Println("\n----------")
	log.Println(tool.PostReviewFollowupMessage())
	return nil
}

func parentFollowupDialog(currentBranch, remoteName, trunkBranch string) error {
	// Decide whether to merge the current branch into trunk or not.
	var trunkModified bool
	fmt.Printf(`
You might want to merge the current branch (%v) into trunk, right?

  1) do nothing
  2) merge into trunk
  3) merge into trunk (--no-ff)

`, currentBranch)
	index, err := prompt.PromptIndex("Choose [1-3]: ", 1, 3)
	if err != nil {
		return err
	}
	switch index {
	case 1:
	case 2:
		fmt.Println()
		if err := merge(currentBranch, trunkBranch); err != nil {
			return err
		}
		trunkModified = true
	case 3:
		fmt.Println()
		if err := merge(currentBranch, trunkBranch, "--no-ff"); err != nil {
			return err
		}
		trunkModified = true
	}

	// Decide whether to push trunk or not.
	if !trunkModified {
		return nil
	}
	fmt.Print(`
Now that the trunk branch has been modified, you might want to push it, right?

  1) do nothing
  2) push

`)
	index, err = prompt.PromptIndex("Choose [1-2]: ", 1, 2)
	if err != nil {
		return err
	}
	switch index {
	case 1:
	case 2:
		fmt.Println()
		task := fmt.Sprintf("Push branch '%v'", trunkBranch)
		log.Run(task)
		err := git.Push(remoteName, fmt.Sprintf("%v:%v", trunkBranch, trunkBranch))
		if err != nil {
			return errs.NewError(task, err, nil)
		}
	}

	return nil
}

// merge merges commit into branch.
func merge(commit, branch string, flags ...string) error {
	task := fmt.Sprintf("Merge '%v' into branch '%v'", commit, branch)
	log.Run(task)

	currentBranch, err := git.CurrentBranch()
	if err != nil {
		return errs.NewError(task, err, nil)
	}

	if err := git.Checkout(branch); err != nil {
		return errs.NewError(task, err, nil)
	}

	args := make([]string, 1, 1+len(flags))
	args[0] = commit
	args = append(args, flags...)
	if _, err := git.RunCommand("merge", args...); err != nil {
		return errs.NewError(task, err, nil)
	}

	if err := git.Checkout(currentBranch); err != nil {
		return errs.NewError(task, err, nil)
	}
	return nil
}
