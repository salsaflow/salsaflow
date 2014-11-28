package bumpCmd

import (
	// Stdlib
	"os"

	// Internal
	"github.com/salsaflow/salsaflow/errs"
	"github.com/salsaflow/salsaflow/version"

	// Other
	"gopkg.in/tchap/gocli.v2"
)

var Command = &gocli.Command{
	UsageLine: "bump VERSION",
	Short:     "bump version to the specified value",
	Long: `
  Bump the version string to the specified value.

  This command only affects the working tree, it is not committing the changes.
	`,
	Action: run,
}

func run(cmd *gocli.Command, args []string) {
	if len(args) != 1 {
		cmd.Usage()
		os.Exit(2)
	}

	if err := runMain(args[0]); err != nil {
		errs.Fatal(err)
	}
}

func runMain(versionString string) error {
	// Make sure the version string is correct.
	ver, err := version.Parse(versionString)
	if err != nil {
		return err
	}

	// Set the version.
	return version.Set(ver)
}
