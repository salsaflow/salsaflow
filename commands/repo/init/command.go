package initCmd

import (
	// Stdlib
	"os"

	// Internal
	"github.com/salsita/salsaflow/app"
	"github.com/salsita/salsaflow/log"

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
		log.Fatalln("\nError: " + err.Error())
	}

	log.Log("The repository is initialised")
}
