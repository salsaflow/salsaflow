package jira

import (
	// Stdlib
	"fmt"
	"net/url"
	"path"
	"strings"

	// Internal
	"github.com/salsita/salsaflow/config"
	"github.com/salsita/salsaflow/errs"
)

const Id = "jira"

// Local configuration -------------------------------------------------------

type LocalConfig struct {
	JIRA struct {
		BaseURL    string `yaml:"server_url"`
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
		return errs.NewError(task, &config.ErrKeyNotSet{Id + ".server_url"}, nil)
	case jr.ProjectKey == "":
		return errs.NewError(task, &config.ErrKeyNotSet{Id + ".project_key"}, nil)
	}

	if _, err := url.Parse(jr.BaseURL); err != nil {
		return errs.NewError(task, &config.ErrKeyInvalid{Id + ".server_url", jr.BaseURL}, nil)
	}

	return nil
}

// Global configuration --------------------------------------------------------

type Credentials struct {
	Base     string `yaml:"server_prefix"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

type GlobalConfig struct {
	JIRA struct {
		Credentials []*Credentials `yaml:"credentials"`
	} `yaml:"jira"`
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

	var proxy configProxy

	// Load local config.
	var local LocalConfig
	if err := config.UnmarshalLocalConfig(&local); err != nil {
		return nil, err
	}
	if err := local.validate(); err != nil {
		return nil, err
	}

	// Process the project key.
	proxy.projectKey = local.JIRA.ProjectKey

	// Process the base URL.
	base := local.JIRA.BaseURL
	if !strings.HasSuffix(base, "/") {
		base += "/"
	}
	baseURL, err := url.Parse(base)
	if err != nil {
		// Already checked during validation,
		// so let's just explode on error.
		panic(err)
	}
	proxy.baseURL = baseURL

	// Load global config.
	var global GlobalConfig
	if err := config.UnmarshalGlobalConfig(&global); err != nil {
		return nil, err
	}

	// Process the credentials.
	creds := credentialsForBaseURL(global.JIRA.Credentials, baseURL)
	if creds == nil {
		return nil, fmt.Errorf("no JIRA credentials found for base URL '%v'", baseURL)
	}
	proxy.creds = creds

	// Save the new instance into the cache and return.
	configCache = &proxy
	return configCache, nil
}

type configProxy struct {
	baseURL    *url.URL
	creds      *Credentials
	projectKey string
}

func (proxy *configProxy) BaseURL() *url.URL {
	return proxy.baseURL
}

func (proxy *configProxy) Username() string {
	return proxy.creds.Username
}

func (proxy *configProxy) Password() string {
	return proxy.creds.Password
}

func (proxy *configProxy) ProjectKey() string {
	return proxy.projectKey
}

// credentialsForBaseURL finds the credentials matching the given base URL the best,
// i.e. the associated base prefix is the longest available.
func credentialsForBaseURL(credList []*Credentials, base *url.URL) *Credentials {
	var (
		longestMatch *Credentials
		prefix       = path.Join(base.Host, base.Path)
	)
	for _, cred := range credList {
		// Drop the scheme.
		credBaseURL, err := url.Parse(cred.Base)
		if err != nil {
			continue
		}
		credPrefix := path.Join(credBaseURL.Host, credBaseURL.Path)

		// Continue if the prefixes do not match at all.
		if !strings.HasPrefix(credPrefix, prefix) {
			continue
		}
		// Replace only if the current base prefix is longer.
		if longestMatch == nil || len(longestMatch.Base) < len(cred.Base) {
			longestMatch = cred
		}
	}
	return longestMatch
}
