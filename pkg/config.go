package pkg

import (
	cfg "github.com/salsita/salsaflow/config"
	"github.com/salsita/salsaflow/errs"
)

const (
	DefaultGitHubOwner = "salsita"
	DefaultGitHubRepo  = "salsaflow"
)

// Global config ---------------------------------------------------------------

type GlobalConfig struct {
	GitHub struct {
		Token string `yaml:"token"`
	} `yaml:"github"`
}

func (gc *GlobalConfig) validate() error {
	if gc.GitHub.Token == "" {
		return &cfg.ErrKeyNotSet{"github.token"}
	}
	return nil
}

// Config proxy ----------------------------------------------------------------

type config struct {
	global *GlobalConfig
}

func loadConfig() (*config, error) {
	msg := "Load global GitHub config"
	proxy := &config{&GlobalConfig{}}
	if err := cfg.FillGlobalConfig(proxy.global); err != nil {
		return nil, errs.NewError(msg, nil, err)
	}

	msg = "Validate global GitHub config"
	if err := proxy.global.validate(); err != nil {
		return nil, errs.NewError(msg, nil, err)
	}

	return proxy, nil
}

func (proxy *config) GitHubToken() string {
	return proxy.global.GitHub.Token
}
