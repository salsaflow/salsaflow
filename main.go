package main

import (
	// Stdlib
	"fmt"
	"os"
	"os/signal"

	// Internal
	"github.com/salsaflow/salsaflow/app"
	"github.com/salsaflow/salsaflow/app/metadata"
	"github.com/salsaflow/salsaflow/commands/pkg"
	"github.com/salsaflow/salsaflow/commands/release"
	"github.com/salsaflow/salsaflow/commands/repo"
	"github.com/salsaflow/salsaflow/commands/review"
	"github.com/salsaflow/salsaflow/commands/story"
	"github.com/salsaflow/salsaflow/commands/version"

	// Other
	"gopkg.in/tchap/gocli.v2"
)

func main() {
	// Initialise the application.
	trunk := gocli.NewApp("salsaflow")
	trunk.UsageLine = "salsaflow SUBCMD [SUBCMD_OPTION ...]"
	trunk.Short = "the ultimate Trunk Based Development CLI utility"
	trunk.Version = metadata.Version
	trunk.Long = `
  salsaflow is a CLI utility that provides some useful shortcuts for
  Trunk Based Development. See the list of subcommands.`

	// Set up the top-level action function.
	var flagVersion bool
	trunk.Flags.BoolVar(&flagVersion, "version", flagVersion, "print SalsaFlow version and exit")
	trunk.Action = func(cmd *gocli.Command, args []string) {
		if len(args) != 0 || !flagVersion {
			cmd.Usage()
			os.Exit(2)
		}
		fmt.Println(metadata.Version)
	}

	// Register global flags.
	app.RegisterGlobalFlags(&trunk.Flags)

	// Register subcommands.
	trunk.MustRegisterSubcommand(pkgCmd.Command)
	trunk.MustRegisterSubcommand(releaseCmd.Command)
	trunk.MustRegisterSubcommand(repoCmd.Command)
	trunk.MustRegisterSubcommand(reviewCmd.Command)
	trunk.MustRegisterSubcommand(storyCmd.Command)
	trunk.MustRegisterSubcommand(versionCmd.Command)

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
