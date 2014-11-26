package versionCmd

import (
	// Stdlib
	"fmt"
	"os"

	// Internal
	"github.com/salsaflow/salsaflow/commands/version/bump"
	"github.com/salsaflow/salsaflow/errs"
	"github.com/salsaflow/salsaflow/version"

	// Other
	"gopkg.in/tchap/gocli.v1"
)

var Command = &gocli.Command{
	UsageLine: "version",
	Short:     "various version-related actions",
	Long: `
  Print SalsaFlow version and exit. No more, no less.

  There are also some cool subcommands available. Check them out.
	`,
	Action: func(cmd *gocli.Command, args []string) {
		if len(args) != 0 {
			cmd.Usage()
			os.Exit(2)
		}

		ver, err := version.Get()
		if err != nil {
			errs.Fatal(err)
		}

		fmt.Println(ver)
	},
}

func init() {
	Command.MustRegisterSubcommand(bumpCmd.Command)
}
