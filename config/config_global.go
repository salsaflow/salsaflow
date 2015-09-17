package config

import (
	// Stdlib
	"os/user"
	"path/filepath"

	// Internal
	"github.com/salsaflow/salsaflow/app/appflags"
	"github.com/salsaflow/salsaflow/errs"
)

const (
	// GlobalConfigFilename is the filename of the configuration file
	// that represents global user-specific SalsaFlow configuration.
	//
	// This file is expected to be placed in the user's home directory,
	// although the location can be configured by a command line flag.
	GlobalConfigFilename = ".salsaflow.json"
)

type GlobalConfig struct {
	*ConfigurationsSection
}

func NewEmptyGlobalConfig() *GlobalConfig {
	return &GlobalConfig{newConfigurationsSection()}
}

func (global *GlobalConfig) SaveChanges() error {
	return WriteGlobalConfig(global)
}

// ReadGlobalConfig reads and parses the global configuration file
// and returns a struct representing the content.
func ReadGlobalConfig() (*GlobalConfig, error) {
	task := "Read and parse the global configuration file"

	// Get the global configuration file absolute path.
	path, err := GlobalConfigFileAbsolutePath()
	if err != nil {
		return nil, errs.NewError(task, err)
	}

	// Read and parse the file.
	var global GlobalConfig
	if err := readAndUnmarshalConfig(path, &global); err != nil {
		return nil, errs.NewError(task, err)
	}

	// Return the config struct.
	return &global, nil
}

// WriteGlobalConfig writes the given configuration struct
// into the global configuration file.
//
// In case the target path does not exist, it is created,
// including the parent directories.
//
// In case the file exists, it is truncated.
func WriteGlobalConfig(config *GlobalConfig) error {
	task := "Write the global configuration file"

	// Get the global configuration file absolute path.
	path, err := GlobalConfigFileAbsolutePath()
	if err != nil {
		return errs.NewError(task, err)
	}

	// Write the file.
	if err := writeConfig(path, config, 0644); err != nil {
		return errs.NewError(task, err)
	}

	return nil
}

func GlobalConfigFileAbsolutePath() (string, error) {
	task := "Get the global configuration file absolute path"

	// Check the command line flag for custom path first.
	if path := appflags.FlagConfig; path != "" {
		return path, nil
	}

	// Otherwise use the default file in the user's home directory.
	me, err := user.Current()
	if err != nil {
		return "", errs.NewError(task, err)
	}
	return filepath.Join(me.HomeDir, GlobalConfigFilename), nil
}
