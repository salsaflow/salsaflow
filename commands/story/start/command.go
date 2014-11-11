package startCmd

import (
	// Stdlib
	"errors"
	"fmt"
	"os"

	// Internal
	"github.com/salsita/salsaflow/app"
	"github.com/salsita/salsaflow/errs"
	"github.com/salsita/salsaflow/git"
	"github.com/salsita/salsaflow/log"
	"github.com/salsita/salsaflow/modules"
	"github.com/salsita/salsaflow/modules/common"
	"github.com/salsita/salsaflow/prompt"

	// Other
	"github.com/extemporalgenome/slug"
	"gopkg.in/tchap/gocli.v1"
)

var Command = &gocli.Command{
	UsageLine: "start [-no_branch]",
	Short:     "start a new story",
	Long: `
Start a new issue tracker story.

The user is shown the list of stories that can be started.
When they choose one, they are assigned to that story and the story
is started in the issue tracker.

Unless -no_branch is specified, the user is asked to insert
the branch name to be used for the branch holding the story commits.
The branch of the given name is created on top of the trunk branch
and checked out.
	`,
	Action: run,
}

var flagNoBranch bool

func init() {
	Command.Flags.BoolVar(&flagNoBranch, "no_branch", flagNoBranch,
		"do not create a new story branch")
}

func run(cmd *gocli.Command, args []string) {
	if len(args) != 0 {
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
		return errs.NewError(task, err, nil)
	}
	if len(stories) == 0 {
		return errs.NewError(task, errors.New("no startable stories found"), nil)
	}

	// Filter out the stories that are not relevant,
	// i.e. not owned by the current user or assigned to someone else.
	task = "Fetch the current user record from the issue tracker"
	user, err := tracker.CurrentUser()
	if err != nil {
		return errs.NewError(task, err, nil)
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
	story, err := prompt.PromptStory(
		"\nYou can start working on one of the following stories:", stories)
	if err != nil {
		if err == prompt.ErrCanceled {
			panic(err)
		}
		return err
	}
	fmt.Println()

	// Create the story branch, optionally.
	if !flagNoBranch {
		var action common.Action
		action, err = createBranch()
		if err != nil {
			return err
		}
		// Roll back on error.
		defer func() {
			if err != nil {
				if err := action.Rollback(); err != nil {
					errs.Log(err)
				}
			}
		}()
	}

	// Add the current user to the list of story assignees.
	task = "Amend the list of story assignees"
	log.Run(task)
	originalAssignees := story.Assignees()
	if err := story.AddAssignee(user); err != nil {
		return errs.NewError(task, err, nil)
	}
	defer func(task string) {
		// On error, reset the list of story assignees.
		if err != nil {
			log.Rollback(task)
			if err := story.SetAssignees(originalAssignees); err != nil {
				errs.LogError("Reset the list of story assignees", err, nil)
			}
		}
	}(task)

	// Start the selected story. No need to roll back.
	task = "Start the selected story"
	log.Run(task)
	if err := story.Start(); err != nil {
		return errs.NewError(task, err, nil)
	}

	return nil
}

func createBranch() (common.Action, error) {
	// Get the current branch name.
	originalBranch, err := git.CurrentBranch()
	if err != nil {
		return nil, err
	}

	// Fetch the remote repository.
	task := "Fetch the remote repository"
	log.Run(task)

	gitConfig, err := git.LoadConfig()
	if err != nil {
		return nil, errs.NewError(task, err, nil)
	}

	var (
		remoteName  = gitConfig.RemoteName()
		trunkBranch = gitConfig.TrunkBranchName()
	)
	if err := git.UpdateRemotes(remoteName); err != nil {
		return nil, errs.NewError(task, err, nil)
	}

	// Make sure the trunk branch is up to date.
	task = fmt.Sprintf("Make sure branch '%v' is up to date", trunkBranch)
	log.Run(task)
	if err := git.EnsureBranchSynchronized(trunkBranch, remoteName); err != nil {
		return nil, errs.NewError(task, err, nil)
	}

	// Prompt the user for the branch name.
	task = "Prompt the user for the branch name"
	line, err := prompt.Prompt("\nPlease insert branch slug: ")
	if err != nil {
		return nil, errs.NewError(task, err, nil)
	}

	exitFunc := func() {
		fmt.Println("\nI will exit now. Just run me again later if you want.")
		os.Exit(0)
	}

	sluggedLine := slug.Slug(line)
	if sluggedLine == "" {
		exitFunc()
	}

	branchName := "story/" + sluggedLine
	ok, err := prompt.Confirm(
		fmt.Sprintf("\nThe branch that is going to be created will be called '%s'.\nIs that alright?", branchName))
	if err != nil {
		return nil, errs.NewError(task, err, nil)
	}
	if !ok {
		exitFunc()
	}
	fmt.Println()

	createTask := fmt.Sprintf(
		"Create branch '%v' on top of branch '%v'", branchName, trunkBranch)
	log.Run(createTask)
	if err := git.Branch(branchName, trunkBranch); err != nil {
		return nil, errs.NewError(createTask, err, nil)
	}

	deleteTask := fmt.Sprintf("Delete branch '%v'", branchName)
	deleteBranch := func() error {
		// Roll back and delete the newly created branch.
		log.Rollback(createTask)
		if err := git.Branch("-D", branchName); err != nil {
			return errs.NewError(deleteTask, err, nil)
		}
		return nil
	}

	checkoutTask := fmt.Sprintf("Checkout branch '%v'", branchName)
	log.Run(checkoutTask)
	if err := git.Checkout(branchName); err != nil {
		errs.Log(deleteBranch())
		return nil, errs.NewError(checkoutTask, err, nil)
	}

	return common.ActionFunc(func() error {
		// Checkout the original branch.
		log.Rollback(checkoutTask)
		if err := git.Checkout(originalBranch); err != nil {
			return errs.NewError(
				fmt.Sprintf("Checkout the original branch '%v'", originalBranch), err, nil)
		}
		// Delete the newly created branch.
		return deleteBranch()
	}), nil
}
