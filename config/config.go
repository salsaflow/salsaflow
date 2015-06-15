package config

import (
	// Stdlib
	"bytes"
	"io/ioutil"
	"os"
	"os/user"
	"path/filepath"

	// Internal
	"github.com/salsaflow/salsaflow/errs"
	"github.com/salsaflow/salsaflow/git/gitutil"

	// Other
	"gopkg.in/yaml.v2"
)

const (
	// LocalConfigDirname is the directory relative to the repository root
	// that is being used to store all SalsaFlow-related files.
	LocalConfigDirname = ".salsaflow"

	// LocalConfigFilename is the filename of the configuration file
	// that represents local project-specific SalsaFlow configuration.
	//
	// This file is expected to be placed in the repository root.
	LocalConfigFilename = "config.yml"

	// GlobalConfigFilename is the filename of the configuration file
	// that represents global user-specific SalsaFlow configuration.
	//
	// This file is expected to be placed in the user's home directory.
	GlobalConfigFilename = ".salsaflow.yml"
)

// Local config ----------------------------------------------------------------

var localContentCache []byte

func UnmarshalLocalConfig(v interface{}) error {
	// Read the local config file into the cache in case it's not there yet.
	if localContentCache == nil {
		localContent, err := readLocalConfig()
		if err != nil {
			return err
		}
		localContentCache = localContent.Bytes()
	}

	// Unmarshall the local config file.
	task := "Unmarshal the local config file"
	if err := yaml.Unmarshal(localContentCache, v); err != nil {
		return errs.NewErrorWithHint(
			task, err, "Make sure the configuration file is valid YAML\n")
	}
	return nil
}

func LocalConfigDirectoryAbsolutePath() (string, error) {
	root, err := gitutil.RepositoryRootAbsolutePath()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, LocalConfigDirname), nil
}

func readLocalConfig() (content *bytes.Buffer, err error) {
	// Get the config file absolute path.
	path, err := LocalConfigDirectoryAbsolutePath()
	if err != nil {
		return nil, err
	}
	path = filepath.Join(path, LocalConfigFilename)

	// Read the content and return it.
	task := "Read the local config file"
	contentBytes, err := ioutil.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			hint := `
The local configuration file was not found.
Check 'repo bootstrap' to see how to generate it.

`
			return nil, errs.NewErrorWithHint(task, err, hint)
		}
		return nil, errs.NewError(task, err)
	}
	return bytes.NewBuffer(contentBytes), nil
}

// Global config ---------------------------------------------------------------

var globalContentCache []byte

func UnmarshalGlobalConfig(v interface{}) error {
	// Read the global config file into the cache in case it's not there yet.
	if globalContentCache == nil {
		globalContent, err := readGlobalConfig()
		if err != nil {
			return err
		}
		globalContentCache = globalContent.Bytes()
	}

	// Unmarshal the global config file.
	task := "Unmarshal the global configuration file"
	if err := yaml.Unmarshal(globalContentCache, v); err != nil {
		return errs.NewErrorWithHint(
			task, err, "Make sure the configuration file is valid YAML\n")
	}
	return nil
}

func GlobalConfigFileAbsolutePath() (string, error) {
	task := "Get the global configuration file absolute path"
	me, err := user.Current()
	if err != nil {
		return "", errs.NewError(task, err)
	}
	return filepath.Join(me.HomeDir, GlobalConfigFilename), nil
}

func readGlobalConfig() (content *bytes.Buffer, err error) {
	// Get the global config file path.
	path, err := GlobalConfigFileAbsolutePath()
	if err != nil {
		return nil, err
	}

	// Read the global config file.
	task := "Read the global config file"
	contentBytes, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, errs.NewError(task, err)
	}
	return bytes.NewBuffer(contentBytes), nil
}
