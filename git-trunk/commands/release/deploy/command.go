package deployCmd

import (
	// Stdlib
	"bytes"
	"fmt"
	"os"

	// Internal
	"github.com/salsita/SalsaFlow/git-trunk/app"
	"github.com/salsita/SalsaFlow/git-trunk/config"
	"github.com/salsita/SalsaFlow/git-trunk/git"
	"github.com/salsita/SalsaFlow/git-trunk/log"

	// Other
	"gopkg.in/tchap/gocli.v1"
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

	app.MustInit()

	if err := runMain(args[0]); err != nil {
		log.Fatalln("\nError: " + err.Error())
	}
}

func runMain(ref string) (err error) {
	var (
		msg           string
		stderr        *bytes.Buffer
		currentBranch string
	)
	defer func() {
		// Print error details.
		if err != nil {
			log.FailWithDetails(msg, stderr)
		}

		// Checkout the original branch.
		if currentBranch == "" {
			return
		}
		msg = "Checkout the original branch"
		log.Run(msg)
		out, ex := git.Checkout(currentBranch)
		if ex != nil {
			log.FailWithDetails(msg, out)
			return
		}
	}()

	// Remember the current branch.
	msg = "Remember the current branch"
	log.Run(msg)
	currentBranch, stderr, err = git.CurrentBranch()
	if err != nil {
		return
	}

	// Make sure that the target ref exists.
	msg = "Make sure that the target git reference exists"
	exists, stderr, err := git.RefExists(ref)
	if err != nil {
		return
	}
	if !exists {
		err = fmt.Errorf("git reference '%v' not found", ref)
		return
	}

	// Reset the master branch to point to the chosen ref.
	msg = "Reset the master branch to point to " + ref
	log.Run(msg)
	origMaster, stderr, err := git.Hexsha("refs/heads/" + config.MasterBranch)
	if err != nil {
		return
	}
	stderr, err = git.ResetKeep(config.MasterBranch, ref)
	if err != nil {
		return err
	}
	defer func(taskMsg string) {
		// On error, reset the master branch to the origin position.
		if err != nil {
			log.Rollback(msg)
			out, ex := git.ResetKeep(config.MasterBranch, origMaster)
			if ex != nil {
				log.FailWithDetails(taskMsg, out)
			}
		}
	}(msg)

	// Push the master branch to trigger deployment.
	msg = "Push the master branch to trigger deployment"
	log.Run(msg)
	stderr, err = git.Push(
		config.OriginName, "-f", config.MasterBranch+":"+config.MasterBranch)
	return
}
