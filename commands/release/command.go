package releaseCmd

import (
	// "github.com/salsita/salsaflow/commands/release/changes"
	// "github.com/salsita/salsaflow/commands/release/deploy"
	"github.com/salsita/salsaflow/commands/release/stage"
	"github.com/salsita/salsaflow/commands/release/start"

	"gopkg.in/tchap/gocli.v1"
)

var Command = &gocli.Command{
	UsageLine: "release",
	Short:     "various release-related actions",
	Long: `
  Perform various release-related actions. See the subcommands.
	`,
}

func init() {
	// Command.MustRegisterSubcommand(changesCmd.Command)
	// Command.MustRegisterSubcommand(deployCmd.Command)
	Command.MustRegisterSubcommand(startCmd.Command)
	Command.MustRegisterSubcommand(stageCmd.Command)
}
