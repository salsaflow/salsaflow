package app

import (
	// Stdlib
	"flag"

	// Internal
	"github.com/salsita/SalsaFlow/git-trunk/config"
	"github.com/salsita/SalsaFlow/git-trunk/errors"
	flags "github.com/salsita/SalsaFlow/git-trunk/flag"
	"github.com/salsita/SalsaFlow/git-trunk/log"
	"github.com/salsita/SalsaFlow/git-trunk/modules"
)

var (
	LogFlag *flags.StringEnumFlag = flags.NewStringEnumFlag(
		log.LevelStrings(), log.MustLevelToString(log.Info))
)

func RegisterGlobalFlags(flags *flag.FlagSet) {
	flags.Var(LogFlag, "log", "set logging verbosity; {trace|debug|verbose|info|off}")
}

func Init() *errors.Error {
	// Set up logging.
	log.SetV(log.MustStringToLevel(LogFlag.Value()))

	// Load the workflow configuration.
	if err := config.Load(); err != nil {
		return err
	}

	// Bootstrap the modules.
	if err := modules.Bootstrap(); err != nil {
		return err
	}

	return nil
}

func MustInit() {
	var logger = log.V(log.Info)
	if err := Init(); err != nil {
		err.Fatal(logger)
	}
}
