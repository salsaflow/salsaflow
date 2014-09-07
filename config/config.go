package config

import (
	// Stdlib
	"bytes"
	"io"
	"os"
	"os/user"
	"path/filepath"

	// Internal
	"github.com/tchap/git-trunk/git"
	"github.com/tchap/git-trunk/log"

	// Other
	"gopkg.in/yaml.v1"
)

const (
	LocalConfigFileName  = "gitflow.yml"
	GlobalConfigFileName = ".gitflow.yml"

	ConfigBranch = "gitflow-config"
)

var (
	localConfigContent  []byte
	globalConfigContent []byte
)

func init() {
	localConfig, stderr, err := readLocalConfig()
	if err != nil {
		log.FailWithContext("Read local configuration file", stderr)
		log.Fatalln(err)
	}
	localConfigContent = localConfig.Bytes()

	globalConfig, err := readGlobalConfig()
	if err != nil {
		log.Fail("Read global configuration file")
		log.Fatalln(err)
	}
	globalConfigContent = globalConfig.Bytes()
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
