package releaseCmd

import (
	"github.com/tchap/git-trunk/commands/release/close"
	"github.com/tchap/git-trunk/commands/release/create"

	"github.com/tchap/gocli"
)

var Command = &gocli.Command{
	UsageLine: "release",
	Short:     "various release-related actions",
	Long: `
  Perform various release-related actions. See the subcommands.
	`,
}

func init() {
	Command.MustRegisterSubcommand(createCmd.Command)
	Command.MustRegisterSubcommand(closeCmd.Command)
}
