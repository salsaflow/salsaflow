package stageCmd

import (
	// Stdlib
	"os"

	// Internal
	"github.com/salsaflow/salsaflow/app"
	"github.com/salsaflow/salsaflow/errs"
	"github.com/salsaflow/salsaflow/releases/commands"

	// Other
	"gopkg.in/tchap/gocli.v2"
)

var Command = &gocli.Command{
	UsageLine: `
  stage`,
	Short: "stage current release",
	Long: `
  Stage the release that is currently running, i.e.

    1) Make sure the release can be staged (the stories are reviewed and tested)
    2) Reset the staging branch to point to the release branch.
    3) Delete the release branch.
    4) Bump the version for the staging branch.
    5) Push the changes.
	`,
	Action: run,
}

func run(cmd *gocli.Command, args []string) {
	if len(args) != 0 {
		cmd.Usage()
		os.Exit(2)
	}

	app.InitOrDie()

	if _, err := commands.Stage(nil); err != nil {
		errs.Fatal(err)
	}
}
