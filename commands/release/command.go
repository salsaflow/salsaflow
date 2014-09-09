package releaseCmd

import (
	"github.com/tchap/git-trunk/commands/release/changes"
	"github.com/tchap/git-trunk/commands/release/deploy"
	"github.com/tchap/git-trunk/commands/release/stage"
	"github.com/tchap/git-trunk/commands/release/start"

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
	Command.MustRegisterSubcommand(changesCmd.Command)
	Command.MustRegisterSubcommand(deployCmd.Command)
	Command.MustRegisterSubcommand(startCmd.Command)
	Command.MustRegisterSubcommand(stageCmd.Command)
}
