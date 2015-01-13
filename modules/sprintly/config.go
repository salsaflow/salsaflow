package sprintly

import (
	// Internal
	"github.com/salsaflow/salsaflow/config"
	"github.com/salsaflow/salsaflow/errs"
)

const (
	Id = "sprintly"

	DefaultNoReviewLabel = "no review"
	DefaultReviewedLabel = "reviewed"
)

var DefaultSkipCheckLabels = []string{"dupe", "wontfix"}

// Local configuration ---------------------------------------------------------

type LocalConfig struct {
	Sprintly struct {
		ProductId int `yaml:"product_id"`
		Labels    struct {
			NoReviewLabel   string   `yaml:"no_review"`
			ReviewedLabel   string   `yaml:"reviewed"`
			SkipCheckLabels []string `yaml:"skip_check_labels"`
		} `yaml:"labels"`
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
	case sprintly.Labels.NoReviewLabel == "":
		err = &config.ErrKeyNotSet{Id + ".labels.no_review"}
	case sprintly.Labels.ReviewedLabel == "":
		err = &config.ErrKeyNotSet{Id + ".labels.reviewed"}
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
	NoReviewLabel() string
	ReviewedLabel() string
	SkipCheckLabels() []string
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

	labels := &local.Sprintly.Labels
	if labels.NoReviewLabel == "" {
		labels.NoReviewLabel = DefaultNoReviewLabel
	}
	if labels.ReviewedLabel == "" {
		labels.ReviewedLabel = DefaultReviewedLabel
	}
	labels.SkipCheckLabels = append(labels.SkipCheckLabels, DefaultSkipCheckLabels...)

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

func (proxy *configProxy) NoReviewLabel() string {
	return proxy.local.Sprintly.Labels.NoReviewLabel
}

func (proxy *configProxy) ReviewedLabel() string {
	return proxy.local.Sprintly.Labels.ReviewedLabel
}

func (proxy *configProxy) SkipCheckLabels() []string {
	return proxy.local.Sprintly.Labels.SkipCheckLabels
}

func (proxy *configProxy) Username() string {
	return proxy.global.Sprintly.Username
}

func (proxy *configProxy) Token() string {
	return proxy.global.Sprintly.Token
}
