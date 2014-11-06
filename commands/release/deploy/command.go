package deployCmd

import (
	// Stdlib
	"bytes"
	"fmt"
	"os"

	// Internal
	"github.com/salsita/salsaflow/app"
	"github.com/salsita/salsaflow/config"
	"github.com/salsita/salsaflow/git"
	"github.com/salsita/salsaflow/log"

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
		task          string
		stderr        *bytes.Buffer
		currentBranch string
	)
	defer func() {
		// Print error details.
		if err != nil {
			log.FailWithDetails(task, stderr)
		}

		// Checkout the original branch.
		if currentBranch == "" {
			return
		}
		task = "Checkout the original branch"
		log.Run(task)
		out, ex := git.Checkout(currentBranch)
		if ex != nil {
			log.FailWithDetails(task, out)
			return
		}
	}()

	// Remember the current branch.
	task = "Remember the current branch"
	log.Run(task)
	currentBranch, stderr, err = git.CurrentBranch()
	if err != nil {
		return
	}

	// Make sure that the target ref exists.
	task = "Make sure that the target git reference exists"
	exists, stderr, err := git.RefExists(ref)
	if err != nil {
		return
	}
	if !exists {
		err = fmt.Errorf("git reference '%v' not found", ref)
		return
	}

	// Reset the master branch to point to the chosen ref.
	task = fmt.Sprintf("Reset the master branch to point to '%v'", ref)
	log.Run(task)
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
			log.Rollback(task)
			out, ex := git.ResetKeep(config.MasterBranch, origMaster)
			if ex != nil {
				log.FailWithDetails(taskMsg, out)
			}
		}
	}(task)

	// Push the master branch to trigger deployment.
	task = "Push the master branch to trigger deployment"
	log.Run(task)
	stderr, err = git.PushForce(config.OriginName, config.MasterBranch+":"+config.MasterBranch)
	return
}
