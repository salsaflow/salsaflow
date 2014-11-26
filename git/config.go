package git

import (
	"github.com/salsaflow/salsaflow/config"
	"github.com/salsaflow/salsaflow/errs"
)

// Local configuration -------------------------------------------------------

const (
	DefaultRemoteName = "origin"

	DefaultTrunkBranchName   = "develop"
	DefaultReleaseBranchName = "release"
	DefaultStageBranchName   = "client"
	DefaultStableBranchName  = "master"
)

const ConfigKeyRemote = "salsaflow.remote"

type LocalConfig struct {
	Git struct {
		Branches struct {
			Trunk   string `yaml:"trunk"`
			Release string `yaml:"release"`
			Stage   string `yaml:"stage"`
			Stable  string `yaml:"stable"`
		} `yaml:"branches"`
	} `yaml:"git"`
}

func (local *LocalConfig) fillDefaults() {
	bs := &local.Git.Branches
	if bs.Trunk == "" {
		bs.Trunk = DefaultTrunkBranchName
	}
	if bs.Release == "" {
		bs.Release = DefaultReleaseBranchName
	}
	if bs.Stage == "" {
		bs.Stage = DefaultStageBranchName
	}
	if bs.Stable == "" {
		bs.Stable = DefaultStableBranchName
	}
}

// Configuration proxy ---------------------------------------------------------

type Config interface {
	RemoteName() string
	TrunkBranchName() string
	ReleaseBranchName() string
	StagingBranchName() string
	StableBranchName() string
}

var configCache Config

func LoadConfig() (Config, error) {
	// Try the cache first.
	if configCache != nil {
		return configCache, nil
	}

	task := "Load git-related SalsaFlow config"

	// Parse the local config file.
	proxy := &configProxy{
		local: &LocalConfig{},
	}
	if err := config.UnmarshalLocalConfig(proxy.local); err != nil {
		return nil, errs.NewError(task, err, nil)
	}
	proxy.local.fillDefaults()

	// Consult git config for the project remote name.
	remote, err := GetConfigString(ConfigKeyRemote)
	if err != nil {
		return nil, errs.NewError(task, err, nil)
	}
	if remote == "" {
		remote = DefaultRemoteName
	}
	proxy.remote = remote

	// Save the new instance into the cache and return.
	configCache = proxy
	return configCache, nil
}

type configProxy struct {
	remote string
	local  *LocalConfig
}

func (proxy *configProxy) RemoteName() string {
	return proxy.remote
}

func (proxy *configProxy) TrunkBranchName() string {
	return proxy.local.Git.Branches.Trunk
}

func (proxy *configProxy) ReleaseBranchName() string {
	return proxy.local.Git.Branches.Release
}

func (proxy *configProxy) StagingBranchName() string {
	return proxy.local.Git.Branches.Stage
}

func (proxy *configProxy) StableBranchName() string {
	return proxy.local.Git.Branches.Stable
}
