package pkg

import (
	// Internal
	"github.com/salsaflow/salsaflow/config/loader"
	"github.com/salsaflow/salsaflow/errs"
	"github.com/salsaflow/salsaflow/github"
	"github.com/salsaflow/salsaflow/prompt"

	// Vendor
	"github.com/bgentry/speakeasy"
	gh "github.com/google/go-github/github"
)

func newGitHubClient() (*gh.Client, error) {
	task := "Instantiate a GitHub API client"

	// Get the access token.
	spec := newConfigSpec()
	if err := loader.LoadConfig(spec); err != nil {
		return nil, errs.NewError(task, err)
	}

	// Return a new API client.
	return github.NewClient(spec.global.GitHubToken), nil
}

// Configuration ===============================================================

const ConfigKey = "salsaflow.core.updater"

// Configuration spec ----------------------------------------------------------

func newConfigSpec() *configSpec {
	return &configSpec{}
}

type configSpec struct {
	global *GlobalConfig
}

func (spec *configSpec) ConfigKey() string {
	return ConfigKey
}

func (spec *configSpec) GlobalConfig() loader.ConfigContainer {
	spec.global = &GlobalConfig{}
	return spec.global
}

func (spec *configSpec) LocalConfig() loader.ConfigContainer {
	return nil
}

// Global config ---------------------------------------------------------------

type GlobalConfig struct {
	GitHubToken string `json:"github_token"`
}

func (global *GlobalConfig) PromptUserForConfig() error {
	token, err := speakeasy.Ask("Insert the GitHub token to be used for SalsaFlow updater: ")
	if token == "" {
		prompt.PanicCancel()
	}
	if err != nil {
		return err
	}

	global.GitHubToken = token
	return nil
}
