package startCmd

import (
	// Stdlib
	"errors"
	"fmt"
	"os"

	// Internal
	"github.com/salsaflow/salsaflow/action"
	"github.com/salsaflow/salsaflow/app"
	"github.com/salsaflow/salsaflow/app/appflags"
	"github.com/salsaflow/salsaflow/errs"
	"github.com/salsaflow/salsaflow/git"
	"github.com/salsaflow/salsaflow/git/gitutil"
	"github.com/salsaflow/salsaflow/log"
	"github.com/salsaflow/salsaflow/modules"
	"github.com/salsaflow/salsaflow/modules/common"
	"github.com/salsaflow/salsaflow/prompt"
	"github.com/salsaflow/salsaflow/prompt/storyprompt"

	// Other
	"github.com/extemporalgenome/slug"
	"gopkg.in/tchap/gocli.v2"
)

var Command = &gocli.Command{
	UsageLine: "start [-base=BASE] [-no_branch] [-push]",
	Short:     "start a new story",
	Long: `
  Start a new issue tracker story.

  The user is shown the list of stories that can be started.
  When they choose one, they are assigned to that story and the story
  is started in the issue tracker.

  Unless -no_branch is specified, the user is asked to insert
  the branch name to be used for the branch holding the story commits.
  The branch of the given name is created on top of the trunk branch
  and checked out. A custom base branch can be set by using -base.
  The story branch is then pushed in case -push is specified.
	`,
	Action: run,
}

var (
	flagBase     string
	flagNoBranch bool
	flagPush     bool
)

func init() {
	// Register flags.
	Command.Flags.StringVar(&flagBase, "base", flagBase,
		"the branch to base the story branch on")
	Command.Flags.BoolVar(&flagNoBranch, "no_branch", flagNoBranch,
		"do not create a new story branch")
	Command.Flags.BoolVar(&flagPush, "push", flagPush,
		"push the newly created story branch")

	// Register global flags.
	appflags.RegisterGlobalFlags(&Command.Flags)
}

func run(cmd *gocli.Command, args []string) {
	if len(args) != 0 {
		cmd.Usage()
		os.Exit(2)
	}

	app.InitOrDie()

	defer prompt.RecoverCancel()

	if err := runMain(); err != nil {
		errs.Fatal(err)
	}
}

func runMain() error {
	tracker, err := modules.GetIssueTracker()
	if err != nil {
		return err
	}

	// Fetch stories from the issue tracker.
	task := "Fetch stories from the issue tracker"
	log.Run(task)
	stories, err := tracker.StartableStories()
	if err != nil {
		return errs.NewError(task, err)
	}
	if len(stories) == 0 {
		return errs.NewError(task, errors.New("no startable stories found"))
	}

	// Filter out the stories that are not relevant,
	// i.e. not owned by the current user or assigned to someone else.
	task = "Fetch the current user record from the issue tracker"
	user, err := tracker.CurrentUser()
	if err != nil {
		return errs.NewError(task, err)
	}

	var filteredStories []common.Story
StoryLoop:
	for _, story := range stories {
		assignees := story.Assignees()
		// Include the story in case there is no assignee set yet.
		if len(assignees) == 0 {
			filteredStories = append(filteredStories, story)
			continue StoryLoop
		}
		// Include the story in case the current user is assigned.
		for _, assignee := range assignees {
			if assignee.Id() == user.Id() {
				filteredStories = append(filteredStories, story)
				continue StoryLoop
			}
		}
	}
	stories = filteredStories

	// Prompt the user to select a story.
	story, err := dialog(
		"\nYou can start working on one of the following stories:", stories)
	if err != nil {
		switch err {
		case prompt.ErrNoStories:
			return errors.New("no startable stories found")
		case prompt.ErrCanceled:
			prompt.PanicCancel()
		default:
			return err
		}
	}
	fmt.Println()

	// Create the story branch, optionally.
	if flagNoBranch {
		log.Log("Not creating any feature branch")
	} else {
		var act action.Action
		act, err = createBranch()
		if err != nil {
			return err
		}
		// Roll back on error.
		defer action.RollbackTaskOnError(&err, task, act)
	}

	// Add the current user to the list of story assignees.
	task = "Amend the list of story assignees"
	log.Run(task)
	originalAssignees := story.Assignees()
	if err := story.AddAssignee(user); err != nil {
		return errs.NewError(task, err)
	}
	defer action.RollbackTaskOnError(&err, task, action.ActionFunc(func() error {
		task := "Reset the list of story assignees"
		if err := story.SetAssignees(originalAssignees); err != nil {
			return errs.NewError(task, err)
		}
		return nil
	}))

	// Start the selected story. No need to roll back.
	task = "Start the selected story"
	log.Run(task)
	if err := story.Start(); err != nil {
		return errs.NewError(task, err)
	}

	return nil
}

func createBranch() (action.Action, error) {
	// Get the current branch name.
	originalBranch, err := gitutil.CurrentBranch()
	if err != nil {
		return nil, err
	}

	// Fetch the remote repository.
	task := "Fetch the remote repository"
	log.Run(task)

	gitConfig, err := git.LoadConfig()
	if err != nil {
		return nil, errs.NewError(task, err)
	}

	var (
		remoteName = gitConfig.RemoteName
		baseBranch = gitConfig.TrunkBranchName
	)
	if flagBase != "" {
		baseBranch = flagBase
	}

	// Fetch the remote repository.
	if err := git.UpdateRemotes(remoteName); err != nil {
		return nil, errs.NewError(task, err)
	}

	// Make sure the trunk branch is up to date.
	task = fmt.Sprintf("Make sure branch '%v' is up to date", baseBranch)
	log.Run(task)
	if err := git.CheckOrCreateTrackingBranch(baseBranch, remoteName); err != nil {
		return nil, errs.NewError(task, err)
	}

	// Prompt the user for the branch name.
	task = "Prompt the user for the branch name"
	line, err := prompt.Prompt(`
Please insert the branch slug now.
Insert an empty string to skip the branch creation step: `)
	if err != nil && err != prompt.ErrCanceled {
		return nil, errs.NewError(task, err)
	}

	sluggedLine := slug.Slug(line)
	if sluggedLine == "" {
		fmt.Println()
		log.Log("Not creating any feature branch")
		return nil, nil
	}

	branchName := "story/" + sluggedLine
	ok, err := prompt.Confirm(
		fmt.Sprintf(
			"\nThe branch that is going to be created will be called '%s'.\nIs that alright?",
			branchName),
		true)
	if err != nil {
		return nil, errs.NewError(task, err)
	}
	if !ok {
		panic(prompt.ErrCanceled)
	}
	fmt.Println()

	createTask := fmt.Sprintf(
		"Create branch '%v' on top of branch '%v'", branchName, baseBranch)
	log.Run(createTask)
	if err := git.Branch(branchName, baseBranch); err != nil {
		return nil, errs.NewError(createTask, err)
	}

	deleteTask := fmt.Sprintf("Delete branch '%v'", branchName)
	deleteBranch := func() error {
		// Roll back and delete the newly created branch.
		log.Rollback(createTask)
		if err := git.Branch("-D", branchName); err != nil {
			return errs.NewError(deleteTask, err)
		}
		return nil
	}

	// Checkout the newly created branch.
	checkoutTask := fmt.Sprintf("Checkout branch '%v'", branchName)
	log.Run(checkoutTask)
	if err := git.Checkout(branchName); err != nil {
		if err := deleteBranch(); err != nil {
			errs.Log(err)
		}
		return nil, errs.NewError(checkoutTask, err)
	}

	// Push the newly created branch unless -no_push.
	pushTask := fmt.Sprintf("Push branch '%v' to remote '%v'", branchName, remoteName)
	if flagPush {
		log.Run(pushTask)
		if err := git.Push(remoteName, branchName); err != nil {
			if err := deleteBranch(); err != nil {
				errs.Log(err)
			}
			return nil, errs.NewError(pushTask, err)
		}
	}

	return action.ActionFunc(func() error {
		// Checkout the original branch.
		log.Rollback(checkoutTask)
		if err := git.Checkout(originalBranch); err != nil {
			return errs.NewError(
				fmt.Sprintf("Checkout the original branch '%v'", originalBranch), err)
		}

		// Delete the newly created branch.
		deleteErr := deleteBranch()

		// In case we haven't pushed anything, we are done.
		if !flagPush {
			return deleteErr
		}

		// Delete the branch from the remote repository.
		log.Rollback(pushTask)
		if _, err := git.Run("push", "--delete", remoteName, branchName); err != nil {
			// In case deleteBranch failed, tell the user now
			// since we are not going to return that error.
			if deleteErr != nil {
				errs.Log(deleteErr)
			}

			return errs.NewError(
				fmt.Sprintf("Delete branch '%v' from remote '%v'", branchName, remoteName), err)
		}

		// Return deleteErr to make sure it propagates up.
		return deleteErr
	}), nil
}

func dialog(msg string, stories []common.Story) (common.Story, error) {
	fmt.Println(msg)
	fmt.Println()

	dialog := storyprompt.NewDialog()
	dialog.PushOptions(storyprompt.NewIndexOption())
	dialog.PushOptions(storyprompt.NewReturnOrAbortOptions()...)
	dialog.PushOptions(storyprompt.NewFilterOption())
	return dialog.Run(stories)
}
