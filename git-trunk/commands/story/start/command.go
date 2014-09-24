package startCmd

import (
	// Stdlib
	"bytes"
	"fmt"
	"os"
	"regexp"

	// Internal
	"github.com/salsita/SalsaFlow/git-trunk/app"
	"github.com/salsita/SalsaFlow/git-trunk/config"
	"github.com/salsita/SalsaFlow/git-trunk/git"
	"github.com/salsita/SalsaFlow/git-trunk/log"
	"github.com/salsita/SalsaFlow/git-trunk/modules"
	"github.com/salsita/SalsaFlow/git-trunk/modules/common"
	"github.com/salsita/SalsaFlow/git-trunk/prompt"

	// Other
	"github.com/extemporalgenome/slug"
	"gopkg.in/tchap/gocli.v1"
)

var Command = &gocli.Command{
	UsageLine: `
  start`,
	Short: "start a new story",
	Long: `
  Starts a new story, i.e., create a new branch off ` + config.TrunkBranch + ` or check out an existing branch containing the story id if there's already one. Then updates the story state in PM tool and sets you as the story owner.
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

func handleError(err error, stderr *bytes.Buffer) {
	log.FailWithDetails(err.Error(), stderr)
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
			handleError(err, stderr)
			return
		}
	}()

	// Remember the current branch.
	log.Run("Remember the current branch")
	currentBranch, stderr, err := git.CurrentBranch()
	if err != nil {
		handleError(err, stderr)
		return err
	}

	log.Run("Loading stories from PM tool")
	stories, err := modules.GetIssueTracker().GetStartableStories()
	if err != nil {
		handleError(err, nil)
		return err
	}

	// List stories that can be started.
	for i, story := range stories {
		log.Printf("[%d]: %s -> %s\n", i, story.GetId(), story.GetTitle())
	}

	// Prompt user to choose.
	index, err := prompt.PromptIndex("Choose story: ", 0, len(stories)-1)
	if err != nil {
		log.Fatalln(err.Error())
	}

	selectedStory = stories[index]

	// Get all branches.
	localRefs, remoteRefs, stderr, err := git.ListStoryRefs()
	if err != nil {
		handleError(err, nil)
		return err
	}

	// Get all story branches connected to `selectedStory` id.
	log.Run("Check existing branches")
	matchingBranches := make(map[string]bool)

	for _, ref := range append(localRefs, remoteRefs...) {
		storyId, err := git.RefToStoryId(ref)
		if err != nil {
			handleError(err, nil)
			return err
		}
		if storyId == selectedStory.GetId() {
			re := regexp.MustCompile(".*/(story/.+/.+)$")
			branchName := re.ReplaceAllString(ref, "$1")
			matchingBranches[branchName] = true
		}
	}

	if len(matchingBranches) == 1 {
		// There is only one branch => checkout and work on that one.
		for branch := range matchingBranches {
			log.Log(fmt.Sprintf("Found one existing story branch: %s", branch))
			log.Run(fmt.Sprintf("Checking out %s", branch))
			if stderr, err = git.Checkout(branch); err != nil {
				handleError(err, stderr)
				return
			}
		}

	} else if len(matchingBranches) > 1 {
		// There are multiple branches. Let the user choose which one to work on.
		log.Log("Found multiple existing story branches:")
		for branch := range matchingBranches {
			log.Printf("\t - %s\n", branch)
		}
		log.Log("Please checkout one of them and work there.")

	} else {
		// There is no branch => create a new one.
		line, err := prompt.Prompt("Please insert branch slug: ")
		if err != nil {
			handleError(err, nil)
			return err
		}
		branchName := fmt.Sprintf("story/%s/%s", slug.Slug(line), selectedStory.GetId())
		ok, err := prompt.Confirm(
			fmt.Sprintf("The branch will be called '%s', OK?", branchName))
		if err != nil {
			handleError(err, nil)
			return err
		}
		if !ok {
			log.Fatalln("I will exit now. If you want to try again, just run me again!")
		}

		msg := "Checkout " + branchName
		if _, stderr, err = git.Git("branch", branchName, config.TrunkBranch); err != nil {
			handleError(err, stderr)
			return err
		}

		if _, stderr, err = git.Git("checkout", branchName); err != nil {
			handleError(err, stderr)
			return err
		}

		// Delete created branch on rollback.
		defer func(msg string) {
			if err != nil {
				log.Rollback(msg)
				if stderr, err := git.Branch("-D", branchName); err != nil {
					handleError(err, stderr)
				}
			}
		}(msg)
	}

	log.Run(fmt.Sprintf("Starting story %s", selectedStory.GetId()))
	if err := selectedStory.Start(); err != nil {
		handleError(err, err.Stderr)
		return err
	}

	log.Run("Setting you as the story owner")
	user, err := modules.GetIssueTracker().CurrentUser()
	if err != nil {
		handleError(err, nil)
		return err
	}
	// TODO: We should update, not overwrite the owners.
	if err := selectedStory.SetOwners([]common.User{user}); err != nil {
		handleError(err, nil)
		return err
	}

	return nil
}
