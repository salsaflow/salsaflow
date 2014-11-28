package repoCmd

import (
	"github.com/salsaflow/salsaflow/commands/repo/bootstrap"
	"github.com/salsaflow/salsaflow/commands/repo/init"

	"gopkg.in/tchap/gocli.v2"
)

var Command = &gocli.Command{
	UsageLine: "repo",
	Short:     "various repository-related actions",
	Long: `
  Perform various repository-related actions. See the subcommands.
	`,
}

func init() {
	Command.MustRegisterSubcommand(bootstrapCmd.Command)
	Command.MustRegisterSubcommand(initCmd.Command)
}
