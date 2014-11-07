package scripts

import (
	"github.com/salsita/salsaflow/config"
	"github.com/salsita/salsaflow/errs"
)

// Local configuration -------------------------------------------------------

type LocalConfig struct {
	Scripts struct {
		GetVersion string `yaml:"get_version"`
		SetVersion string `yaml:"set_version"`
	} `yaml:"scripts"`
}

// Configuration proxy ---------------------------------------------------------

type Config interface {
	GetVersionScriptRelativePath() string
	SetVersionScriptRelativePath() string
}

var configCache Config

func LoadConfig() (Config, error) {
	// Try the cache first.
	if configCache != nil {
		return configCache, nil
	}

	task := "Load scripts-related SalsaFlow config"

	// Parse the local config file.
	proxy := &configProxy{
		local: &LocalConfig{},
	}
	if err := config.UnmarshalLocalConfig(proxy.local); err != nil {
		return nil, errs.NewError(task, err, nil)
	}

	// Save the new instance into the cache and return.
	configCache = proxy
	return configCache, nil
}

type configProxy struct {
	local *LocalConfig
}

func (proxy *configProxy) GetVersionScriptRelativePath() string {
	return proxy.local.Scripts.GetVersion
}

func (proxy *configProxy) SetVersionScriptRelativePath() string {
	return proxy.local.Scripts.SetVersion
}
