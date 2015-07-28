package initCmd

import (
	// Stdlib
	"os"

	// Internal
	"github.com/salsaflow/salsaflow/app"
	"github.com/salsaflow/salsaflow/app/appflags"
	"github.com/salsaflow/salsaflow/errs"
	"github.com/salsaflow/salsaflow/log"
	"github.com/salsaflow/salsaflow/repo"

	// Other
	"gopkg.in/tchap/gocli.v2"
)

var Command = &gocli.Command{
	UsageLine: "init",
	Short:     "initialize the repository",
	Long: `
  Initialize the repository so that it works with SalsaFlow.
	`,
	Action: run,
}

var flagForce bool

func init() {
	// Register flags.
	Command.Flags.BoolVar(&flagForce, "force", flagForce, "force repo init process")

	// Register global flags.
	appflags.RegisterGlobalFlags(&Command.Flags)
}

func run(cmd *gocli.Command, args []string) {
	if len(args) != 0 {
		cmd.Usage()
		os.Exit(2)
	}

	if err := app.Init(flagForce); err != nil {
		if errs.RootCause(err) == repo.ErrInitialised {
			log.Log("The repository has been already initialised")
			return
		}
		errs.Fatal(err)
	}
}
