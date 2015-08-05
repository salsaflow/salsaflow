package repoCmd

import (
	"github.com/salsaflow/salsaflow/app/appflags"
	"github.com/salsaflow/salsaflow/commands/repo/bootstrap"
	"github.com/salsaflow/salsaflow/commands/repo/init"
	"github.com/salsaflow/salsaflow/commands/repo/prune"

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
	// Register global flags.
	appflags.RegisterGlobalFlags(&Command.Flags)

	// Register subcommands.
	Command.MustRegisterSubcommand(bootstrapCmd.Command)
	Command.MustRegisterSubcommand(initCmd.Command)
	Command.MustRegisterSubcommand(pruneCmd.Command)
}
