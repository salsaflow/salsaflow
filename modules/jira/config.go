package jira

import (
	// Stdlib
	"errors"
	"net/url"
	"strings"

	// Internal
	cfg "github.com/salsita/salsaflow/config"
	"github.com/salsita/salsaflow/log"
)

const (
	Id = "jira"
)

func loadConfig() error {
	if err := loadGlobalConfig(); err != nil {
		return err
	}
	if err := loadLocalConfig(); err != nil {
		return err
	}
	config = mustNewJiraConfig()
	return nil
}

// Global configuration --------------------------------------------------------

type globalConfig struct {
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

func (config *globalConfig) Validate() error {
	switch {
	case config.Username == "":
		return &cfg.ErrKeyNotSet{Id + ".username"}
	case config.Password == "":
		return &cfg.ErrKeyNotSet{Id + ".password"}
	}
	return nil
}

var globalWrapper struct {
	C *globalConfig `yaml:"jira"`
}

func loadGlobalConfig() error {
	msg := "Load global Jira configuration"
	if err := cfg.FillGlobalConfig(&globalWrapper); err != nil {
		log.Fail(msg)
		return err
	}

	if globalWrapper.C == nil {
		log.Fail(msg)
		return errors.New("Jira global configuration section missing")
	}

	if err := globalWrapper.C.Validate(); err != nil {
		log.Fail(msg)
		return err
	}

	return nil
}

// Local configuration -------------------------------------------------------

type localConfig struct {
	BaseURL        string `yaml:"base_url"`
	ProjectIdOrKey string `yaml:"project_id_or_key"`
}

func (config *localConfig) Validate() error {
	switch {
	case config.BaseURL == "":
		return &cfg.ErrKeyNotSet{Id + ".base_url"}
	case config.ProjectIdOrKey == "":
		return &cfg.ErrKeyNotSet{Id + ".project_id_or_key"}
	}

	if _, err := url.Parse(config.BaseURL); err != nil {
		return &cfg.ErrKeyInvalid{Id + ".base_url", config.BaseURL}
	}

	return nil
}

var localWrapper struct {
	C *localConfig `yaml:"jira"`
}

func loadLocalConfig() error {
	msg := "Load local Jira configuration"
	if err := cfg.FillLocalConfig(&localWrapper); err != nil {
		log.Fail(msg)
		return err
	}

	if localWrapper.C == nil {
		log.Fail(msg)
		return errors.New("Jira local configuration section missing")
	}

	if err := localWrapper.C.Validate(); err != nil {
		log.Fail(msg)
		return err
	}

	return nil
}

// Config proxy object ---------------------------------------------------------

var config *jiraConfig

type jiraConfig struct {
	baseURL *url.URL
}

func mustNewJiraConfig() *jiraConfig {
	// Make sure the URL is absolute.
	base := localWrapper.C.BaseURL
	if !strings.HasSuffix(base, "/") {
		base += "/"
	}
	baseURL, err := url.Parse(base)
	if err != nil {
		panic(err)
	}
	return &jiraConfig{baseURL}
}

/*
 * Global config
 */

func (c *jiraConfig) Username() string {
	return globalWrapper.C.Username
}

func (c *jiraConfig) Password() string {
	return globalWrapper.C.Password
}

/*
 * Local config
 */

func (c *jiraConfig) BaseURL() *url.URL {
	return c.baseURL
}

func (c *jiraConfig) ProjectIdOrKey() string {
	return localWrapper.C.ProjectIdOrKey
}
