package github

import (
	"github.com/salsaflow/salsaflow/config"
	"github.com/salsaflow/salsaflow/errs"
)

// Global config ---------------------------------------------------------------

type GlobalConfig struct {
	GitHub struct {
		Token string `yaml:"token"`
	} `yaml:"github"`
}

func (global *GlobalConfig) validate() error {
	task := "Validate the global GitHub configuration"
	if global.GitHub.Token == "" {
		return errs.NewError(task, &config.ErrKeyNotSet{"github.token"}, nil)
	}
	return nil
}

// Config proxy ----------------------------------------------------------------

type Config interface {
	GitHubToken() string
}

var configCache Config

func LoadConfig() (Config, error) {
	// Try the cache first.
	if configCache != nil {
		return configCache, nil
	}

	// Create a new configProxy instance.
	proxy := &configProxy{&GlobalConfig{}}
	if err := config.UnmarshalGlobalConfig(proxy.global); err != nil {
		return nil, err
	}
	if err := proxy.global.validate(); err != nil {
		return nil, err
	}

	// Save the new instance into the cache and return.
	configCache = proxy
	return proxy, nil
}

type configProxy struct {
	global *GlobalConfig
}

func (proxy *configProxy) GitHubToken() string {
	return proxy.global.GitHub.Token
}
