package startCmd

import (
	// Stdlib
	"bytes"
	"fmt"
	"io"
	"os"
	"regexp"
	"text/tabwriter"

	// Internal
	"github.com/salsita/salsaflow/app"
	"github.com/salsita/salsaflow/config"
	"github.com/salsita/salsaflow/errors"
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
	UsageLine: `
  start`,
	Short: "start a new story",
	Long: `
  Starts a new story, i.e., create a new branch off ` + config.TrunkBranch + `
  or check out an existing branch containing the story id if there's already one.
  Then updates the story state in PM tool and sets you as the story owner.
	`,
	Action: run,
}

func run(cmd *gocli.Command, args []string) {
	if len(args) != 0 {
		cmd.Usage()
		os.Exit(2)
	}

	app.MustInit()

	if err := runMain(); err != nil {
		log.Fatalln("\nError: " + err.Error())
	}
}

func handleError(task string, err error, stderr *bytes.Buffer) error {
	errors.NewError(task, stderr, err).Log(log.V(log.Info))
	return err
}

func runMain() (err error) {
	var (
		selectedStory common.Story
		currentBranch string
	)

	defer func() {
		// Checkout the original branch.
		if currentBranch == "" {
			return
		}
		msg := fmt.Sprintf("Checkout the original branch (%s)", currentBranch)
		log.Run(msg)
		stderr, err := git.Checkout(currentBranch)
		if err != nil {
			handleError(msg, err, stderr)
			return
		}
	}()

	// Remember the current branch.
	msg := "Remember the current branch"
	log.Run(msg)
	currentBranch, stderr, err := git.CurrentBranch()
	if err != nil {
		return handleError(msg, err, stderr)
	}

	// Fetch stories from the issue tracker.
	msg = "Fetch stories from the issue tracker"
	log.Run(msg)
	stories, err := modules.GetIssueTracker().StartableStories()
	if err != nil {
		return handleError(msg, err, nil)
	}

	// List stories that can be started.
	tw := tabwriter.NewWriter(os.Stdout, 0, 8, 4, '\t', 0)
	io.WriteString(tw, "\nYou can start working on one of the following stories:\n\n")
	io.WriteString(tw, "Index\tStory ID\tStory Title\n")
	io.WriteString(tw, "=====\t========\t===========\n")
	for i, story := range stories {
		fmt.Fprintf(tw, "%v\t%v\t%v\n", i, story.ReadableId(), story.Title())
	}
	io.WriteString(tw, "\n")
	tw.Flush()

	// Prompt user to choose.
	index, err := prompt.PromptIndex("Choose a story by inserting its index: ", 0, len(stories)-1)
	if err != nil {
		log.Fatalln(err.Error())
	}
	fmt.Println()

	selectedStory = stories[index]

	// Fetch the remote repository.
	msg = "Fetch the remote repository"
	log.Run(msg)
	stderr, err = git.UpdateRemotes(config.OriginName)
	if err != nil {
		return handleError(msg, err, nil)
	}

	// Get all branches.
	msg = "Collect all story branches"
	localRefs, remoteRefs, stderr, err := git.ListStoryRefs()
	if err != nil {
		return handleError(msg, err, nil)
	}

	// Get all story branches connected to `selectedStory` id.
	msg = "Check existing branches"
	log.Run(msg)
	matchingBranches := map[string]struct{}{}

	stripRemotePrefixRe := regexp.MustCompile(".*/(story/.+/.+)$")
	for _, ref := range append(localRefs, remoteRefs...) {
		storyId, err := git.RefToStoryId(ref)
		if err != nil {
			return handleError(msg, err, nil)
		}
		if storyId == selectedStory.ReadableId() {
			// We found a matching story branch. Let's strip off the `remote/origin` etc
			// part so that we have just the branch name.
			branchName := stripRemotePrefixRe.ReplaceAllString(ref, "$1")
			matchingBranches[branchName] = struct{}{}
		}
	}

	if len(matchingBranches) == 1 {
		// There is only one branch => checkout and work on that one.
		for branch := range matchingBranches {
			log.Log("Found one existing story branch: " + branch)
			msg := "Checkout branch " + branch
			log.Run(msg)
			if stderr, err = git.Checkout(branch); err != nil {
				return handleError(msg, err, stderr)
			}
		}

	} else if len(matchingBranches) > 1 {
		// There are multiple branches. Let the user choose which one to work on.
		logger := log.V(log.Info)
		logger.Lock()
		logger.UnsafeLog("Found multiple existing story branches:")
		for branch := range matchingBranches {
			logger.UnsafeNewLine("  - " + branch)
		}
		logger.UnsafeNewLine("Please checkout one of them and work there.")
		logger.Unlock()

	} else {
		// There is no branch => create a new one.
		msg := "Prompt for the new story branch name"
		line, err := prompt.Prompt("Please insert branch slug: ")
		if err != nil {
			return handleError(msg, err, nil)
		}
		branchName := fmt.Sprintf("story/%s/%s", slug.Slug(line), selectedStory.ReadableId())
		ok, err := prompt.Confirm(
			fmt.Sprintf("The branch will be called '%s', OK?", branchName))
		if err != nil {
			return handleError(msg, err, nil)
		}
		if !ok {
			log.Fatalln("I will exit now. If you want to try again, just run me again!")
		}
		fmt.Println()

		createMsg := "Create branch " + branchName
		log.Run(msg)
		if _, stderr, err = git.Git("branch", branchName, config.TrunkBranch); err != nil {
			return handleError(msg, err, stderr)
		}

		msg = "Checkout branch " + branchName
		if _, stderr, err = git.Git("checkout", branchName); err != nil {
			return handleError(msg, err, stderr)
		}

		// Delete the newly created branch on rollback.
		defer func(msg string) {
			if err != nil {
				log.Rollback(msg)
				if stderr, err := git.Branch("-D", branchName); err != nil {
					handleError("Delete branch "+branchName, err, stderr)
				}
			}
		}(createMsg)
	}

	msg = fmt.Sprintf("Start the selected story (%v)", selectedStory.ReadableId())
	log.Run(msg)
	if err := selectedStory.Start(); err != nil {
		return handleError(msg, err, err.Stderr)
	}

	msg = "Set you as the story owner"
	log.Run(msg)
	user, err := modules.GetIssueTracker().CurrentUser()
	if err != nil {
		return handleError(msg, err, nil)
	}
	// TODO: We should update, not overwrite the owners.
	if err := selectedStory.SetOwners([]common.User{user}); err != nil {
		return handleError(msg, err, nil)
	}

	// Do not checkout the original branch, the story branch is active now.
	currentBranch = ""
	return nil
}
