package config

import (
	// Stdlib
	"bytes"
	"io/ioutil"
	"os/user"
	"path/filepath"

	// Internal
	"github.com/salsaflow/salsaflow/errs"
	"github.com/salsaflow/salsaflow/git/gitutil"

	// Other
	"gopkg.in/yaml.v2"
)

const (
	// LocalConfigFilename is the filename of the configuration file
	// that represents local project-specific SalsaFlow configuration.
	//
	// This file is expected to be placed in the repository root.
	LocalConfigFilename = "salsaflow.yml"

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
		return errs.NewError(
			task, err, bytes.NewBufferString("Make sure the configuration file is valid YAML\n"))
	}
	return nil
}

func readLocalConfig() (content *bytes.Buffer, err error) {
	// Get the config file absolute path.
	root, err := gitutil.RepositoryRootAbsolutePath()
	if err != nil {
		return nil, err
	}
	path := filepath.Join(root, LocalConfigFilename)

	// Read the content and return it.
	task := "Read the local config file"
	contentBytes, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, errs.NewError(task, err, nil)
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
		return errs.NewError(
			task, err, bytes.NewBufferString("Make sure the configuration file is valid YAML\n"))
	}
	return nil
}

func readGlobalConfig() (content *bytes.Buffer, err error) {
	// Get the global config file path.
	task := "Get the current user's home directory"
	me, err := user.Current()
	if err != nil {
		return nil, errs.NewError(task, err, nil)
	}
	path := filepath.Join(me.HomeDir, GlobalConfigFilename)

	// Read the global config file.
	task = "Read the global config file"
	contentBytes, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, errs.NewError(task, err, nil)
	}
	return bytes.NewBuffer(contentBytes), nil
}
