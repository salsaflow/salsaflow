package app

import (
	// Stdlib
	"flag"

	// Internal
	"github.com/salsita/salsaflow/config"
	"github.com/salsita/salsaflow/errors"
	flags "github.com/salsita/salsaflow/flag"
	"github.com/salsita/salsaflow/log"
	"github.com/salsita/salsaflow/modules"
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
