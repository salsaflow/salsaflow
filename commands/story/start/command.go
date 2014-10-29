package startCmd

import (
	// Stdlib
	"bytes"
	"errors"
	"fmt"
	"os"

	// Internal
	"github.com/salsita/salsaflow/app"
	"github.com/salsita/salsaflow/config"
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

	app.MustInit()

	if err := runMain(); err != nil {
		log.Fatalln("\nFatal error: " + err.Error())
	}
}

func handleError(task string, err error, stderr *bytes.Buffer) error {
	errs.NewError(task, stderr, err).Log(log.V(log.Info))
	return err
}

func runMain() (err error) {
	tracker := modules.GetIssueTracker()

	// Fetch stories from the issue tracker.
	msg := "Fetch stories from the issue tracker"
	log.Run(msg)
	stories, err := tracker.StartableStories()
	if err != nil {
		return errs.LogFail(msg, err)
	}
	if len(stories) == 0 {
		return errs.LogFail(msg, errors.New("no startable stories found"))
	}

	// Filter out the stories that are not relevant,
	// i.e. not owned by the current user or assigned to someone else.
	msg = "Fetch the current user record from the issue tracker"
	user, err := tracker.CurrentUser()
	if err != nil {
		return errs.LogFail(msg, err)
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
		return errs.Log(err)
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
	msg = "Amend the list of story assignees"
	log.Run(msg)
	originalAssignees := story.Assignees()
	if err := story.AddAssignee(user); err != nil {
		return errs.LogFail(msg, err)
	}
	defer func(msg string) {
		// On error, reset the list of story assignees.
		if err != nil {
			log.Rollback(msg)
			if err := story.SetAssignees(originalAssignees); err != nil {
				errs.Log(errs.NewError("Reset the list of story assignees", nil, err))
			}
		}
	}(msg)

	// Start the selected story. No need to roll back.
	msg = "Start the selected story"
	log.Run(msg)
	if err := story.Start(); err != nil {
		return errs.Log(err)
	}

	return nil
}

func createBranch() (common.Action, error) {
	// Get the current branch name.
	msg := "Get the current branch name"
	originalBranch, stderr, err := git.CurrentBranch()
	if err != nil {
		return nil, handleError(msg, err, stderr)
	}

	// Fetch the remote repository.
	msg = "Fetch the remote repository"
	log.Run(msg)
	stderr, err = git.UpdateRemotes(config.OriginName)
	if err != nil {
		return nil, handleError(msg, err, stderr)
	}

	// Make sure the trunk branch is up to date.
	msg = fmt.Sprintf("Make sure branch '%v' is up to date", config.TrunkBranch)
	log.Run(msg)
	stderr, err = git.EnsureBranchSynchronized(config.TrunkBranch, config.OriginName)
	if err != nil {
		return nil, handleError(msg, err, stderr)
	}

	// Prompt the user for the branch name.
	msg = "Prompt the user for the branch name"
	line, err := prompt.Prompt("\nPlease insert branch slug: ")
	if err != nil {
		return nil, errs.LogFail(msg, err)
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
		return nil, errs.LogFail(msg, err)
	}
	if !ok {
		exitFunc()
	}
	fmt.Println()

	createMsg := fmt.Sprintf(
		"Create branch '%v' on top of branch '%v'", branchName, config.TrunkBranch)
	log.Run(createMsg)
	if stderr, err = git.Branch(branchName, config.TrunkBranch); err != nil {
		return nil, handleError(msg, err, stderr)
	}

	deleteMsg := fmt.Sprintf("Delete branch '%v'", branchName)
	deleteBranch := func() error {
		// Roll back and delete the newly created branch.
		log.Rollback(createMsg)
		if stderr, err := git.Branch("-D", branchName); err != nil {
			return errs.NewError(deleteMsg, stderr, err)
		}
		return nil
	}

	checkoutMsg := fmt.Sprintf("Checkout branch '%v'", branchName)
	log.Run(checkoutMsg)
	if stderr, err = git.Checkout(branchName); err != nil {
		errs.Log(deleteBranch())
		return nil, handleError(msg, err, stderr)
	}

	return common.ActionFunc(func() error {
		// Checkout the original branch.
		log.Rollback(checkoutMsg)
		if stderr, err := git.Checkout(originalBranch); err != nil {
			return errs.NewError(
				fmt.Sprintf("Checkout the original branch '%v'", originalBranch), stderr, err)
		}
		// Delete the newly created branch.
		return deleteBranch()
	}), nil
}
