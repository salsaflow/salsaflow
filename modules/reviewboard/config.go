package reviewboard

import (
	// Stdlib
	"net/url"

	// Internal
	"github.com/salsaflow/salsaflow/config"
	"github.com/salsaflow/salsaflow/config/loader"
	"github.com/salsaflow/salsaflow/prompt"
)

// Configuration ===============================================================

type moduleConfig struct {
	ServerURL *url.URL
}

func loadConfig() (*moduleConfig, error) {
	spec := newConfigSpec()
	if err := loader.LoadConfig(spec); err != nil {
		return nil, err
	}

	serverURL, _ := url.Parse(spec.local.ServerURL)
	return &moduleConfig{serverURL}, nil
}

// Configuration spec ----------------------------------------------------------

type configSpec struct {
	local *LocalConfig
}

func newConfigSpec() *configSpec {
	return &configSpec{}
}

func (spec *configSpec) ConfigKey() string {
	return ModuleId
}

func (spec *configSpec) ModuleKind() loader.ModuleKind {
	return ModuleKind
}

func (spec *configSpec) GlobalConfig() loader.ConfigContainer {
	return nil
}

func (spec *configSpec) LocalConfig() loader.ConfigContainer {
	spec.local = &LocalConfig{}
	return spec.local
}

// Local configuration ---------------------------------------------------------

type LocalConfig struct {
	ServerURL string `prompt:"server URL of the Review Board server" json:"server_url"`
}

// PromptUserForConfig is a part of loader.ConfigContainer interface.
func (local *LocalConfig) PromptUserForConfig() error {
	var c LocalConfig
	err := prompt.Dialog(&c, "Insert the")
	if err != nil {
		return err
	}

	*local = c
	return nil
}

// Validate implements loader.Validator interface.
func (local *LocalConfig) Validate(sectionPath string) error {
	_, err := url.Parse(local.ServerURL)
	if err != nil {
		return &config.ErrKeyInvalid{sectionPath + ".server_url", local.ServerURL}
	}
	return nil
}
