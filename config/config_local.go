package config

import (
	// Stdlib
	"path/filepath"
	"time"

	// Internal
	"github.com/salsaflow/salsaflow/errs"
	"github.com/salsaflow/salsaflow/git/gitutil"
)

const (
	// LocalConfigDirname is the directory relative to the repository root
	// that is being used to store all SalsaFlow-related files.
	LocalConfigDirname = ".salsaflow"

	// LocalConfigFilename is the filename of the configuration file
	// that represents local project-specific SalsaFlow configuration.
	//
	// This file is expected to be placed in the repository root.
	LocalConfigFilename = "config.json"
)

// LocalConfig represents the local configuration file content.
type LocalConfig struct {
	EnabledTimestamp *time.Time `json:"salsaflow_enabled_timestamp"`

	Modules struct {
		IssueTracking string `json:"issue_tracking"`
		CodeReview    string `json:"code_review"`
		ReleaseNotes  string `json:"release_notes"`
	} `json:"active_modules,omitempty"`

	*ConfigurationsSection
}

func NewEmptyLocalConfig() *LocalConfig {
	now := time.Now()
	return &LocalConfig{
		EnabledTimestamp:      &now,
		ConfigurationsSection: newConfigurationsSection(),
	}
}

func (local *LocalConfig) SaveChanges() error {
	return WriteLocalConfig(local)
}

// ReadLocalConfig reads and parses the global configuration file
// and returns a struct representing the content.
func ReadLocalConfig() (*LocalConfig, error) {
	task := "Read and parse the local configuration file"

	// Get the global configuration file absolute path.
	path, err := LocalConfigFileAbsolutePath()
	if err != nil {
		return nil, errs.NewError(task, err)
	}

	// Read and parse the file.
	var local LocalConfig
	if err := readAndUnmarshalConfig(path, &local); err != nil {
		return nil, errs.NewError(task, err)
	}

	// Return the config struct.
	return &local, nil
}

// WriteLocalConfig writes the given configuration struct
// into the global configuration file.
//
// In case the target path does not exist, it is created,
// including the parent directories.
//
// In case the file exists, it is truncated.
func WriteLocalConfig(config *LocalConfig) error {
	task := "Write the local configuration file"

	// Get the local configuration file absolute path.
	path, err := LocalConfigFileAbsolutePath()
	if err != nil {
		return errs.NewError(task, err)
	}

	// Write the file.
	if err := writeConfig(path, config, 0600); err != nil {
		return errs.NewError(task, err)
	}

	return nil
}

// LocalConfigFileAbsolutePath returns the absolute path
// of the local configuration directory.
func LocalConfigDirectoryAbsolutePath() (string, error) {
	root, err := gitutil.RepositoryRootAbsolutePath()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, LocalConfigDirname), nil
}

// LocalConfigFileAbsolutePath returns the absolute path
// of the local configuration file.
func LocalConfigFileAbsolutePath() (string, error) {
	configDir, err := LocalConfigDirectoryAbsolutePath()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, LocalConfigFilename), nil
}
