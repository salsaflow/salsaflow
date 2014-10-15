package initCmd

import (
	// Stdlib
	"os"

	// Internal
	"github.com/salsita/salsaflow/app"
	"github.com/salsita/salsaflow/errs"
	"github.com/salsita/salsaflow/log"
	"github.com/salsita/salsaflow/repo"

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
		if err.Err == repo.ErrInitialised {
			log.Log("The repository has been already initialised")
			return
		}
		errs.Log(err)
		log.Fatalln("\nError: " + err.Error())
	}
}
