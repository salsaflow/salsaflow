package main

import (
	// Stdlib
	"fmt"
	"os"
	"os/signal"

	// Internal
	"github.com/salsita/salsaflow/app"
	"github.com/salsita/salsaflow/commands/init"
	"github.com/salsita/salsaflow/commands/release"
	"github.com/salsita/salsaflow/commands/repo"
	"github.com/salsita/salsaflow/commands/review"
	"github.com/salsita/salsaflow/commands/story"

	// Other
	"gopkg.in/tchap/gocli.v1"
)

const version = "0.2.1"

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
	trunk.MustRegisterSubcommand(repoCmd.Command)
	trunk.MustRegisterSubcommand(storyCmd.Command)
	trunk.MustRegisterSubcommand(initCmd.Command)
	trunk.MustRegisterSubcommand(reviewCmd.Command)

	// Start processing signals.
	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, os.Interrupt)
	go catchSignals(signalCh)

	// Run the application.
	trunk.Run(os.Args[1:])
}

func catchSignals(ch chan os.Signal) {
	<-ch
	fmt.Print(`
+-----------------------------------------------------+
| Signal received, the child processes were notified. |
| Send the signal again to exit immediately.          |
+-----------------------------------------------------+
	`)
	signal.Stop(ch)
}
