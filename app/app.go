package app

import (
	// Stdlib
	"errors"
	"flag"

	// Internal
	"github.com/salsaflow/salsaflow/errs"
	flags "github.com/salsaflow/salsaflow/flag"
	"github.com/salsaflow/salsaflow/log"
	"github.com/salsaflow/salsaflow/repo"
)

var ErrRepositoryNotInitialised = errors.New("repository not initialised")

var (
	LogFlag *flags.StringEnumFlag = flags.NewStringEnumFlag(
		log.LevelStrings(), log.MustLevelToString(log.Info))
)

func RegisterGlobalFlags(flags *flag.FlagSet) {
	flags.Var(LogFlag, "log", "set logging verbosity; {trace|debug|verbose|info|off}")
}

func Init(force bool) error {
	// Set up logging.
	log.SetV(log.MustStringToLevel(LogFlag.Value()))

	// Make sure the repo is initialised.
	if err := repo.Init(force); err != nil {
		return err
	}

	return nil
}

func InitOrDie() {
	if err := Init(false); err != nil {
		if ex, ok := err.(*errs.Error); !ok || ex.RootCause() != repo.ErrInitialised {
			errs.Fatal(err)
		}
	}
}
