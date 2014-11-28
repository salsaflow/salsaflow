package reviewCmd

import (
	"github.com/salsaflow/salsaflow/commands/review/post"

	"gopkg.in/tchap/gocli.v2"
)

var Command = &gocli.Command{
	UsageLine: "review",
	Short:     "various review-related actions",
	Long: `
  Perform various review related actions. See the subcommands.
	`,
}

func init() {
	Command.MustRegisterSubcommand(postCmd.Command)
}
