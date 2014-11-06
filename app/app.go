package app

import (
	// Stdlib
	"errors"
	"flag"

	// Internal
	"github.com/salsita/salsaflow/errs"
	flags "github.com/salsita/salsaflow/flag"
	"github.com/salsita/salsaflow/log"
	"github.com/salsita/salsaflow/repo"
)

var ErrRepositoryNotInitialised = errors.New("repository not initialised")

var (
	LogFlag *flags.StringEnumFlag = flags.NewStringEnumFlag(
		log.LevelStrings(), log.MustLevelToString(log.Info))
)

func RegisterGlobalFlags(flags *flag.FlagSet) {
	flags.Var(LogFlag, "log", "set logging verbosity; {trace|debug|verbose|info|off}")
}

func Init() error {
	// Set up logging.
	log.SetV(log.MustStringToLevel(LogFlag.Value()))

	// Make sure the repo is initialised.
	if err := repo.Init(); err != nil {
		return err
	}

	return nil
}

func InitOrDie() {
	if err := Init(); err != nil {
		if ex, ok := err.(*errs.Error); !ok || ex.RootCause() != repo.ErrInitialised {
			errs.Fatal(err)
		}
	}
}
