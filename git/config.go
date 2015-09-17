package git

import (
	// Internal
	"github.com/salsaflow/salsaflow/config/loader"
	"github.com/salsaflow/salsaflow/errs"
	"github.com/salsaflow/salsaflow/prompt"
)

func init() {
	loader.RegisterBootstrapConfigSpec(newConfigSpec())
}

// Configuration ===============================================================

const ConfigKey = "salsaflow.core.git"

const DefaultRemoteName = "origin"

// Config holds the complete Git-related config needed by SalsaFlow.
type Config struct {
	LocalConfig
	RemoteName string
}

// LoadConfig can be used to load Git-related configuration for SalsaFlow.
func LoadConfig() (*Config, error) {
	// Load the configuration according to the spec.
	task := "Load Git-related configuration"
	spec := newConfigSpec()
	if err := loader.LoadConfig(spec); err != nil {
		return nil, errs.NewError(task, err)
	}

	// Get the remote name, which may be stored in git config.
	task = "Get the remote name for the main project repository"
	remoteName, err := GetConfigString(GitConfigKeyRemote)
	if err != nil {
		return nil, errs.NewError(task, err)
	}
	if remoteName == "" {
		remoteName = DefaultRemoteName
	}

	// Return the main config struct.
	return &Config{
		LocalConfig: *spec.local,
		RemoteName:  remoteName,
	}, nil
}

// Configuration spec ----------------------------------------------------------

func newConfigSpec() *configSpec {
	return &configSpec{}
}

// configSpec implements loader.ConfigSpec inteface.
type configSpec struct {
	local *LocalConfig
}

// ConfigKey is a part of loader.ConfigSpec interface.
func (spec *configSpec) ConfigKey() string {
	return ConfigKey
}

// GlobalConfig is a part of loader.ConfigSpec inteface.
func (spec *configSpec) GlobalConfig() loader.ConfigContainer {
	return nil
}

// LocalConfig is a part of loader.ConfigSpec interface.
func (spec *configSpec) LocalConfig() loader.ConfigContainer {
	spec.local = &LocalConfig{}
	return spec.local
}

// Local config ----------------------------------------------------------------

const GitConfigKeyRemote = "salsaflow.remote"

// LocalConfig implements loader.ConfigContainer interface.
type LocalConfig struct {
	TrunkBranchName   string `prompt:"trunk branch name"   default:"develop" json:"trunk_branch"`
	ReleaseBranchName string `prompt:"release branch name" default:"release" json:"release_branch"`
	StagingBranchName string `prompt:"staging branch name" default:"stage"   json:"staging_branch"`
	StableBranchName  string `prompt:"stable branch name"  default:"master"  json:"stable_branch"`
}

// PromptUserForConfig is a part of loader.ConfigContainer interface.
func (local *LocalConfig) PromptUserForConfig() error {
	task := "Prompt the user for local Git-related configuration"

	var c LocalConfig
	if err := prompt.Dialog(&c, "Insert the"); err != nil {
		return errs.NewError(task, err)
	}

	*local = c
	return nil
}
