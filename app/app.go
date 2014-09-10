package app

import (
	// Stdlib
	"flag"

	// Internal
	flags "github.com/tchap/git-trunk/flag"
	"github.com/tchap/git-trunk/log"
)

var (
	LogFlag *flags.StringEnumFlag = flags.NewStringEnumFlag(log.LevelStrings(), log.MustLevelToString(log.Info))
)

func RegisterGlobalFlags(flags *flag.FlagSet) {
	flags.Var(LogFlag, "log", "set logging verbosity; {trace|debug|verbose|info|off}")
}

func Init() {
	log.SetV(log.MustStringToLevel(LogFlag.Value()))
}
