package github

import (
	"github.com/salsaflow/salsaflow/config"
	"github.com/salsaflow/salsaflow/errs"
)

const (
	DefaultReviewIssueLabel      = "review"
	DefaultStoryImplementedLabel = "implemented"
)

// Local configuration -------------------------------------------------------

type LocalConfig struct {
	GitHub struct {
		ReviewLabel           string `yaml:"review_issue_label"`
		StoryImplementedLabel string `yaml:"story_implemented_label"`
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
		return errs.NewError(task, &config.ErrKeyNotSet{"github.token"})
	}
	return nil
}

// Config proxy ----------------------------------------------------------------

type Config interface {
	ReviewLabel() string
	StoryImplementedLabel() string
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
	if local.GitHub.StoryImplementedLabel == "" {
		local.GitHub.StoryImplementedLabel = DefaultStoryImplementedLabel
	}

	// Save the new instance into the cache and return.
	configCache = &configProxy{
		apiToken:              global.GitHub.Token,
		reviewIssueLabel:      local.GitHub.ReviewLabel,
		storyImplementedLabel: local.GitHub.StoryImplementedLabel,
	}
	return configCache, nil
}

type configProxy struct {
	apiToken              string
	reviewIssueLabel      string
	storyImplementedLabel string
}

func (proxy *configProxy) Token() string {
	return proxy.apiToken
}

func (proxy *configProxy) ReviewLabel() string {
	return proxy.reviewIssueLabel
}

func (proxy *configProxy) StoryImplementedLabel() string {
	return proxy.storyImplementedLabel
}
