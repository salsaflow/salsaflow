package releaseCmd

import (
	"github.com/salsaflow/salsaflow/app/appflags"
	"github.com/salsaflow/salsaflow/commands/release/changes"
	"github.com/salsaflow/salsaflow/commands/release/deploy"
	"github.com/salsaflow/salsaflow/commands/release/notes"
	"github.com/salsaflow/salsaflow/commands/release/stage"
	"github.com/salsaflow/salsaflow/commands/release/start"

	"gopkg.in/tchap/gocli.v2"
)

var Command = &gocli.Command{
	UsageLine: "release",
	Short:     "various release-related actions",
	Long: `
  Perform various release-related actions. See the subcommands.
	`,
}

func init() {
	// Register global flags.
	appflags.RegisterGlobalFlags(&Command.Flags)

	// Register subcommands.
	Command.MustRegisterSubcommand(changesCmd.Command)
	Command.MustRegisterSubcommand(deployCmd.Command)
	Command.MustRegisterSubcommand(notesCmd.Command)
	Command.MustRegisterSubcommand(startCmd.Command)
	Command.MustRegisterSubcommand(stageCmd.Command)
}
