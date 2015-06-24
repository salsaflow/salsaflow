package metastore

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

// Local configuration -------------------------------------------------------

type LocalConfig struct {
	MetaStore struct {
		ServerURL string `yaml:"server_url"`
	} `yaml:"metastore"`
}

func (local *LocalConfig) validate() error {
	var (
		task      = "Validate the local MetaStore configuration"
		serverURL = local.MetaStore.ServerURL
	)
	if serverURL == "" {
		return errs.NewError(task, &config.ErrKeyNotSet{"metastore.server_url"})
	}
	if _, err := url.Parse(serverURL); err != nil {
		return errs.NewError(task, &config.ErrKeyInvalid{"metastore.server_url", serverURL})
	}
	return nil
}

// Global configuration --------------------------------------------------------

type Credentials struct {
	index int

	ServerPrefix string `yaml:"server_prefix"`
	Token        string `yaml:"token"`
}

func (cred *Credentials) validate() error {
	task := "Validate selected MetaStore credentials"
	switch {
	case cred.ServerPrefix == "":
		key := fmt.Sprintf("metastore.credentials[%v].server_prefix", cred.index)
		return errs.NewError(task, &config.ErrKeyNotSet{key})
	case cred.Token == "":
		key := fmt.Sprintf("metastore.credentials[%v].token", cred.index)
		return errs.NewError(task, &config.ErrKeyNotSet{key})
	}
	return nil
}

type GlobalConfig struct {
	MetaStore struct {
		Credentials []*Credentials `yaml:"credentials"`
	} `yaml:"jira"`
}

// Proxy struct ----------------------------------------------------------------

type Config interface {
	ServerURL() *url.URL
	Token() string
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

	// Process the server URL.
	server := local.MetaStore.ServerURL
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
	creds := credentialsForServerURL(global.MetaStore.Credentials, serverURL)
	if creds == nil {
		return nil, fmt.Errorf("no MetaStore credentials found for server URL '%v'", serverURL)
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

func (proxy *configProxy) Token() string {
	return proxy.creds.Token
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
