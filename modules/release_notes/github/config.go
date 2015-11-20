package github

import (
	// Internal
	"github.com/salsaflow/salsaflow/config/loader"
	"github.com/salsaflow/salsaflow/prompt"
)

// Configuration ===============================================================

type moduleConfig struct {
	*GlobalConfig
}

func loadConfig() (*moduleConfig, error) {
	// Load the config.
	spec := newConfigSpec()
	if err := loader.LoadConfig(spec); err != nil {
		return nil, err
	}

	// Assemble the config object.
	return &moduleConfig{spec.global}, nil
}

// Configuration spec ----------------------------------------------------------

type configSpec struct {
	global *GlobalConfig
}

func newConfigSpec() *configSpec {
	return &configSpec{}
}

// ConfigKey is a part of loader.ConfigSpec
func (spec *configSpec) ConfigKey() string {
	return ModuleId
}

// ModuleKind is a part of loader.ModuleConfigSpec
func (spec *configSpec) ModuleKind() loader.ModuleKind {
	return ModuleKind
}

// GlobalConfig is a part of loader.ConfigSpec
func (spec *configSpec) GlobalConfig() loader.ConfigContainer {
	spec.global = &GlobalConfig{}
	return spec.global
}

// LocalConfig is a part of loader.ConfigSpec
func (spec *configSpec) LocalConfig() loader.ConfigContainer {
	return nil
}

// Global configuration --------------------------------------------------------

type GlobalConfig struct {
	Token string `prompt:"GitHub token to be used for posting release notes" secret:"true" json:"token"`
}

// PromptUserForConfig is a part of loader.ConfigContainer
func (global *GlobalConfig) PromptUserForConfig() error {
	var c GlobalConfig
	if err := prompt.Dialog(&c, "Insert the"); err != nil {
		return err
	}

	*global = c
	return nil
}
