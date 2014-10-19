package postCmd

import (
	// Stdlib
	"bytes"
	"fmt"
	"io"
	"os"
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
	"github.com/salsita/salsaflow/prompt"

	// Other
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

func handleError(task string, err error, stderr *bytes.Buffer) error {
	errs.NewError(task, stderr, err).Log(log.V(log.Info))
	return err
}

func runMain() (err error) {
	// Get the current branch name.
	msg := "Get the current branch name"
	currentBranch, stderr, err := git.CurrentBranch()
	if err != nil {
		return handleError(msg, err, nil)
	}

	// Parse the branch name. Return in case we are not on a story branch.
	msg = "Parse the branch name"
	storyId, err := git.RefToStoryId(currentBranch)
	if err != nil {
		_, ok := err.(*git.ErrNotStoryBranch)
		if !ok {
			return handleError(msg, err, nil)
		}
		return handleError(msg, err, nil)
	}

	// Fetch the remote repository.
	msg = "Fetch the remote repository"
	log.Run(msg)
	if stderr, err := git.UpdateRemotes(config.OriginName); err != nil {
		return handleError(msg, err, stderr)
	}

	// Make sure the trunk branch is up to date.
	msg = "Make sure the trunk branch is up to date"
	log.Run(msg)
	stderr, err = git.EnsureBranchSynchronized(config.TrunkBranch, config.OriginName)
	if err != nil {
		return handleError(msg, err, stderr)
	}

	// Get all the commits that are new compared to the trunk branch.
	msg = "List and parse the commits on the story branch"
	commits, stderr, err := git.ShowCommitRange(
		fmt.Sprintf("%s..%s", config.TrunkBranch, currentBranch))
	if err != nil {
		return handleError(msg, err, stderr)
	}

	// Tell the user what is going to happen and ask for confirmation.
	msg = "Ask the user to confirm the actions to follow"
	fmt.Printf(`
You are posting reviews for story %s. Swell!

Here's what's going to happen:
  1) Branch '%s' will be rebased onto branch '%s'.
  2) A review request will be posted for each of the following commits:`,
		storyId, currentBranch, config.TrunkBranch)

	tw := tabwriter.NewWriter(os.Stdout, 0, 8, 4, '\t', 0)
	io.WriteString(tw, "\n\n")
	io.WriteString(tw, "     Commit SHA\tCommit Title\n")
	io.WriteString(tw, "     ==========\t============\n")
	for _, commit := range commits {
		fmt.Fprintf(tw, "     %v\t%v\n", commit.SHA, commit.MessageTitle)
	}
	io.WriteString(tw, "\n")
	tw.Flush()

	confirmed, err := prompt.Confirm("You cool with that?")
	if err != nil {
		return handleError(msg, err, nil)
	}
	if !confirmed {
		fmt.Println("\nFair enough, have a nice day!")
		return nil
	}

	asciiart.PrintSnoopy()

	// Rebase the story branch on top of the trunk branch.
	msg = fmt.Sprintf("Rebase the story branch on top of branch '%v'", config.TrunkBranch)
	log.Run(msg)
	_, stderr, err = git.Git("rebase", config.TrunkBranch)
	if err != nil {
		return handleError(msg, err, stderr)
	}

	// Post the review requests.
	msg = "Post the review requests"
	tool := modules.GetCodeReviewTool()

	var wg sync.WaitGroup
	wg.Add(len(commits))
	for _, commit := range commits {
		go func(commit *git.Commit) {
			defer wg.Done()
			msg := "Post the review request for commit " + commit.SHA
			log.Go(msg)
			if err := tool.PostReviewRequest(commit); err != nil {
				errs.LogFail(msg, err)
			}
		}(commit)
	}
	wg.Wait()

	// Tell the user what to do next.
	fmt.Println("\n----------")
	tool.PrintPostReviewRequestFollowup()
	fmt.Print(`  ###########################################################
  # IMPORTANT: Your code has not been merged and/or pushed. #
  ###########################################################
	
`)

	return nil
}
