package github

import (
	"github.com/salsaflow/salsaflow/config"
	"github.com/salsaflow/salsaflow/errs"
)

const DefaultReviewIssueLabel = "review"

// Local configuration -------------------------------------------------------

type LocalConfig struct {
	GitHub struct {
		ReviewLabel string `yaml:"review_issue_label"`
	} `yaml:"github"`
}

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
	ReviewLabel() string
	Token() string
}

var configCache Config

func LoadConfig() (Config, error) {
	// Try the cache first.
	if configCache != nil {
		return configCache, nil
	}

	// Unmarshal.
	var global GlobalConfig
	if err := config.UnmarshalGlobalConfig(&global); err != nil {
		return nil, err
	}
	if err := global.validate(); err != nil {
		return nil, err
	}

	var local LocalConfig
	if err := config.UnmarshalLocalConfig(&local); err != nil {
		return nil, err
	}
	if local.GitHub.ReviewLabel == "" {
		local.GitHub.ReviewLabel = DefaultReviewIssueLabel
	}

	// Save the new instance into the cache and return.
	configCache = &configProxy{
		apiToken:         global.GitHub.Token,
		reviewIssueLabel: local.GitHub.ReviewLabel,
	}
	return configCache, nil
}

type configProxy struct {
	apiToken         string
	reviewIssueLabel string
}

func (proxy *configProxy) Token() string {
	return proxy.apiToken
}

func (proxy *configProxy) ReviewLabel() string {
	return proxy.reviewIssueLabel
}
