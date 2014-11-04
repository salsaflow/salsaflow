package repoCmd

import (
	"github.com/salsita/salsaflow/commands/repo/init"

	"gopkg.in/tchap/gocli.v1"
)

var Command = &gocli.Command{
	UsageLine: "repo",
	Short:     "various repository-related actions",
	Long: `
  Perform various repository-related actions. See the subcommands.
	`,
}

func init() {
	Command.MustRegisterSubcommand(initCmd.Command)
}
