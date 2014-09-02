package release

import (
	"github.com/tchap/git-trunk/commands/release/close"
	"github.com/tchap/git-trunk/commands/release/create"

	"github.com/tchap/gocli"
)

var Command *gocli.Command

func init() {
	Command = &gocli.Command{
		UsageLine: "release",
		Short:     "various release actions",
		Long: `
  Perform various release-related actions. See the subcommands.
		`,
	}

	Command.MustRegisterSubcommand(createCmd.Command)
	Command.MustRegisterSubcommand(closeCmd.Command)
}
