package app

import (
	// Stdlib
	"errors"

	// Internal
	"github.com/salsaflow/salsaflow/app/appflags"
	"github.com/salsaflow/salsaflow/errs"
	"github.com/salsaflow/salsaflow/log"
	"github.com/salsaflow/salsaflow/repo"
)

var ErrRepositoryNotInitialised = errors.New("repository not initialised")

func Init(force bool) error {
	InitLogging()

	// Make sure the repo is initialised.
	if err := repo.Init(force); err != nil {
		return err
	}

	return nil
}

func InitLogging() {
	// Set up logging.
	log.SetV(log.MustStringToLevel(appflags.FlagLog.Value()))
}

func InitOrDie() {
	if err := Init(false); err != nil {
		if errs.RootCause(err) != repo.ErrInitialised {
			errs.Fatal(err)
		}
	}
}
