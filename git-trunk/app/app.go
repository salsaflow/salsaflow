package app

import (
	// Stdlib
	"flag"

	// Internal
	"github.com/salsita/SalsaFlow/git-trunk/config"
	flags "github.com/salsita/SalsaFlow/git-trunk/flag"
	"github.com/salsita/SalsaFlow/git-trunk/log"
)

var (
	LogFlag *flags.StringEnumFlag = flags.NewStringEnumFlag(
		log.LevelStrings(), log.MustLevelToString(log.Info))
)

func RegisterGlobalFlags(flags *flag.FlagSet) {
	flags.Var(LogFlag, "log", "set logging verbosity; {trace|debug|verbose|info|off}")
}

func MustInit() {
	// Set up logging.
	log.SetV(log.MustStringToLevel(LogFlag.Value()))

	// Load the workflow configuration.
	config.MustLoad()
}
