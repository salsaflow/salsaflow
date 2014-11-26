package common

import (
	"github.com/salsaflow/salsaflow/config"
	"github.com/salsaflow/salsaflow/errs"
)

// Local config ----------------------------------------------------------------

type LocalConfig struct {
	IssueTrackerId   string `yaml:"issue_tracker"`
	CodeReviewToolId string `yaml:"code_review_tool"`
}

func (local *LocalConfig) validate() error {
	task := "Validate the local modules config"
	switch {
	case local.IssueTrackerId == "":
		return errs.NewError(task, &config.ErrKeyNotSet{"issue_tracker"}, nil)
	case local.CodeReviewToolId == "":
		return errs.NewError(task, &config.ErrKeyNotSet{"code_review_tool"}, nil)
	}
	return nil
}

// Configuration proxy ---------------------------------------------------------

type Config interface {
	IssueTrackerId() string
	CodeReviewToolId() string
}

var configCache Config

// LoadConfig returns the config that is useful for modules in general.
//
// Even when an error is returned, Config can be partially filled in
// in case the error being returned is a validation error.
func LoadConfig() (Config, error) {
	// Parse the config files if necessary.
	var err error
	if configCache == nil {
		task := "Load modules-related SalsaFlow config"
		proxy := &configProxy{&LocalConfig{}}
		if err := config.UnmarshalLocalConfig(proxy.local); err != nil {
			return nil, errs.NewError(task, err, nil)
		}
		configCache = proxy
		err = proxy.local.validate()
	}

	// Return the config as saved in the cache.
	return configCache, err
}

type configProxy struct {
	local *LocalConfig
}

func (proxy *configProxy) IssueTrackerId() string {
	return proxy.local.IssueTrackerId
}

func (proxy *configProxy) CodeReviewToolId() string {
	return proxy.local.CodeReviewToolId
}
