package versionCmd

import (
	// Stdlib
	"fmt"
	"os"

	// Internal
	"github.com/salsita/salsaflow/app/metadata"

	// Other
	"gopkg.in/tchap/gocli.v1"
)

var Command = &gocli.Command{
	UsageLine: "version",
	Short:     "print SalsaFlow version and exit",
	Long: `
  Print SalsaFlow version and exit. No more, no less.
	`,
	Action: func(cmd *gocli.Command, args []string) {
		if len(args) != 0 {
			cmd.Usage()
			os.Exit(2)
		}

		fmt.Println(metadata.Version)
	},
}
