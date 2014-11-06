package hooks

import (
	// Stdlib
	"flag"
	"fmt"
	"os"

	// Internal
	"github.com/salsita/salsaflow/app/metadata"
)

const versionFlag = "salsaflow.version"

func IdentifyYourself() {
	// Add a special command line flag.
	flagIdentify := flag.Bool(versionFlag, false,
		"print the associated SalsaFlow version and exit")

	// Parse the command line.
	flag.Parse()

	// In case the special flag is set, print the desired output and exit.
	if *flagIdentify {
		fmt.Println(metadata.Version)
		os.Exit(0)
	}
}
