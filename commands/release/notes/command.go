package notesCmd

import (
	// Stdlib
	"fmt"
	"os"
	"strings"

	// Internal
	"github.com/salsaflow/salsaflow/app"
	"github.com/salsaflow/salsaflow/errs"
	"github.com/salsaflow/salsaflow/flag"
	"github.com/salsaflow/salsaflow/modules"
	"github.com/salsaflow/salsaflow/releases/notes"
	"github.com/salsaflow/salsaflow/version"

	// Vendor
	"gopkg.in/tchap/gocli.v2"
)

var Command = &gocli.Command{
	UsageLine: "notes [-format=FORMAT] [-pretty] VERSION",
	Short:     "print release notes",
	Long: fmt.Sprintf(`
  Print release notes for release VERSION.

  Supported formats: %v
	`, strings.Join(notes.AvailableEncodings(), ", ")),
	Action: run,
}

var (
	flagFormat = flag.NewStringEnumFlag(notes.AvailableEncodings(), string(notes.EncodingJson))
	flagPretty bool
)

func init() {
	Command.Flags.Var(flagFormat, "format", "output format")
	Command.Flags.BoolVar(&flagPretty, "pretty", flagPretty, "pretty-print the output")
}

func run(cmd *gocli.Command, args []string) {
	if len(args) != 1 {
		cmd.Usage()
		os.Exit(2)
	}

	app.InitOrDie()

	if err := runMain(args[0]); err != nil {
		errs.Fatal(err)
	}
}

func runMain(versionString string) (err error) {
	// Get issue tracker.
	tracker, err := modules.GetIssueTracker()
	if err != nil {
		return err
	}

	// Parse the version string.
	v, err := version.Parse(versionString)
	if err != nil {
		return err
	}

	// Generate the release notes.
	nts, err := tracker.ReleaseNotes(v)
	if err != nil {
		return err
	}

	// Dump the release notes.
	encoder, err := notes.NewEncoder(notes.Encoding(flagFormat.Value()), os.Stdout)
	if err != nil {
		return err
	}

	return encoder.Encode(nts, &notes.EncodeOptions{
		Pretty: flagPretty,
	})
}
