package jira

import (
	// Stdlib
	"net/url"
	"strings"

	// Internal
	"github.com/salsita/salsaflow/config"
	"github.com/salsita/salsaflow/errs"
)

const Id = "jira"

// Local configuration -------------------------------------------------------

type LocalConfig struct {
	JIRA struct {
		BaseURL    string `yaml:"base_url"`
		ProjectKey string `yaml:"project_key"`
	} `yaml:"jira"`
}

func (local *LocalConfig) validate() error {
	var (
		task = "Validate the local JIRA configuration"
		jr   = &local.JIRA
	)
	switch {
	case jr.BaseURL == "":
		return errs.NewError(task, &config.ErrKeyNotSet{Id + ".base_url"}, nil)
	case jr.ProjectKey == "":
		return errs.NewError(task, &config.ErrKeyNotSet{Id + ".project_key"}, nil)
	}

	if _, err := url.Parse(jr.BaseURL); err != nil {
		return errs.NewError(task, &config.ErrKeyInvalid{Id + ".base_url", jr.BaseURL}, nil)
	}

	return nil
}

// Global configuration --------------------------------------------------------

type GlobalConfig struct {
	JIRA struct {
		Username string `yaml:"username"`
		Password string `yaml:"password"`
	} `yaml:"jira"`
}

func (global *GlobalConfig) validate() error {
	var (
		task = "Validate the global JIRA configuration"
		jr   = &global.JIRA
	)
	switch {
	case jr.Username == "":
		return errs.NewError(task, &config.ErrKeyNotSet{Id + ".username"}, nil)
	case jr.Password == "":
		return errs.NewError(task, &config.ErrKeyNotSet{Id + ".password"}, nil)
	}
	return nil
}

// Proxy struct ----------------------------------------------------------------

type Config interface {
	BaseURL() *url.URL
	Username() string
	Password() string
	ProjectKey() string
}

var configCache Config

func LoadConfig() (Config, error) {
	// Try the cache first.
	if configCache != nil {
		return configCache, nil
	}

	// Create a new configProxy instance.
	proxy := &configProxy{
		local:  &LocalConfig{},
		global: &GlobalConfig{},
	}

	// Load local config.
	local := proxy.local
	if err := config.UnmarshalLocalConfig(local); err != nil {
		return nil, err
	}
	if err := local.validate(); err != nil {
		return nil, err
	}

	base := local.JIRA.BaseURL
	if !strings.HasSuffix(base, "/") {
		base += "/"
	}
	// This cannot really fail since we check this in the validation function.
	baseURL, _ := url.Parse(base)
	proxy.baseURL = baseURL

	// Load global config.
	global := proxy.global
	if err := config.UnmarshalGlobalConfig(global); err != nil {
		return nil, err
	}
	if err := global.validate(); err != nil {
		return nil, err
	}

	// Save the new instance into the cache and return.
	configCache = proxy
	return proxy, nil
}

type configProxy struct {
	local  *LocalConfig
	global *GlobalConfig

	baseURL *url.URL
}

func (proxy *configProxy) Username() string {
	return proxy.global.JIRA.Username
}

func (proxy *configProxy) Password() string {
	return proxy.global.JIRA.Password
}

func (proxy *configProxy) BaseURL() *url.URL {
	return proxy.baseURL
}

func (proxy *configProxy) ProjectKey() string {
	return proxy.local.JIRA.ProjectKey
}
