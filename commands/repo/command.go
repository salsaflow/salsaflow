package repoCmd

import (
	"github.com/tchap/git-trunk/commands/repo/prune"

	"github.com/tchap/gocli"
)

var Command = &gocli.Command{
	UsageLine: "repo",
	Short:     "various repository-related actions",
	Long: `
  Perform various repository-related actions. See the subcommands.
	`,
}

func init() {
	Command.MustRegisterSubcommand(pruneCmd.Command)
}
