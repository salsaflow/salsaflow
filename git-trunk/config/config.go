package config

import (
	// Stdlib
	"bytes"
	"io"
	"os"
	"os/user"
	"path/filepath"

	// Internal
	"github.com/salsita/SalsaFlow/git-trunk/errors"
	"github.com/salsita/SalsaFlow/git-trunk/git"

	// Other
	"gopkg.in/yaml.v2"
)

const (
	LocalConfigFileName  = "salsaflow.yml"
	GlobalConfigFileName = ".salsaflow.yml"

	ConfigBranch = TrunkBranch
)

var IssueTrackerName string

var (
	localConfigContent  []byte
	globalConfigContent []byte
)

type readResult struct {
	msg    string
	stderr *bytes.Buffer
	err    error
}

func Load() *errors.Error {

	localConfig, stderr, err := readLocalConfig()
	if err != nil {
		return errors.NewError("Read local configuration file", stderr, err)
	}
	localConfigContent = localConfig.Bytes()

	globalConfig, err := readGlobalConfig()
	if err != nil {
		return errors.NewError("Read global configuration file", stderr, err)
	}
	globalConfigContent = globalConfig.Bytes()

	// Parse the local config to know what config modules to bootstrap.
	msg := "Parse project configuration file"
	var config struct {
		IssueTracker string `yaml:"issue_tracker"`
	}
	if err := yaml.Unmarshal(localConfigContent, &config); err != nil {
		return errors.NewError(msg, nil, err)
	}
	if config.IssueTracker == "" {
		return errors.NewError(msg, nil, &ErrKeyNotSet{"issue_tracker"})
	}

	// Set the issue tracker name.
	IssueTrackerName = config.IssueTracker

	return nil
}

func readLocalConfig() (content, stderr *bytes.Buffer, err error) {
	// Return the file content as committed on the config branch.
	return git.ShowByBranch(ConfigBranch, LocalConfigFileName)
}

func readGlobalConfig() (content *bytes.Buffer, err error) {
	// Generate the global config file path.
	me, err := user.Current()
	if err != nil {
		return nil, err
	}
	path := filepath.Join(me.HomeDir, GlobalConfigFileName)

	// Read the global config file.
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var p bytes.Buffer
	if _, err := io.Copy(&p, file); err != nil {
		return nil, err
	}

	// Return the content.
	return &p, nil
}

func FillLocalConfig(v interface{}) error {
	return yaml.Unmarshal(localConfigContent, v)
}

func FillGlobalConfig(v interface{}) error {
	return yaml.Unmarshal(globalConfigContent, v)
}
