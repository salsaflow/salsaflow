package main

import (
	// Stdlib
	"time"

	// Internal
	"github.com/salsaflow/salsaflow/config"
	"github.com/salsaflow/salsaflow/errs"
)

type LocalConfig struct {
	EnabledTimestamp string `yaml:"salsaflow_enabled_timestamp"`
}

func SalsaFlowEnabledTimestamp() (time.Time, error) {
	task := "Load hook-related SalsaFlow config"

	var lc LocalConfig
	if err := config.UnmarshalLocalConfig(&lc); err != nil {
		return time.Time{}, errs.NewError(task, err, nil)
	}

	// Who knows why Time.String() uses a custom format...
	tc, err := time.Parse("2006-01-02 15:04:05.999999999 -0700 MST", lc.EnabledTimestamp)
	if err != nil {
		return time.Time{}, errs.NewError(task, err, nil)
	}
	return tc, nil
}
