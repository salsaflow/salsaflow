package postCmd

import (
	// Stdlib
	"bytes"
	"fmt"
	"io"
	"os"
	"text/tabwriter"

	// Internal
	"github.com/salsita/salsaflow/app"
	"github.com/salsita/salsaflow/asciiart"
	"github.com/salsita/salsaflow/config"
	"github.com/salsita/salsaflow/errs"
	"github.com/salsita/salsaflow/git"
	"github.com/salsita/salsaflow/log"
	"github.com/salsita/salsaflow/prompt"
	"github.com/salsita/salsaflow/shell"

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

type readResult struct {
	msg    string
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

	// Remember the current branch.
	msg := "Remember the current branch"
	log.Run(msg)
	currentBranch, stderr, err = git.CurrentBranch()
	if err != nil {
		return handleError(msg, err, nil)
	}

	// Parse the branch name. Return in case we are not on a story branch.
	msg = "Parse the branch name"
	storyId, err = git.RefToStoryId(currentBranch)
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
	commits, stderr, err := git.ListBranchCommits(currentBranch, config.TrunkBranch)
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
		fmt.Fprintf(tw, "     %v\t%v\n", commit.SHA, commit.Title)
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
	resCh := make(chan *readResult, len(commits))

	for _, commit := range commits {
		go func(commit *git.Commit) {
			msg := "Post the review request for commit " + commit.SHA
			log.Go(msg)
			stdout, stderr, err := shell.Run(
				"rbt", "post", "--guess-fields", "yes", "--branch", storyId, commit.SHA)
			resCh <- &readResult{msg, stdout, stderr, err}
		}(commit)
	}
	for i := 0; i < cap(resCh); i++ {
		if res := <-resCh; res != nil {
			logRbtOutput(res)
		}
	}

	// Tell the user what to do next.
	log.Println(`
----------

Now, please, take some time to go through all the review requests,
check and annotate them for the reviewers to make them more happy (less sad).

If you find any issues you want to fix right before publishing, fix them now,
amend the relevant commits and use:

  $ rbt post -r <RB request id> <commit SHA>

to update the relevant review request.

  ###########################################################
  # IMPORTANT: Your code has not been merged and/or pushed. #
  ###########################################################

When you think the review requests are ready to be published,
publish them in Review Board. Then merge your branch into ` + config.TrunkBranch + ` and push.
`)

	return nil
}

func logRbtOutput(res *readResult) {
	var logger = log.V(log.Info)
	if res.err != nil {
		errs.NewError(res.msg, res.stderr, res.err).Log(logger)
	} else {
		logger.Lock()
		logger.UnsafeNewLine("")
		logger.UnsafeOk(res.msg)
		logger.UnsafePrint(res.stdout)
		logger.Unlock()
	}
}
