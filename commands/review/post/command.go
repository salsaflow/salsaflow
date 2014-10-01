package postCmd

import (
	"bytes"
	"fmt"
	"os"

	"github.com/salsita/salsaflow/app"
	"github.com/salsita/salsaflow/asciiart"
	"github.com/salsita/salsaflow/config"
	"github.com/salsita/salsaflow/git"
	"github.com/salsita/salsaflow/log"
	"github.com/salsita/salsaflow/prompt"
	"github.com/salsita/salsaflow/shell"

	"gopkg.in/tchap/gocli.v1"
)

var Command = &gocli.Command{
	UsageLine: "post",
	Short:     "post reviews for commits in a feature branch",
	Long: `
  Posts a review (using rbt tool) for each commit on the feature.
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

type readResult struct {
	sha    string
	stdout *bytes.Buffer
	stderr *bytes.Buffer
	err    error
}

func runMain() (err error) {
	var (
		currentBranch string
		storyId       string
		stderr        *bytes.Buffer
	)

	currentBranch, stderr, err = git.CurrentBranch()
	if err != nil {
		log.FailWithDetails("get current branch", stderr)
		return err
	}

	storyId, err = git.RefToStoryId(currentBranch)
	if err != nil {
		_, ok := err.(*git.ErrNotStoryBranch)
		if !ok {
			return err
		}
		log.Fail("No story branch detected.")
		log.Println("I'm sorry Dave, I can't let you do that. You have to checkout a story branch.")
		log.Fatalln("Have a good day!")
	}

	commits, stderr, err := git.ListBranchCommits(currentBranch, config.TrunkBranch)
	if err != nil {
		log.FailWithDetails("Listing commits on the story branch", stderr)
		return err
	}

	originTrunkBranch := fmt.Sprintf("%s/%s", config.OriginName, config.TrunkBranch)

	log.Printf(`
You are posting reviews for story %s. Swell!

Here's what's going to happen:
	1) We'll rebase your branch '%s' on top of branch '%s'.
	2) We'll post a review for each commit on your branch (that should be %d commits).`,
		storyId, currentBranch, originTrunkBranch, len(commits))

	asciiart.PrintSnoopy()

	log.Run(fmt.Sprintf("Fetching %s", config.OriginName))
	if stderr, err := git.UpdateRemotes(config.OriginName); err != nil {
		log.FailWithDetails(fmt.Sprintf("Could not fetch data from %s", config.OriginName), stderr)
		return err
	}

	log.Run(fmt.Sprintf("Rebasing on top of %s", originTrunkBranch))
	_, stderr, err = git.Git(
		"rebase", fmt.Sprintf("%s", originTrunkBranch))
	if err != nil {
		log.FailWithDetails("Error when rebasing", stderr)
		return err
	}
	log.Printf("Sweet, branch '%s' is now totally rebased on top of %s!\n\n",
		currentBranch, originTrunkBranch)

	log.Println("I will now post reviews for the following commits:")
	for _, commit := range commits {
		log.Printf("  %s (%s)\n", commit.SHA, commit.Title)
	}
	log.Println()

	confirmed, err := prompt.Confirm("You cool with that?")
	if err != nil {
		return err
	}
	if !confirmed {
		log.Println("I will exit now. If you want to try again, just run me again.")
		log.Println("Have a nice day!")
		return nil
	}

	resCh := make(chan *readResult, len(commits))

	for _, commit := range commits {
		go func(commit *git.Commit) {
			log.Go(fmt.Sprintf("Posting review for commit %s", commit.SHA))
			out, stderr, err := shell.Run(
				"rbt", "post", "--guess-fields", "yes", "--branch", storyId, commit.SHA)
			resCh <- &readResult{commit.SHA, out, stderr, err}
		}(commit)
	}
	for i := 0; i < cap(resCh); i++ {
		if res := <-resCh; res != nil {
			logRbtOutput(res)
		}
	}

	log.Println()
	log.Println(`
Please take some time go through the reviews, check and annotate them for the reviewer.

If you find any issues you want to fix right before publishing, fix them now, amend
the relevant commits and use:

rbt post -r <RB request id> --parent ` + config.TrunkBranch + ` <commit SHA>

to update the review.

  ###########################################################
  # IMPORTANT: Your code has not been merged and/or pushed. #
  ###########################################################

When you think the reviews are ready to be published, publish them in Review Board.
Then merge your branch into ` + config.TrunkBranch + ` and push.
`)

	return nil
}

func logRbtOutput(res *readResult) {
	var logger = log.V(log.Info)
	logger.Lock()
	defer logger.Unlock()
	if res.err != nil {
		logger.UnsafeFail(
			fmt.Sprintf("Could not post review request for commit %s", res.sha))
		logger.UnsafeStderr(res.stderr)
	} else {
		logger.UnsafeOk(fmt.Sprintf("Review for commit %s posted", res.sha))
		logger.UnsafePrint(res.stdout)
	}
}
