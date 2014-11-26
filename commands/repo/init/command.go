package initCmd

import (
	// Stdlib
	"os"

	// Internal
	"github.com/salsaflow/salsaflow/app"
	"github.com/salsaflow/salsaflow/errs"
	"github.com/salsaflow/salsaflow/log"
	"github.com/salsaflow/salsaflow/repo"

	// Other
	"gopkg.in/tchap/gocli.v1"
)

var Command = &gocli.Command{
	UsageLine: "init",
	Short:     "initialize the repository",
	Long: `
  Initialize the repository so that it works with SalsaFlow.
	`,
	Action: run,
}

func run(cmd *gocli.Command, args []string) {
	if len(args) != 0 {
		cmd.Usage()
		os.Exit(2)
	}

	if err := app.Init(); err != nil {
		if ex, ok := err.(*errs.Error); ok && ex.RootCause() == repo.ErrInitialised {
			log.Log("The repository has been already initialised")
			return
		}
		errs.Fatal(err)
	}
}
