package version

import (
	// Internal
	"github.com/salsaflow/salsaflow/config"

	// Vendor
	"github.com/blang/semver"
)

// Local configuration -------------------------------------------------------

const (
	DefaultTrunkSuffix   = "dev"
	DefaultTestingSuffix = "qa"
	DefaultStageSuffix   = "stage"
)

type LocalConfig struct {
	V struct {
		TrunkSuffix   string `yaml:"trunk_suffix"`
		TestingSuffix string `yaml:"testing_suffix"`
		StageSuffix   string `yaml:"stage_suffix"`
	} `yaml:"versioning"`
}

func newLocalConfig() *LocalConfig {
	var local LocalConfig
	local.V.TrunkSuffix = DefaultTrunkSuffix
	local.V.TestingSuffix = DefaultTestingSuffix
	local.V.StageSuffix = DefaultStageSuffix
	return &local
}

// Proxy struct ----------------------------------------------------------------

type Config interface {
	TrunkSuffix() semver.PRVersion
	TestingSuffix() semver.PRVersion
	StageSuffix() semver.PRVersion
}

var configCache Config

func LoadConfig() (Config, error) {
	// Try the cache first.
	if configCache != nil {
		return configCache, nil
	}

	// Load local config.
	local := newLocalConfig()
	if err := config.UnmarshalLocalConfig(local); err != nil {
		return nil, err
	}

	// Parse version suffixes.
	trunk, err := semver.NewPRVersion(local.V.TrunkSuffix)
	if err != nil {
		return nil, err
	}
	testing, err := semver.NewPRVersion(local.V.TestingSuffix)
	if err != nil {
		return nil, err
	}
	stage, err := semver.NewPRVersion(local.V.StageSuffix)
	if err != nil {
		return nil, err
	}

	// Save the new instance into the cache and return.
	configCache = &configProxy{trunk, testing, stage}
	return configCache, nil
}

type configProxy struct {
	trunk   semver.PRVersion
	testing semver.PRVersion
	stage   semver.PRVersion
}

func (proxy *configProxy) TrunkSuffix() semver.PRVersion {
	return proxy.trunk
}

func (proxy *configProxy) TestingSuffix() semver.PRVersion {
	return proxy.testing
}

func (proxy *configProxy) StageSuffix() semver.PRVersion {
	return proxy.stage
}
