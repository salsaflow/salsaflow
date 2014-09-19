package config

import (
	"errors"
	"net/url"
)

const sectionJira = "jira"

func mustInitJira() {
	mustInitJiraLocal()
	mustInitJiraGlobal()
}

// Global configuration --------------------------------------------------------

type jiraGlobalConfig struct {
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

func (config *jiraGlobalConfig) Validate() error {
	switch {
	case config.Username == "":
		return &ErrKeyNotSet{sectionJira + ".username"}
	case config.Password == "":
		return &ErrKeyNotSet{sectionJira + ".password"}
	}

	return nil
}

var jiraGlobalWrapper struct {
	C *jiraGlobalConfig `yaml:"jira"`
}

func mustInitJiraGlobal() {
	msg := "Parse global Jira configuration"
	if err := FillGlobalConfig(&jiraGlobalWrapper); err != nil {
		die(msg, err)
	}

	if jiraGlobalWrapper.C == nil {
		die(msg, errors.New("Jira global configuration section missing"))
	}

	if err := jiraGlobalWrapper.C.Validate(); err != nil {
		die(msg, err)
	}
}

// Local configuration -------------------------------------------------------

type jiraLocalConfig struct {
	BaseURL        string `yaml:"base_url"`
	ProjectIdOrKey string `yaml:"project_id_or_key"`
}

func (config *jiraLocalConfig) Validate() error {
	switch {
	case config.BaseURL == "":
		return &ErrKeyNotSet{sectionJira + ".base_url"}
	case config.ProjectIdOrKey == "":
		return &ErrKeyNotSet{sectionJira + ".project_id_or_key"}
	}

	if _, err := url.Parse(config.BaseURL); err != nil {
		return &ErrKeyInvalid{sectionJira + ".base_url", config.BaseURL}
	}

	return nil
}

var jiraLocalWrapper struct {
	C *jiraLocalConfig `yaml:"jira"`
}

func mustInitJiraLocal() {
	msg := "Parse local Jira configuration"
	if err := FillLocalConfig(&jiraLocalWrapper); err != nil {
		die(msg, err)
	}

	if jiraLocalWrapper.C == nil {
		die(msg, errors.New("Jira local configuration section missing"))
	}

	if err := jiraLocalWrapper.C.Validate(); err != nil {
		die(msg, err)
	}
}

// Config proxy object ---------------------------------------------------------

var Jira JiraConfig

type JiraConfig struct{}

/*
 * Global config
 */

func (c *JiraConfig) Username() string {
	return jiraGlobalWrapper.C.Username
}

func (c *JiraConfig) Password() string {
	return jiraGlobalWrapper.C.Password
}

/*
 * Local config
 */

func (c *JiraConfig) BaseURL() string {
	return jiraLocalWrapper.C.BaseURL
}

func (c *JiraConfig) ProjectIdOrKey() string {
	return jiraLocalWrapper.C.ProjectIdOrKey
}
