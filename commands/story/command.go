package storyCmd

import (
	"github.com/salsaflow/salsaflow/app/appflags"
	"github.com/salsaflow/salsaflow/commands/story/changes"
	"github.com/salsaflow/salsaflow/commands/story/open"
	"github.com/salsaflow/salsaflow/commands/story/start"

	"gopkg.in/tchap/gocli.v2"
)

var Command = &gocli.Command{
	UsageLine: "story",
	Short:     "various story-related actions",
	Long: `
  Perform various story-related actions. See the subcommands.
	`,
}

func init() {
	// Register global flags.
	appflags.RegisterGlobalFlags(&Command.Flags)

	// Register subcommands.
	Command.MustRegisterSubcommand(changesCmd.Command)
	Command.MustRegisterSubcommand(openCmd.Command)
	Command.MustRegisterSubcommand(startCmd.Command)
}
