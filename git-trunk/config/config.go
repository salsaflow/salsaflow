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
	"github.com/salsita/SalsaFlow/git-trunk/log"

	// Other
	"gopkg.in/yaml.v1"
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

func MustLoad() {
	// Let's go async!
	resCh := make(chan *readResult, 2)

	// Read the config files from the disk.
	go func() {
		msg := "Read project configuration file"
		localConfig, stderr, err := readLocalConfig()
		if err != nil {
			resCh <- &readResult{msg, stderr, err}
			return
		}
		localConfigContent = localConfig.Bytes()
		resCh <- nil
	}()

	go func() {
		msg := "Read global configuration file"
		globalConfig, err := readGlobalConfig()
		if err != nil {
			resCh <- &readResult{msg, nil, err}
			return
		}
		globalConfigContent = globalConfig.Bytes()
		resCh <- nil
	}()

	// Wait for the files to be read.
	var failed bool
	for i := 0; i < cap(resCh); i++ {
		if res := <-resCh; res != nil && res.err != nil {
			errors.NewError(res.msg, res.stderr, res.err).Log(log.V(log.Info))
			failed = true
		}
	}
	if failed {
		log.Fatalln("\nError: failed to load configuration")
	}

	// Parse the local config to know what config modules to bootstrap.
	msg := "Parse project configuration file"
	var config struct {
		IssueTracker string `yaml:"issue_tracker"`
	}
	if err := yaml.Unmarshal(localConfigContent, &config); err != nil {
		die(msg, err)
	}
	if config.IssueTracker == "" {
		die(msg, &ErrKeyNotSet{"issue_tracker"})
	}

	// Set the issue tracker name.
	IssueTrackerName = config.IssueTracker
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
