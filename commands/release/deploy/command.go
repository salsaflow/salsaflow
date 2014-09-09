package deployCmd

import (
	// Stdlib
	"bytes"
	"os"

	// Internal
	"github.com/tchap/git-trunk/config"
	"github.com/tchap/git-trunk/git"
	"github.com/tchap/git-trunk/log"

	// Other
	"github.com/tchap/gocli"
)

var Command = &gocli.Command{
	UsageLine: "deploy REF",
	Short:     "deploy a ref into production",
	Long: `
  Deploy the chosen Git ref into production.

  This basically means that the master branch is reset
  to point to REF, then force pushed.
	`,
	Action: run,
}

func run(cmd *gocli.Command, args []string) {
	if len(args) != 1 {
		cmd.Usage()
		os.Exit(2)
	}

	if err := runMain(args[0]); err != nil {
		log.Fatalln("\nError: " + err.Error())
	}
}

func runMain(ref string) (err error) {
	var (
		taskMsg       string
		stderr        *bytes.Buffer
		currentBranch string
	)
	defer func() {
		// Print error details.
		if err != nil {
			log.FailWithContext(taskMsg, stderr)
		}

		// Checkout the original branch.
		if currentBranch == "" {
			return
		}
		taskMsg = "Checkout the original branch"
		log.Run(taskMsg)
		out, ex := git.Checkout(currentBranch)
		if ex != nil {
			log.FailWithContext(taskMsg, out)
			return
		}
	}()

	// Remember the current branch.
	taskMsg = "Remember the current branch"
	log.Run(taskMsg)
	currentBranch, stderr, err = git.CurrentBranch()
	if err != nil {
		return
	}

	// Reset the master branch to point to the chosen ref.
	taskMsg = "Reset the master branch to point to " + ref
	log.Run(taskMsg)
	origMaster, stderr, err := git.Hexsha("refs/heads/" + config.MasterBranch)
	if err != nil {
		return
	}
	stderr, err = git.ResetKeep(config.MasterBranch, ref)
	if err != nil {
		return err
	}
	defer func(msg string) {
		// On error, reset the master branch to the origin position.
		if err != nil {
			log.Rollback(msg)
			out, ex := git.ResetKeep(config.MasterBranch, origMaster)
			if ex != nil {
				log.FailWithContext(msg, out)
			}
		}
	}(taskMsg)

	// Push the master branch to trigger deployment.
	taskMsg = "Push the master branch to trigger deployment"
	log.Run(taskMsg)
	stderr, err = git.Push(
		config.OriginName, "-f", config.MasterBranch+":"+config.MasterBranch)
	return
}
