package reviewboard

import (
	// Stdlib
	"net/url"
	"strings"

	// Internal
	"github.com/salsaflow/salsaflow/config"
	"github.com/salsaflow/salsaflow/errs"
)

const Id = "review_board"

// Local configuration -------------------------------------------------------

type LocalConfig struct {
	RB struct {
		ServerURL string `yaml:"server_url"`
	} `yaml:"review_board"`
}

func (local LocalConfig) validate() error {
	var (
		task      = "Validate the local Review Board configuration"
		serverURL = local.RB.ServerURL
	)
	switch {
	case serverURL == "":
		return errs.NewError(task, &config.ErrKeyNotSet{Id + ".server_url"}, nil)
	}

	if _, err := url.Parse(serverURL); err != nil {
		return errs.NewError(task, &config.ErrKeyInvalid{Id + ".server_url", serverURL}, nil)
	}

	return nil
}

// Proxy struct ----------------------------------------------------------------

type Config interface {
	ServerURL() *url.URL
}

var configCache Config

func LoadConfig() (Config, error) {
	// Try the cache first.
	if configCache != nil {
		return configCache, nil
	}

	// Load local config.
	var (
		proxy configProxy
		local LocalConfig
	)
	if err := config.UnmarshalLocalConfig(&local); err != nil {
		return nil, err
	}
	if err := local.validate(); err != nil {
		return nil, err
	}

	server := local.RB.ServerURL
	if !strings.HasSuffix(server, "/") {
		server += "/"
	}
	// This cannot really fail since we check this in the validation function.
	// So in case this actually fails, panic to signal there is something wrong.
	serverURL, err := url.Parse(server)
	if err != nil {
		panic(err)
	}
	proxy.serverURL = serverURL

	// Save the new instance into the cache and return.
	configCache = &proxy
	return configCache, nil
}

type configProxy struct {
	serverURL *url.URL
}

func (proxy *configProxy) ServerURL() *url.URL {
	return proxy.serverURL
}
