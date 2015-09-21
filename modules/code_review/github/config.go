package github

import (
	// Internal
	"github.com/salsaflow/salsaflow/config/loader"
	"github.com/salsaflow/salsaflow/prompt"
)

// Configuration ===============================================================

type moduleConfig struct {
	Token                 string
	ReviewLabel           string
	StoryImplementedLabel string
}

func loadConfig() (*moduleConfig, error) {
	spec := newConfigSpec()
	if err := loader.LoadConfig(spec); err != nil {
		return nil, err
	}
	return &moduleConfig{
		Token:                 spec.global.Token,
		ReviewLabel:           spec.local.ReviewLabel,
		StoryImplementedLabel: spec.local.StoryImplementedLabel,
	}, nil
}

// Configuration spec ----------------------------------------------------------

type configSpec struct {
	global *GlobalConfig
	local  *LocalConfig
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
	spec.global = &GlobalConfig{}
	return spec.global
}

func (spec *configSpec) LocalConfig() loader.ConfigContainer {
	spec.local = &LocalConfig{}
	return spec.local
}

// Local configuration ---------------------------------------------------------

type LocalConfig struct {
	ReviewLabel           string `prompt:"with"           default:"review"      json:"review_issue_label"`
	StoryImplementedLabel string `prompt:"as implemented" default:"implemented" json:"story_implemented_label"`
}

// PromptUserForConfig is a part of loader.ConfigContainer interface.
func (local *LocalConfig) PromptUserForConfig() error {
	var c LocalConfig
	err := prompt.Dialog(&c, "Insert the label to be used to mark the GitHub review issues")
	if err != nil {
		return err
	}

	*local = c
	return nil
}

// Global configuration --------------------------------------------------------

type GlobalConfig struct {
	Token string `prompt:"token to be used for creating GitHub review issues" secret:"true" json:"token"`
}

// PromptUserForConfig is a part of loader.ConfigContainer interface.
func (global *GlobalConfig) PromptUserForConfig() error {
	var c GlobalConfig
	if err := prompt.Dialog(&c, "Insert the"); err != nil {
		return err
	}

	*global = c
	return nil
}
