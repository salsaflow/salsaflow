package bumpCmd

import (
	// Stdlib
	"os"

	// Internal
	"github.com/salsaflow/salsaflow/app/appflags"
	"github.com/salsaflow/salsaflow/errs"
	"github.com/salsaflow/salsaflow/git/gitutil"
	"github.com/salsaflow/salsaflow/version"

	// Other
	"gopkg.in/tchap/gocli.v2"
)

var Command = &gocli.Command{
	UsageLine: "bump [-commit] VERSION",
	Short:     "bump version to the specified value",
	Long: `
  Bump the version string to the specified value.

  In case -commit is set, the changes are committed as well.
  The repository must be clean for the commit to be created.
	`,
	Action: run,
}

var (
	flagCommit bool
)

func init() {
	// Register flags.
	Command.Flags.BoolVar(&flagCommit, "commit", flagCommit,
		"commit the new version string")

	// Register global flags.
	appflags.RegisterGlobalFlags(&Command.Flags)
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
	task := "Parse the command line VERSION argument"
	ver, err := version.Parse(versionString)
	if err != nil {
		hint := `
The version string must be in the form of Major.Minor.Patch
and no part of the version string can be omitted.

`
		return errs.NewErrorWithHint(task, err, hint)
	}

	// In case -commit is set, set and commit the version string.
	if flagCommit {
		currentBranch, err := gitutil.CurrentBranch()
		if err != nil {
			return err
		}

		_, err = version.SetForBranch(ver, currentBranch)
		return err
	}

	// Otherwise just set the version.
	return version.Set(ver)
}
