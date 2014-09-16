package config

import (
	// Stdlib
	"bytes"
	"io"
	"os"
	"os/user"
	"path/filepath"
	"sync"

	// Internal
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

var (
	localConfigContent  []byte
	globalConfigContent []byte
)

func MustLoad() {
	// Let's go async!
	var wg sync.WaitGroup
	wg.Add(2)

	// Read the config files from the disk.
	go func() {
		defer wg.Done()
		msg := "Read project configuration file"
		localConfig, stderr, err := readLocalConfig()
		if err != nil {
			log.FailWithContext(msg, stderr)
			log.Fatalln("\nError:", err)
		}
		localConfigContent = localConfig.Bytes()
	}()

	go func() {
		defer wg.Done()
		msg := "Read global configuration file"
		globalConfig, err := readGlobalConfig()
		if err != nil {
			die(msg, err)
		}
		globalConfigContent = globalConfig.Bytes()
	}()

	// Wait for the files to be read.
	wg.Wait()

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

	// Initialize the modules.
	switch config.IssueTracker {
	case sectionPivotalTracker:
		mustInitPivotalTracker()
	case sectionJira:
		mustInitJira()
	default:
		die(msg, &ErrKeyInvalid{"issue_tracker", config.IssueTracker})
	}
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

func fillLocalConfig(v interface{}) error {
	return yaml.Unmarshal(localConfigContent, v)
}

func fillGlobalConfig(v interface{}) error {
	return yaml.Unmarshal(globalConfigContent, v)
}
