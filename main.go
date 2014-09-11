package main

import (
	// Stdlib
	"os"

	// Internal
	"github.com/tchap/git-trunk/app"
	"github.com/tchap/git-trunk/commands/release"
	"github.com/tchap/git-trunk/commands/story"

	// Other
	"github.com/tchap/gocli"
)

const version = "0.1.4"

func main() {
	// Initialise the application.
	trunk := gocli.NewApp("git-trunk")
	trunk.UsageLine = "git-trunk SUBCMD [SUBCMD_OPTION ...]"
	trunk.Short = "the ultimate Trunk Based Development CLI utility"
	trunk.Version = version
	trunk.Long = `
  git-trunk is a git plugin that provides some useful shortcuts for
  Trunk Based Development. See the list of subcommands.`

	// Register global flags.
	app.RegisterGlobalFlags(&trunk.Flags)

	// Register subcommands.
	trunk.MustRegisterSubcommand(releaseCmd.Command)
	trunk.MustRegisterSubcommand(storyCmd.Command)

	// Run the application.
	trunk.Run(os.Args[1:])
}
