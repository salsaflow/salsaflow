package storyCmd

import (
	"github.com/salsita/SalsaFlow/git-trunk/commands/story/changes"
	"github.com/salsita/SalsaFlow/git-trunk/commands/story/start"

	"gopkg.in/tchap/gocli.v1"
)

var Command = &gocli.Command{
	UsageLine: "story",
	Short:     "various story-related actions",
	Long: `
  Perform various story-related actions. See the subcommands.
	`,
}

func init() {
	Command.MustRegisterSubcommand(changesCmd.Command)
	Command.MustRegisterSubcommand(startCmd.Command)
}
