package sprintly

import (
	// Internal
	"github.com/salsaflow/salsaflow/config"
	"github.com/salsaflow/salsaflow/errs"
)

const (
	Id = "sprintly"

	DefaultNoReviewTag = "no review"
	DefaultReviewedTag = "reviewed"

	DefaultStagingEnvironment    = "staging"
	DefaultProductionEnvironment = "production"
)

// Local configuration ---------------------------------------------------------

type LocalConfig struct {
	Sprintly struct {
		ProductId int `yaml:"product_id"`
		Tags      struct {
			NoReviewTag string `yaml:"no_review"`
			ReviewedTag string `yaml:"reviewed"`
		} `yaml:"tags"`
		Environments struct {
			Staging    string `yaml:"staging"`
			Production string `yaml:"production"`
		} `yaml:"environments"`
	} `yaml:"sprintly"`
}

func (local *LocalConfig) validate() error {
	var (
		task     = "Validate the local Sprintly configuration"
		sprintly = &local.Sprintly
		err      error
	)
	switch {
	case sprintly.ProductId == 0:
		err = &config.ErrKeyNotSet{Id + ".product_id"}
	case sprintly.Tags.NoReviewTag == "":
		err = &config.ErrKeyNotSet{Id + ".tags.no_review"}
	case sprintly.Tags.ReviewedTag == "":
		err = &config.ErrKeyNotSet{Id + ".tags.reviewed"}
	case sprintly.Environments.Staging == "":
		err = &config.ErrKeyNotSet{Id + ".environments.staging"}
	case sprintly.Environments.Production == "":
		err = &config.ErrKeyNotSet{Id + ".environments.production"}
	}
	if err != nil {
		return errs.NewError(task, err, nil)
	}
	return nil
}

// Global configuration --------------------------------------------------------

type GlobalConfig struct {
	Sprintly struct {
		Username string `yaml:"username"`
		Token    string `yaml:"token"`
	} `yaml:"sprintly"`
}

func (global *GlobalConfig) validate() error {
	var (
		task     = "Validate the global Sprintly configuration"
		sprintly = &global.Sprintly
		err      error
	)
	switch {
	case sprintly.Username == "":
		err = &config.ErrKeyNotSet{Id + ".username"}
	case sprintly.Token == "":
		err = &config.ErrKeyNotSet{Id + ".token"}
	}
	if err != nil {
		return errs.NewError(task, err, nil)
	}
	return nil
}

// Proxy object ----------------------------------------------------------------

type Config interface {
	ProductId() int
	NoReviewTag() string
	ReviewedTag() string
	StagingEnvironment() string
	ProductionEnvironment() string
	Username() string
	Token() string
}

var configCache Config

func LoadConfig() (Config, error) {
	// Try the cache first.
	if configCache != nil {
		return configCache, nil
	}

	// Load the local config.
	var local LocalConfig
	if err := config.UnmarshalLocalConfig(&local); err != nil {
		return nil, err
	}

	tags := &local.Sprintly.Tags
	if tags.NoReviewTag == "" {
		tags.NoReviewTag = DefaultNoReviewTag
	}
	if tags.ReviewedTag == "" {
		tags.ReviewedTag = DefaultReviewedTag
	}

	envs := &local.Sprintly.Environments
	if envs.Staging == "" {
		envs.Staging = DefaultStagingEnvironment
	}
	if envs.Production == "" {
		envs.Production = DefaultProductionEnvironment
	}

	if err := local.validate(); err != nil {
		return nil, err
	}

	// Load the global config.
	var global GlobalConfig
	if err := config.UnmarshalGlobalConfig(&global); err != nil {
		return nil, err
	}
	if err := global.validate(); err != nil {
		return nil, err
	}

	configCache = &configProxy{&local, &global}
	return configCache, nil
}

type configProxy struct {
	local  *LocalConfig
	global *GlobalConfig
}

func (proxy *configProxy) ProductId() int {
	return proxy.local.Sprintly.ProductId
}

func (proxy *configProxy) NoReviewTag() string {
	return proxy.local.Sprintly.Tags.NoReviewTag
}

func (proxy *configProxy) ReviewedTag() string {
	return proxy.local.Sprintly.Tags.ReviewedTag
}

func (proxy *configProxy) StagingEnvironment() string {
	return proxy.local.Sprintly.Environments.Staging
}

func (proxy *configProxy) ProductionEnvironment() string {
	return proxy.local.Sprintly.Environments.Production
}

func (proxy *configProxy) Username() string {
	return proxy.global.Sprintly.Username
}

func (proxy *configProxy) Token() string {
	return proxy.global.Sprintly.Token
}
