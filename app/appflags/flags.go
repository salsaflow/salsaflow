package appflags

import (
	// Stdlib
	"flag"

	// Internal
	flags "github.com/salsaflow/salsaflow/flag"
	"github.com/salsaflow/salsaflow/log"
)

var (
	FlagConfig string
	FlagLog    *flags.StringEnumFlag = flags.NewStringEnumFlag(
		log.LevelStrings(), log.MustLevelToString(log.Info))
)

func RegisterGlobalFlags(flags *flag.FlagSet) {
	flags.StringVar(&FlagConfig, "config", FlagConfig, "set custom global configuration file")
	flags.Var(FlagLog, "log", "set logging verbosity; {trace|debug|verbose|info|off}")
}
