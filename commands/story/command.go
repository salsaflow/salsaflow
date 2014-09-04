package storyCmd

import (
	"github.com/tchap/git-trunk/commands/story/changes"

	"github.com/tchap/gocli"
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
}
