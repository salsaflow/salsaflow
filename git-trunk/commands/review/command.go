package reviewCmd

import (
	"github.com/salsita/SalsaFlow/git-trunk/commands/review/post"

	"gopkg.in/tchap/gocli.v1"
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
