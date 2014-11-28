package gocli

import (
	"fmt"
)

// Examples -------------------------------------------------------------------

func ExampleApp() {
	// Create the root App object.
	app := NewApp("app")
	app.Short = "my bloody gocli app"
	app.Version = "1.2.3"
	app.Long = `
  This is a long description of my super uber cool app.`

	// A verbose switch flag.
	var verbose bool

	// Create a subcommand.
	var subcmd = &Command{
		UsageLine: "subcmd [-v]",
		Short:     "some kind of subcommand, you name it",
		Long:      "Brb, too tired to write long descriptions.",
		Action: func(cmd *Command, args []string) {
			fmt.Printf("verbose mode set to %t\n", verbose)
		},
	}

	// Set up the verbose switch. This can be as well called in init() or so.
	subcmd.Flags.BoolVar(&verbose, "v", false, "print verbose output")

	// Register the command with the parent command. Also suitable for init().
	app.MustRegisterSubcommand(subcmd)

	/*
		Run the whole thing.

		app.Run([]string{}) would lead into:

		APPLICATION:
		  app - my bloody gocli app

		OPTIONS:
		  -h=false: print help and exit

		VERSION:
		  1.2.3

		DESCRIPTION:
		  This is a long description of my super uber cool app.

		SUBCOMMANDS:
		  subcmd	 some kind of subcommand, you name it

		, app.Run([]string{"subcmd", "-h"}) into something similar.
	*/

	app.Run([]string{"subcmd"})
	app.Run([]string{"subcmd", "-v"})
	// Output:
	// verbose mode set to false
	// verbose mode set to true
}
