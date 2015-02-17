package main

import (
	// Stdlib
	"time"

	// Internal
	"github.com/salsaflow/salsaflow/config"
	"github.com/salsaflow/salsaflow/errs"
)

type LocalConfig struct {
	EnabledTimestamp time.Time `yaml:"salsaflow_enabled_timestamp"`
}

func SalsaFlowEnabledTimestamp() (time.Time, error) {
	task := "Load hook-related SalsaFlow config"

	var lc LocalConfig
	if err := config.UnmarshalLocalConfig(&lc); err != nil {
		return time.Time{}, errs.NewError(task, err, nil)
	}

	return lc.EnabledTimestamp, nil
}
