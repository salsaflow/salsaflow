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
	"github.com/salsaflow/salsaflow/action"
	"github.com/salsaflow/salsaflow/asciiart"
	"github.com/salsaflow/salsaflow/commands/review/post/constants"
	"github.com/salsaflow/salsaflow/errs"
	"github.com/salsaflow/salsaflow/git"
	"github.com/salsaflow/salsaflow/git/gitutil"
	"github.com/salsaflow/salsaflow/log"
	"github.com/salsaflow/salsaflow/modules"
	"github.com/salsaflow/salsaflow/modules/common"
	"github.com/salsaflow/salsaflow/prompt"
	"github.com/salsaflow/salsaflow/prompt/storyprompt"
)

var ErrNoCommits = errors.New("no commits selected for code review")

func ensureNoMergeCommits(commits []*git.Commit) error {
	var (
		task = "Make sure there are no merge commits"
		hint bytes.Buffer
		err  error
	)
	fmt.Fprintln(&hint)
	for _, commit := range commits {
		if commit.Merge != "" {
			fmt.Fprintf(&hint, "Commit %v is a merge commit\n", commit.SHA)
			err = errors.New("merge commit detected")
		}
	}
	fmt.Fprintln(&hint)
	if err != nil {
		return errs.NewErrorWithHint(task, err, hint.String())
	}
	return nil
}

func promptUserToConfirmCommits(commits []*git.Commit) error {
	// Make sure there are actually some commits to be posted.
	task := "Make sure there are actually some commits to be posted"
	if len(commits) == 0 {
		return errs.NewError(task, ErrNoCommits)
	}

	// Tell the user what is going to happen.
	fmt.Print(`
You are about to post some of the following commits for code review:

`)
	mustListCommits(os.Stdout, commits, "  ")

	// Ask the user for confirmation.
	task = "Prompt the user for confirmation"
	confirmed, err := prompt.Confirm("\nYou cool with that?", true)
	if err != nil {
		return errs.NewError(task, err)
	}
	if !confirmed {
		prompt.PanicCancel()
	}
	fmt.Println()
	return nil
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

func ensureStoryId(commits []*git.Commit) ([]*git.Commit, error) {
	task := "Make sure the commits comply with the rules"
	var err error
	if isStoryIdMissing(commits) {
		commits, err = rewriteCommits(commits)
		if err != nil {
			return nil, errs.NewError(task, err)
		}
	} else {
		log.Log("Commit check passed")
	}
	return commits, nil
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

func rewriteCommits(commits []*git.Commit) ([]*git.Commit, error) {
	// Fetch the stories in progress from the issue tracker.
	storiesTask := "Missing Story-Id detected, fetch stories from the issue tracker"
	log.Run(storiesTask)

	tracker, err := modules.GetIssueTracker()
	if err != nil {
		return nil, errs.NewError(storiesTask, err)
	}

	task := "Fetch the user record from the issue tracker"
	me, err := tracker.CurrentUser()
	if err != nil {
		return nil, errs.NewError(task, err)
	}

	stories, err := tracker.ReviewableStories()
	if err != nil {
		return nil, errs.NewError(storiesTask, err)
	}

	reviewedStories, err := tracker.ReviewedStories()
	if err != nil {
		return nil, errs.NewError(storiesTask, err)
	}

	// Show only the stories owned by the current user.
	// Note: Go sucks here, badly.
	filterStories := func(stories []common.Story, filter func(common.Story) bool) []common.Story {
		ss := make([]common.Story, 0, len(stories))
		for _, story := range stories {
			if filter(story) {
				ss = append(ss, story)
			}
		}
		return ss
	}

	mine := func(story common.Story) bool {
		for _, assignee := range story.Assignees() {
			if assignee.Id() == me.Id() {
				return true
			}
		}
		return false
	}

	stories = filterStories(stories, mine)
	reviewedStories = filterStories(reviewedStories, mine)

	// Tell the user what is happening.
	log.Run("Prepare a temporary branch to rewrite commit messages")

	// Get the current branch name.
	currentBranch, err := gitutil.CurrentBranch()
	if err != nil {
		return nil, err
	}

	// Get the parent of the first commit in the chain.
	task = "Get the parent commit of the commit chain to be posted"
	stdout, err := git.Log("--pretty=%P", "-n", "1", commits[0].SHA)
	if err != nil {
		return nil, errs.NewError(task, err)
	}
	parentSHA := strings.Fields(stdout.String())[0]

	// Prepare a temporary branch that will be used to amend commit messages.
	task = "Create a temporary branch to rewrite commit messages"
	if err := git.SetBranch(constants.TempBranchName, parentSHA); err != nil {
		return nil, errs.NewError(task, err)
	}
	defer func() {
		// Delete the temporary branch on exit.
		task := "Delete the temporary branch"
		if err := git.Branch("-D", constants.TempBranchName); err != nil {
			errs.LogError(task, err)
		}
	}()

	// Checkout the temporary branch.
	task = "Checkout the temporary branch"
	if err := git.Checkout(constants.TempBranchName); err != nil {
		return nil, errs.NewError(task, err)
	}
	defer func() {
		// Checkout the original branch on exit.
		task := fmt.Sprintf("Checkout branch '%v'", currentBranch)
		if err := git.Checkout(currentBranch); err != nil {
			errs.LogError(task, err)
		}
	}()

	// Loop and rewrite the commit messages.
	var story common.Story
	if flagAskOnce {
		header := `
Some of the commits listed above are not assigned to any story.
Please pick up the story that these commits will be assigned to.
You can also insert 'u' to mark the commits as unassigned:`
		selectedStory, err := promptForStory(header, stories, reviewedStories)
		if err != nil {
			return nil, err
		}
		story = selectedStory
	}

	for _, commit := range commits {
		// Cherry-pick the commit.
		task := fmt.Sprintf("Move commit %v onto the temporary branch", commit.SHA)
		if err := git.CherryPick(commit.SHA); err != nil {
			return nil, errs.NewError(task, err)
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
Inserting 'u' will mark the commit as unassigned:`, commit.SHA, commitMessageTitle)
				selectedStory, err := promptForStory(header, stories, reviewedStories)
				if err != nil {
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
				return nil, errs.NewErrorWithHint(task, err, stderr.String())
			}
		}
	}

	// Reset the current branch to point to the new branch.
	task = "Reset the current branch to point to the temporary branch"
	if err := git.SetBranch(currentBranch, constants.TempBranchName); err != nil {
		return nil, errs.NewError(task, err)
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

func push(remote, branch string) error {
	task := fmt.Sprintf("Push branch '%v' to remote '%v'", branch, remote)

	args := make([]string, 0, 3)
	msg := fmt.Sprintf("Pushing branch '%v' to synchronize", branch)
	isCore, err := git.IsCoreBranch(branch)
	if err != nil {
		return nil, err
	}
	if !isCore {
		args = append(args, "-f")
		msg += " (using force)"
	}

	args = append(args, remote, branch)

	log.Log(msg)
	if _, err = git.RunCommand("push", args...); err != nil {
		return nil, errs.NewError(task, err)
	}
	return nil
}

func postReviewRequests(commits []*git.Commit, canAmend bool) (act action.Action, err error) {
	// Pick the commits to be posted for review.
	if flagPick {
		task = "Select the commits to be posted for review"
		commits, err = selectCommitsForReview(commits)
		if err != nil {
			return nil, errs.NewError(task, err)
		}

		if len(commits) == 0 {
			log.NewLine("")
			log.Log("No commits selected, aborting...")
			prompt.PanicCancel()
		}
	}

	// Print Snoopy.
	asciiart.PrintSnoopy()

	// Turn Commits into ReviewContexts.
	task = "Fetch stories for the commits to be posted for review"
	log.Run(task)
	ctxs, err := commitsToReviewContexts(commits)
	if err != nil {
		return nil, errs.NewError(task, err)
	}

	// Mark the stories as implemented, potentially.
	task = "Mark the stories as implemented, optionally"
	implemented, act, err := implementedDialog(ctxs)
	if err != nil {
		return nil, errs.NewError(task, err)
	}
	defer action.RollbackTaskOnError(&err, task, act)

	// Post the review requests.
	task = "Post the review requests"
	if err := sendReviewRequests(ctxs, implemented); err != nil {
		return nil, errs.NewError(task, err)
	}

	return act, nil
}

func selectCommitsForReview(commits []*git.Commit) ([]*git.Commit, error) {
	cs := make([]*git.Commit, 0, len(commits))

	fmt.Println("\nSelect the commits to be posted for code review (-pick flag set):\n")

	for _, commit := range commits {
		selected, err := prompt.Confirm(
			fmt.Sprintf("  %v | %v", commit.SHA, commit.MessageTitle), true)
		if err != nil {
			return nil, err
		}
		if selected {
			cs = append(cs, commit)
		}
	}

	return cs, nil
}

func commitsToReviewContexts(commits []*git.Commit) ([]*common.ReviewContext, error) {
	tracker, err := modules.GetIssueTracker()
	if err != nil {
		return nil, err
	}

	// Fetch the stories from the issue tracker.
	tags := storyTags(tracker, commits)
	stories, err := tracker.ListStoriesByTag(tags)
	if err != nil {
		return nil, err
	}

	// Build the story map.
	storiesByTag := make(map[string]common.Story, 1)
	for i, story := range stories {
		tag := tags[i]
		if story == nil {
			log.Warn(fmt.Sprintf("Story for tag '%v' was not found in the issue tracker", tag))
			continue
		}
		storiesByTag[tag] = story
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

func storyTags(tracker common.IssueTracker, commits []*git.Commit) (tags []string) {
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

	return storyTagList
}

func sendReviewRequests(ctxs []*common.ReviewContext, implemented bool) error {
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
	if flagReviewer != "" {
		postOpts["reviewer"] = flagReviewer
	}
	if flagOpen {
		postOpts["open"] = true
	}
	if implemented {
		postOpts["implemented"] = true
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
			return errs.NewError(task, err)
		}
		return nil
	}

	// Post the review for the whole branch.
	currentBranch, err := gitutil.CurrentBranch()
	if err != nil {
		return err
	}

	task := fmt.Sprintf("Post review request for branch '%v'", currentBranch)
	if err := tool.PostReviewRequests(ctxs, postOpts); err != nil {
		return errs.NewError(task, err)
	}

	return nil
}

func printFollowup() error {
	task := "Print the followup message"
	tool, err := modules.GetCodeReviewTool()
	if err != nil {
		return errs.NewError(task, err)
	}

	log.Println("\n----------")
	log.Println(tool.PostReviewFollowupMessage())
	return nil
}

func implementedDialog(ctxs []*common.ReviewContext) (implemented bool, act action.Action, err error) {
	// Collect the affected stories.
	var (
		stories  = make([]common.Story, 0, len(ctxs))
		storySet = make(map[string]struct{}, len(ctxs))
	)
	for _, ctx := range ctxs {
		story := ctx.Story
		// Skip unassigned commits.
		if story == nil {
			continue
		}
		rid := story.ReadableId()
		if _, ok := storySet[rid]; ok {
			continue
		}
		// Collect only the stories that are Being Implemented.
		// The transition doesn't make sense for other story states.
		if story.State() != common.StoryStateBeingImplemented {
			continue
		}
		storySet[rid] = struct{}{}
		stories = append(stories, story)
	}
	// Do nothing in case there are no stories left.
	if len(stories) == 0 {
		return false, nil, nil
	}

	// Prompt the user for confirmation.
	fmt.Println("\nIt is possible to mark the affected stories as implemented.")
	fmt.Println("The following stories were associated with one or more commits:\n")
	storyprompt.ListStories(stories, os.Stdout)
	fmt.Println()
	confirmed, err := prompt.Confirm(
		"Do you wish to mark these stories as implemented?", false)
	if err != nil {
		return false, nil, err
	}
	fmt.Println()
	if !confirmed {
		return false, nil, nil
	}

	// Always update as many stories as possible.
	var (
		chain           = action.NewActionChain()
		errUpdateFailed = errors.New("failed to update stories in the issue tracker")
		ex              error
	)
	for _, story := range stories {
		task := fmt.Sprintf("Mark story %v as implemented", story.ReadableId())
		log.Run(task)
		act, err := story.MarkAsImplemented()
		if err != nil {
			errs.Log(errs.NewError(task, err))
			ex = errUpdateFailed
			continue
		}
		chain.PushTask(task, act)
	}
	if ex != nil {
		if err := chain.Rollback(); err != nil {
			errs.Log(err)
		}
		return false, nil, ex
	}

	return true, chain, nil
}

func promptForStory(
	header string,
	stories []common.Story,
	reviewedStories []common.Story,
) (common.Story, error) {

	var (
		unassignedOpt *storyprompt.DialogOption
		reviewedOpt   *storyprompt.DialogOption
	)

	pushOptions := func(dialog *storyprompt.Dialog, opts ...*storyprompt.DialogOption) {
		dialog.PushOptions(storyprompt.NewIndexOption())
		dialog.PushOptions(opts...)
		dialog.PushOptions(storyprompt.NewReturnOrAbortOptions()...)
		dialog.PushOptions(storyprompt.NewFilterOption())
	}

	unassignedOpt = &storyprompt.DialogOption{
		Description: []string{
			"Insert 'u' to mark the commit(s) as unassigned",
		},
		IsActive: func(stories []common.Story, depth int) bool {
			return true
		},
		MatchesInput: func(input string, stories []common.Story) bool {
			return input == "u"
		},
		SelectStory: func(
			input string,
			stories []common.Story,
			currentDialog *storyprompt.Dialog,
		) (common.Story, error) {

			return nil, nil
		},
	}

	reviewedOpt = &storyprompt.DialogOption{
		Description: []string{
			"Insert 'r' to select a reviewed story",
		},
		IsActive: func(stories []common.Story, depth int) bool {
			return depth == 1
		},
		MatchesInput: func(input string, stories []common.Story) bool {
			return input == "r"
		},
		SelectStory: func(
			input string,
			stories []common.Story,
			currentDialog *storyprompt.Dialog,
		) (common.Story, error) {

			fmt.Println()
			fmt.Println("Showing the stories that are already reviewed ...")
			fmt.Println()

			subdialog := currentDialog.NewSubdialog()
			pushOptions(subdialog, unassignedOpt)
			return subdialog.Run(reviewedStories)
		},
	}

	fmt.Println(header)
	fmt.Println()

	dialog := storyprompt.NewDialog()
	pushOptions(dialog, unassignedOpt, reviewedOpt)
	return dialog.Run(stories)
}
