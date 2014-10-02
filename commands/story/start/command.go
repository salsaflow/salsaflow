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

	log.Run("Fetch stories from the issue tracker")
	stories, err := modules.GetIssueTracker().StartableStories()
	if err != nil {
		handleError(err, nil)
		return err
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
		if storyId == selectedStory.Id() {
			re := regexp.MustCompile(".*/(story/.+/.+)$")
			branchName := re.ReplaceAllString(ref, "$1")
			matchingBranches[branchName] = true
		}
	}

	if len(matchingBranches) == 1 {
		// There is only one branch => checkout and work on that one.
		for branch := range matchingBranches {
			log.Log(fmt.Sprintf("Found one existing story branch: %s", branch))
			log.Run(fmt.Sprintf("Checkout %s", branch))
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
		branchName := fmt.Sprintf("story/%s/%s", slug.Slug(line), selectedStory.ReadableId())
		ok, err := prompt.Confirm(
			fmt.Sprintf("The branch will be called '%s', OK?", branchName))
		if err != nil {
			handleError(err, nil)
			return err
		}
		if !ok {
			log.Fatalln("I will exit now. If you want to try again, just run me again!")
		}
		fmt.Println()

		msg := "Create and checkout " + branchName
		log.Run(msg)
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

	log.Run(fmt.Sprintf("Start the selected story (%v)", selectedStory.ReadableId()))
	if err := selectedStory.Start(); err != nil {
		handleError(err, err.Stderr)
		return err
	}

	log.Run("Set you as the story owner")
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

	// Do not checkout the original branch, the story branch is active now.
	currentBranch = ""
	return nil
}
