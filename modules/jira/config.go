package jira

import (
	// Stdlib
	"fmt"
	"net/url"
	"path"
	"strings"

	// Internal
	"github.com/salsaflow/salsaflow/config"
	"github.com/salsaflow/salsaflow/errs"
)

const Id = "jira"

// Local configuration -------------------------------------------------------

type LocalConfig struct {
	JIRA struct {
		ServerURL  string `yaml:"server_url"`
		ProjectKey string `yaml:"project_key"`
	} `yaml:"jira"`
}

func (local *LocalConfig) validate() error {
	var (
		task = "Validate the local JIRA configuration"
		jr   = &local.JIRA
	)
	switch {
	case jr.ServerURL == "":
		return errs.NewError(task, &config.ErrKeyNotSet{Id + ".server_url"})
	case jr.ProjectKey == "":
		return errs.NewError(task, &config.ErrKeyNotSet{Id + ".project_key"})
	}

	if _, err := url.Parse(jr.ServerURL); err != nil {
		return errs.NewError(task, &config.ErrKeyInvalid{Id + ".server_url", jr.ServerURL})
	}

	return nil
}

// Global configuration --------------------------------------------------------

type Credentials struct {
	index int

	ServerPrefix string `yaml:"server_prefix"`
	Username     string `yaml:"username"`
	Password     string `yaml:"password"`
}

func (cred *Credentials) validate() error {
	task := "Validate selected JIRA credentials"
	switch {
	case cred.ServerPrefix == "":
		key := fmt.Sprintf("%v.credentials[%v].server_prefix", Id, cred.index)
		return errs.NewError(task, &config.ErrKeyNotSet{key})
	case cred.Username == "":
		key := fmt.Sprintf("%v.credentials[%v].username", Id, cred.index)
		return errs.NewError(task, &config.ErrKeyNotSet{key})
	case cred.Password == "":
		key := fmt.Sprintf("%v.credentials[%v].password", Id, cred.index)
		return errs.NewError(task, &config.ErrKeyNotSet{key})
	}
	return nil
}

type GlobalConfig struct {
	JIRA struct {
		Credentials []*Credentials `yaml:"credentials"`
	} `yaml:"jira"`
}

// Proxy struct ----------------------------------------------------------------

type Config interface {
	ServerURL() *url.URL
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

	// Process the server URL.
	server := local.JIRA.ServerURL
	if !strings.HasSuffix(server, "/") {
		server += "/"
	}
	serverURL, err := url.Parse(server)
	if err != nil {
		// Already checked during validation,
		// so let's just explode on error.
		panic(err)
	}
	proxy.serverURL = serverURL

	// Load global config.
	var global GlobalConfig
	if err := config.UnmarshalGlobalConfig(&global); err != nil {
		return nil, err
	}

	// Process the credentials.
	creds := credentialsForServerURL(global.JIRA.Credentials, serverURL)
	if creds == nil {
		return nil, fmt.Errorf("no JIRA credentials found for server URL '%v'", serverURL)
	}
	if err := creds.validate(); err != nil {
		return nil, err
	}
	proxy.creds = creds

	// Save the new instance into the cache and return.
	configCache = &proxy
	return configCache, nil
}

type configProxy struct {
	serverURL  *url.URL
	creds      *Credentials
	projectKey string
}

func (proxy *configProxy) ServerURL() *url.URL {
	return proxy.serverURL
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

// credentialsForServerURL finds the credentials matching the given server URL the best,
// i.e. the associated server prefix is the longest available.
func credentialsForServerURL(credList []*Credentials, serverURL *url.URL) *Credentials {
	var (
		longestMatch *Credentials
		prefix       = path.Join(serverURL.Host, serverURL.Path)
	)
	for i, cred := range credList {
		cred.index = i

		// Drop the scheme.
		credServerURL, err := url.Parse(cred.ServerPrefix)
		if err != nil {
			continue
		}
		credPrefix := path.Join(credServerURL.Host, credServerURL.Path)

		// Continue if the prefixes do not match at all.
		if !strings.HasPrefix(credPrefix, prefix) {
			continue
		}
		// Replace only if the current server prefix is longer.
		if longestMatch == nil || len(longestMatch.ServerPrefix) < len(cred.ServerPrefix) {
			longestMatch = cred
		}
	}
	return longestMatch
}
