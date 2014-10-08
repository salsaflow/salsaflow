package config

import (
	// Stdlib
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/user"
	"path/filepath"
	"strings"

	// Internal
	"github.com/salsita/salsaflow/errs"
	"github.com/salsita/salsaflow/git"

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

func Load() *errs.Error {
	// Make sure the local configuration file is committed.
	msg := "Make sure the local configuration file is committed"
	if stderr, err := ensureLocalConfigCommitted(); err != nil {
		return errs.NewError(msg, stderr, err)
	}

	// Read the local configuration file.
	msg = "Read local configuration file"
	localConfig, stderr, err := ReadLocalConfig()
	if err != nil {
		return errs.NewError(msg, stderr, err)
	}
	localConfigContent = localConfig.Bytes()

	// Read the global configuration file.
	msg = "Read global configuration file"
	globalConfig, err := ReadGlobalConfig()
	if err != nil {
		return errs.NewError(msg, nil, err)
	}
	globalConfigContent = globalConfig.Bytes()

	// Parse the local config to know what config modules to bootstrap.
	msg = "Parse project configuration file"
	var config struct {
		IssueTracker string `yaml:"issue_tracker"`
	}
	if err := yaml.Unmarshal(localConfigContent, &config); err != nil {
		return errs.NewError(msg, nil, err)
	}
	if config.IssueTracker == "" {
		return errs.NewError(msg, nil, &ErrKeyNotSet{"issue_tracker"})
	}

	// Set the issue tracker name.
	IssueTrackerName = config.IssueTracker

	return nil
}

func ensureLocalConfigCommitted() (stderr *bytes.Buffer, err error) {
	stdout, stderr, err := git.Git("status", "--porcelain")
	if err != nil {
		return stderr, err
	}
	var (
		suffix  = " " + LocalConfigFileName
		scanner = bufio.NewScanner(stdout)
	)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasSuffix(line, suffix) {
			hint := fmt.Sprintf(`
Please commit your %v changes into branch '%v'.
Only then will I let you pass and proceed further!

`, LocalConfigFileName, TrunkBranch)
			return bytes.NewBufferString(hint), errors.New("local configuration file modified")
		}
	}
	return nil, scanner.Err()
}

func ReadLocalConfig() (content, stderr *bytes.Buffer, err error) {
	// Return the file content as committed on the config branch.
	return git.ShowByBranch(ConfigBranch, LocalConfigFileName)
}

func ReadGlobalConfig() (content *bytes.Buffer, err error) {
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
