package closeCmd

import (
	"github.com/tchap/gocli"
)

var Command *gocli.Command

func init() {
	Command = &gocli.Command{
		UsageLine: `
  close`,
		Short: "close the current release",
		Long: `
  Close the release that is currently running. This means that:

    1) Branch 'release' is tagged with its version string.
    2) Branch 'release' is deleted.
    3) Branch 'client' is moved to point to the newly created tag.
    4) Everything is pushed.
		`,
		Action: run,
	}
}

func run(cmd *gocli.Command, args []string) {

}
